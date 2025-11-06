package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// handleCreateGroup creates a new machine group
func (s *Server) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req models.CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Check if group already exists
	existing, err := s.db.GetGroupByName(req.Name)
	if err != nil {
		log.Printf("Failed to check existing group: %v", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "group already exists")
		return
	}

	// Create group
	group, err := s.db.CreateGroup(req.Name, req.Description, req.Tags)
	if err != nil {
		log.Printf("Failed to create group: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create group")
		return
	}

	log.Printf("Created group: %s", group.Name)
	respondJSON(w, http.StatusCreated, group)
}

// handleListGroups lists all groups
func (s *Server) handleListGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.db.ListGroups()
	if err != nil {
		log.Printf("Failed to list groups: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list groups")
		return
	}

	respondJSON(w, http.StatusOK, groups)
}

// handleGetGroup retrieves a single group
func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	group, err := s.db.GetGroup(id)
	if err != nil {
		log.Printf("Failed to get group: %v", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if group == nil {
		respondError(w, http.StatusNotFound, "group not found")
		return
	}

	respondJSON(w, http.StatusOK, group)
}

// handleUpdateGroup updates a group
func (s *Server) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	group, err := s.db.GetGroup(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if group == nil {
		respondError(w, http.StatusNotFound, "group not found")
		return
	}

	var req models.UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if req.Name != "" {
		group.Name = req.Name
	}
	if req.Description != "" {
		group.Description = req.Description
	}
	if req.Tags != nil {
		group.Tags = req.Tags
	}

	if err := s.db.UpdateGroup(group); err != nil {
		log.Printf("Failed to update group: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to update group")
		return
	}

	respondJSON(w, http.StatusOK, group)
}

// handleDeleteGroup deletes a group
func (s *Server) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.db.DeleteGroup(id); err != nil {
		log.Printf("Failed to delete group: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetGroupMachines retrieves all machines in a group
func (s *Server) handleGetGroupMachines(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]

	machines, err := s.db.GetGroupMachines(groupID)
	if err != nil {
		log.Printf("Failed to get group machines: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to get group machines")
		return
	}

	respondJSON(w, http.StatusOK, machines)
}

// handleAddMachineToGroup adds a machine to a group
func (s *Server) handleAddMachineToGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]
	machineID := vars["machine_id"]

	// Verify group exists
	group, err := s.db.GetGroup(groupID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if group == nil {
		respondError(w, http.StatusNotFound, "group not found")
		return
	}

	// Verify machine exists
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if machine == nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	// Add machine to group
	if err := s.db.AddMachineToGroup(groupID, machineID); err != nil {
		log.Printf("Failed to add machine to group: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to add machine to group")
		return
	}

	log.Printf("Added machine %s to group %s", machineID, groupID)
	w.WriteHeader(http.StatusNoContent)
}

// handleRemoveMachineFromGroup removes a machine from a group
func (s *Server) handleRemoveMachineFromGroup(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["id"]
	machineID := vars["machine_id"]

	if err := s.db.RemoveMachineFromGroup(groupID, machineID); err != nil {
		log.Printf("Failed to remove machine from group: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to remove machine from group")
		return
	}

	log.Printf("Removed machine %s from group %s", machineID, groupID)
	w.WriteHeader(http.StatusNoContent)
}

// handleGetMachineGroups retrieves all groups a machine belongs to
func (s *Server) handleGetMachineGroups(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	groups, err := s.db.GetMachineGroups(machineID)
	if err != nil {
		log.Printf("Failed to get machine groups: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to get machine groups")
		return
	}

	respondJSON(w, http.StatusOK, groups)
}
