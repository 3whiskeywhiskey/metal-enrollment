package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// handleSubmitMetrics handles metrics submission from machines
func (s *Server) handleSubmitMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Verify machine exists
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Parse metrics
	var metrics models.MachineMetrics
	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set machine ID and timestamp
	metrics.MachineID = machineID
	metrics.Timestamp = time.Now()

	// Save metrics
	if err := s.db.CreateMachineMetrics(&metrics); err != nil {
		http.Error(w, fmt.Sprintf("Failed to save metrics: %v", err), http.StatusInternalServerError)
		return
	}

	// Update machine last_seen_at
	now := time.Now()
	machine.LastSeenAt = &now
	if err := s.db.UpdateMachine(machine); err != nil {
		// Log but don't fail the request
		log.Printf("Failed to update machine last_seen_at: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(metrics)
}

// handleGetLatestMetrics retrieves the latest metrics for a machine
func (s *Server) handleGetLatestMetrics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Verify machine exists
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Get latest metrics
	metrics, err := s.db.GetLatestMetrics(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusInternalServerError)
		return
	}

	if metrics == nil {
		http.Error(w, "No metrics found for this machine", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleGetMetricsHistory retrieves metrics history for a machine
func (s *Server) handleGetMetricsHistory(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	// Verify machine exists
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machine: %v", err), http.StatusInternalServerError)
		return
	}
	if machine == nil {
		http.Error(w, "Machine not found", http.StatusNotFound)
		return
	}

	// Parse query parameters
	sinceStr := r.URL.Query().Get("since")
	limitStr := r.URL.Query().Get("limit")

	// Default to last 24 hours
	since := time.Now().Add(-24 * time.Hour)
	if sinceStr != "" {
		duration, err := time.ParseDuration(sinceStr)
		if err == nil {
			since = time.Now().Add(-duration)
		}
	}

	// Default limit to 1000
	limit := 1000
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get metrics history
	metrics, err := s.db.ListMetrics(machineID, since, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleGetAllMachinesMetrics retrieves latest metrics for all machines
func (s *Server) handleGetAllMachinesMetrics(w http.ResponseWriter, r *http.Request) {
	// Get all machines
	machines, err := s.db.ListMachines()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get machines: %v", err), http.StatusInternalServerError)
		return
	}

	// Get latest metrics for each machine
	type MachineWithMetrics struct {
		Machine *models.Machine        `json:"machine"`
		Metrics *models.MachineMetrics `json:"metrics,omitempty"`
	}

	result := make([]MachineWithMetrics, 0, len(machines))
	for _, machine := range machines {
		metrics, _ := s.db.GetLatestMetrics(machine.ID)
		result = append(result, MachineWithMetrics{
			Machine: machine,
			Metrics: metrics,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
