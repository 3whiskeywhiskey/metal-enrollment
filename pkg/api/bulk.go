package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
)

// handleBulkOperation handles bulk operations on machines
func (s *Server) handleBulkOperation(w http.ResponseWriter, r *http.Request) {
	var req models.BulkOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate operation type
	if req.Operation == "" {
		respondError(w, http.StatusBadRequest, "operation is required")
		return
	}

	// Get machine IDs either from the request or from a group
	var machineIDs []string
	if req.GroupID != "" {
		// Get machines from group
		machines, err := s.db.GetGroupMachines(req.GroupID)
		if err != nil {
			log.Printf("Failed to get group machines: %v", err)
			respondError(w, http.StatusInternalServerError, "failed to get group machines")
			return
		}
		for _, m := range machines {
			machineIDs = append(machineIDs, m.ID)
		}
	} else if len(req.MachineIDs) > 0 {
		machineIDs = req.MachineIDs
	} else {
		respondError(w, http.StatusBadRequest, "either machine_ids or group_id is required")
		return
	}

	if len(machineIDs) == 0 {
		respondError(w, http.StatusBadRequest, "no machines to operate on")
		return
	}

	// Execute the operation
	var result models.BulkOperationResult
	result.TotalCount = len(machineIDs)

	switch req.Operation {
	case "update":
		result = s.bulkUpdate(machineIDs, req.Data)
	case "build":
		result = s.bulkBuild(machineIDs)
	case "delete":
		result = s.bulkDelete(machineIDs)
	default:
		respondError(w, http.StatusBadRequest, "invalid operation")
		return
	}

	log.Printf("Bulk operation %s: %d/%d succeeded", req.Operation, result.SuccessCount, result.TotalCount)
	respondJSON(w, http.StatusOK, result)
}

// bulkUpdate updates multiple machines
func (s *Server) bulkUpdate(machineIDs []string, data map[string]interface{}) models.BulkOperationResult {
	result := models.BulkOperationResult{
		TotalCount: len(machineIDs),
	}

	for _, id := range machineIDs {
		machine, err := s.db.GetMachine(id)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: %v", id, err))
			continue
		}

		if machine == nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: not found", id))
			continue
		}

		// Update fields from data
		if hostname, ok := data["hostname"].(string); ok && hostname != "" {
			machine.Hostname = hostname
		}
		if description, ok := data["description"].(string); ok {
			machine.Description = description
		}
		if nixosConfig, ok := data["nixos_config"].(string); ok && nixosConfig != "" {
			machine.NixOSConfig = nixosConfig
			machine.Status = models.StatusConfigured
		}

		if err := s.db.UpdateMachine(machine); err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: %v", id, err))
			continue
		}

		result.SuccessCount++
	}

	return result
}

// bulkBuild triggers builds for multiple machines
func (s *Server) bulkBuild(machineIDs []string) models.BulkOperationResult {
	result := models.BulkOperationResult{
		TotalCount: len(machineIDs),
	}

	for _, id := range machineIDs {
		machine, err := s.db.GetMachine(id)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: %v", id, err))
			continue
		}

		if machine == nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: not found", id))
			continue
		}

		if machine.NixOSConfig == "" {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: no configuration", id))
			continue
		}

		// Create build request
		build, err := s.db.CreateBuild(machine.ID, machine.NixOSConfig)
		if err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: %v", id, err))
			continue
		}

		// Update machine status
		machine.Status = models.StatusBuilding
		machine.LastBuildID = &build.ID
		if err := s.db.UpdateMachine(machine); err != nil {
			log.Printf("Failed to update machine status: %v", err)
		}

		log.Printf("Build requested for machine %s: build_id=%s", machine.ID, build.ID)
		result.SuccessCount++
	}

	return result
}

// bulkDelete deletes multiple machines
func (s *Server) bulkDelete(machineIDs []string) models.BulkOperationResult {
	result := models.BulkOperationResult{
		TotalCount: len(machineIDs),
	}

	for _, id := range machineIDs {
		if err := s.db.DeleteMachine(id); err != nil {
			result.FailureCount++
			result.Errors = append(result.Errors, fmt.Sprintf("machine %s: %v", id, err))
			continue
		}

		result.SuccessCount++
	}

	return result
}
