package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/ipmi"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// PowerRequest represents a power control request
type PowerRequest struct {
	Operation string `json:"operation"` // on, off, reset, cycle, status
}

// handlePowerControl handles power control operations
func (s *Server) handlePowerControl(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Check if BMC is configured
	if machine.BMCInfo == nil {
		http.Error(w, "BMC is not configured for this machine", http.StatusBadRequest)
		return
	}

	// Parse request
	var req PowerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user ID from context for audit
	userID := "system"
	if user, ok := r.Context().Value("user").(*models.User); ok {
		userID = user.ID
	}

	// Create power operation record
	powerOp := &models.PowerOperation{
		MachineID:   machineID,
		Operation:   req.Operation,
		Status:      "pending",
		InitiatedBy: userID,
	}

	if err := s.db.CreatePowerOperation(powerOp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create power operation: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute power operation asynchronously
	go func() {
		controller := ipmi.NewPowerController()
		var result string
		var err error

		switch req.Operation {
		case "on":
			result, err = controller.PowerOn(machine.BMCInfo)
		case "off":
			result, err = controller.PowerOff(machine.BMCInfo)
		case "reset":
			result, err = controller.PowerReset(machine.BMCInfo)
		case "cycle":
			result, err = controller.PowerCycle(machine.BMCInfo)
		case "status":
			result, err = controller.GetPowerStatus(machine.BMCInfo)
		default:
			err = fmt.Errorf("unsupported operation: %s", req.Operation)
		}

		// Update power operation record
		now := time.Now()
		powerOp.CompletedAt = &now

		if err != nil {
			powerOp.Status = "failed"
			powerOp.Error = err.Error()
		} else {
			powerOp.Status = "success"
			powerOp.Result = result
		}

		s.db.UpdatePowerOperation(powerOp)
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(powerOp)
}

// handleGetPowerStatus gets the current power status
func (s *Server) handleGetPowerStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Check if BMC is configured
	if machine.BMCInfo == nil {
		http.Error(w, "BMC is not configured for this machine", http.StatusBadRequest)
		return
	}

	// Get power status
	controller := ipmi.NewPowerController()
	status, err := controller.GetPowerStatus(machine.BMCInfo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get power status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"machine_id": machineID,
		"status":     status,
		"timestamp":  time.Now().Format(time.RFC3339),
	})
}

// handleGetPowerOperations retrieves power operation history
func (s *Server) handleGetPowerOperations(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Get power operations
	operations, err := s.db.ListPowerOperations(machineID, 50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get power operations: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(operations)
}

// handleTestBMC tests the BMC connection
func (s *Server) handleTestBMC(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Check if BMC is configured
	if machine.BMCInfo == nil {
		http.Error(w, "BMC is not configured for this machine", http.StatusBadRequest)
		return
	}

	// Test connection
	controller := ipmi.NewPowerController()
	err = controller.TestConnection(machine.BMCInfo)

	response := map[string]interface{}{
		"machine_id": machineID,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response["status"] = "failed"
		response["error"] = err.Error()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	response["status"] = "success"
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetBMCInfo retrieves BMC information
func (s *Server) handleGetBMCInfo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Check if BMC is configured
	if machine.BMCInfo == nil {
		http.Error(w, "BMC is not configured for this machine", http.StatusBadRequest)
		return
	}

	// Get BMC info
	controller := ipmi.NewPowerController()
	info, err := controller.GetBMCInfo(machine.BMCInfo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get BMC info: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// handleGetSensors retrieves sensor readings from BMC
func (s *Server) handleGetSensors(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Check if BMC is configured
	if machine.BMCInfo == nil {
		http.Error(w, "BMC is not configured for this machine", http.StatusBadRequest)
		return
	}

	// Get sensor readings
	controller := ipmi.NewPowerController()
	sensors, err := controller.GetSensorReadings(machine.BMCInfo)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get sensor readings: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sensors)
}
