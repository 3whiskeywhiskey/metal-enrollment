package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	db     *database.DB
	router *mux.Router
	config Config
}

// Config holds server configuration
type Config struct {
	ListenAddr string
	BuilderURL string
}

// New creates a new API server
func New(db *database.DB, config Config) *Server {
	s := &Server{
		db:     db,
		router: mux.NewRouter(),
		config: config,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/enroll", s.handleEnroll).Methods("POST")
	api.HandleFunc("/machines", s.handleListMachines).Methods("GET")
	api.HandleFunc("/machines/{id}", s.handleGetMachine).Methods("GET")
	api.HandleFunc("/machines/{id}", s.handleUpdateMachine).Methods("PUT")
	api.HandleFunc("/machines/{id}", s.handleDeleteMachine).Methods("DELETE")
	api.HandleFunc("/machines/{id}/build", s.handleBuildMachine).Methods("POST")
	api.HandleFunc("/machines/{id}/builds", s.handleListBuilds).Methods("GET")
	api.HandleFunc("/builds/{id}", s.handleGetBuild).Methods("GET")

	// Health check
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Middleware
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.config.ListenAddr)
	return http.ListenAndServe(s.config.ListenAddr, s.router)
}

// handleEnroll handles machine enrollment requests
func (s *Server) handleEnroll(w http.ResponseWriter, r *http.Request) {
	var req models.EnrollmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.ServiceTag == "" || req.MACAddress == "" {
		respondError(w, http.StatusBadRequest, "service_tag and mac_address are required")
		return
	}

	// Check if machine already exists
	existing, err := s.db.GetMachineByServiceTag(req.ServiceTag)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if existing != nil {
		// Update last_seen_at
		now := time.Now()
		existing.LastSeenAt = &now
		if err := s.db.UpdateMachine(existing); err != nil {
			log.Printf("Failed to update last_seen_at: %v", err)
		}
		respondJSON(w, http.StatusOK, existing)
		return
	}

	// Create new machine
	machine, err := s.db.CreateMachine(req)
	if err != nil {
		log.Printf("Failed to create machine: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create machine")
		return
	}

	log.Printf("Enrolled new machine: %s (service_tag: %s)", machine.ID, machine.ServiceTag)
	respondJSON(w, http.StatusCreated, machine)
}

// handleListMachines lists all machines
func (s *Server) handleListMachines(w http.ResponseWriter, r *http.Request) {
	machines, err := s.db.ListMachines()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list machines")
		return
	}

	respondJSON(w, http.StatusOK, machines)
}

// handleGetMachine retrieves a single machine
func (s *Server) handleGetMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	machine, err := s.db.GetMachine(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if machine == nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	respondJSON(w, http.StatusOK, machine)
}

// handleUpdateMachine updates a machine
func (s *Server) handleUpdateMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	machine, err := s.db.GetMachine(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if machine == nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	var updates models.Machine
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if updates.Hostname != "" {
		machine.Hostname = updates.Hostname
	}
	if updates.Description != "" {
		machine.Description = updates.Description
	}
	if updates.NixOSConfig != "" {
		machine.NixOSConfig = updates.NixOSConfig
		machine.Status = models.StatusConfigured
	}

	if err := s.db.UpdateMachine(machine); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update machine")
		return
	}

	respondJSON(w, http.StatusOK, machine)
}

// handleDeleteMachine deletes a machine
func (s *Server) handleDeleteMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.db.DeleteMachine(id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete machine")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleBuildMachine triggers a build for a machine
func (s *Server) handleBuildMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	machine, err := s.db.GetMachine(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if machine == nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	if machine.NixOSConfig == "" {
		respondError(w, http.StatusBadRequest, "machine has no configuration")
		return
	}

	// Create build request
	build, err := s.db.CreateBuild(machine.ID, machine.NixOSConfig)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create build")
		return
	}

	// Update machine status
	machine.Status = models.StatusBuilding
	machine.LastBuildID = &build.ID
	if err := s.db.UpdateMachine(machine); err != nil {
		log.Printf("Failed to update machine status: %v", err)
	}

	// TODO: Send build request to builder service
	log.Printf("Build requested for machine %s: build_id=%s", machine.ID, build.ID)

	respondJSON(w, http.StatusCreated, build)
}

// handleListBuilds lists builds for a machine
func (s *Server) handleListBuilds(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]

	builds, err := s.db.ListBuildsByMachine(machineID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list builds")
		return
	}

	respondJSON(w, http.StatusOK, builds)
}

// handleGetBuild retrieves a build
func (s *Server) handleGetBuild(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	build, err := s.db.GetBuild(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if build == nil {
		respondError(w, http.StatusNotFound, "build not found")
		return
	}

	respondJSON(w, http.StatusOK, build)
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
