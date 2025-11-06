package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/auth"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// Server represents the API server
type Server struct {
	db         *database.DB
	Router     *mux.Router
	config     Config
	jwtManager *auth.JWTManager
}

// Config holds server configuration
type Config struct {
	ListenAddr    string
	BuilderURL    string
	JWTSecret     string
	JWTExpiry     time.Duration
	EnableAuth    bool
}

// New creates a new API server
func New(db *database.DB, config Config) *Server {
	s := &Server{
		db:         db,
		Router:     mux.NewRouter(),
		config:     config,
		jwtManager: auth.NewJWTManager(config.JWTSecret, config.JWTExpiry),
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// API routes
	api := s.Router.PathPrefix("/api/v1").Subrouter()

	// Public routes (no auth required)
	api.HandleFunc("/login", s.handleLogin).Methods("POST")
	api.HandleFunc("/enroll", s.handleEnroll).Methods("POST")
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Prometheus metrics endpoint (public)
	api.HandleFunc("/metrics", s.handlePrometheusMetrics).Methods("GET")

	if s.config.EnableAuth {
		// Auth middleware for protected routes
		authMiddleware := auth.AuthMiddleware(s.jwtManager)

		// Authentication routes
		authAPI := api.PathPrefix("/auth").Subrouter()
		authAPI.Use(authMiddleware)
		authAPI.HandleFunc("/refresh", s.handleRefreshToken).Methods("POST")
		authAPI.HandleFunc("/me", s.handleGetCurrentUser).Methods("GET")

		// User management routes (admin only)
		usersAPI := api.PathPrefix("/users").Subrouter()
		usersAPI.Use(authMiddleware)
		usersAPI.Use(auth.RequireRole(models.RoleAdmin))
		usersAPI.HandleFunc("", s.handleListUsers).Methods("GET")
		usersAPI.HandleFunc("", s.handleRegister).Methods("POST")
		usersAPI.HandleFunc("/{id}", s.handleGetUser).Methods("GET")
		usersAPI.HandleFunc("/{id}", s.handleUpdateUser).Methods("PUT")
		usersAPI.HandleFunc("/{id}", s.handleDeleteUser).Methods("DELETE")

		// Machine routes (authenticated)
		machinesAPI := api.PathPrefix("/machines").Subrouter()
		machinesAPI.Use(authMiddleware)

		// Viewers can read
		machinesAPI.HandleFunc("", s.handleListMachines).Methods("GET")
		machinesAPI.HandleFunc("/{id}", s.handleGetMachine).Methods("GET")
		machinesAPI.HandleFunc("/{id}/builds", s.handleListBuilds).Methods("GET")
		machinesAPI.HandleFunc("/{id}/groups", s.handleGetMachineGroups).Methods("GET")

		// Operators and admins can modify
		operatorRoutes := machinesAPI.PathPrefix("").Subrouter()
		operatorRoutes.Use(auth.RequireRole(models.RoleOperator, models.RoleAdmin))
		operatorRoutes.HandleFunc("/{id}", s.handleUpdateMachine).Methods("PUT")
		operatorRoutes.HandleFunc("/{id}/build", s.handleBuildMachine).Methods("POST")

		// Power control routes (operators and admins only)
		operatorRoutes.HandleFunc("/{id}/power", s.handlePowerControl).Methods("POST")
		operatorRoutes.HandleFunc("/{id}/power/status", s.handleGetPowerStatus).Methods("GET")
		operatorRoutes.HandleFunc("/{id}/power/operations", s.handleGetPowerOperations).Methods("GET")
		operatorRoutes.HandleFunc("/{id}/bmc/test", s.handleTestBMC).Methods("POST")
		operatorRoutes.HandleFunc("/{id}/bmc/info", s.handleGetBMCInfo).Methods("GET")
		operatorRoutes.HandleFunc("/{id}/bmc/sensors", s.handleGetSensors).Methods("GET")

		// Metrics routes - machines can submit (authenticated but no role check)
		machinesAPI.HandleFunc("/{id}/metrics", s.handleSubmitMetrics).Methods("POST")
		machinesAPI.HandleFunc("/{id}/metrics/latest", s.handleGetLatestMetrics).Methods("GET")
		machinesAPI.HandleFunc("/{id}/metrics/history", s.handleGetMetricsHistory).Methods("GET")

		// All machines metrics (authenticated)
		metricsAPI := api.PathPrefix("/metrics").Subrouter()
		metricsAPI.Use(authMiddleware)
		metricsAPI.HandleFunc("/machines", s.handleGetAllMachinesMetrics).Methods("GET")

		// Image testing routes (operators and admins only)
		imageTestsAPI := api.PathPrefix("/image-tests").Subrouter()
		imageTestsAPI.Use(authMiddleware)
		imageTestsAPI.Use(auth.RequireRole(models.RoleOperator, models.RoleAdmin))
		imageTestsAPI.HandleFunc("", s.handleListImageTests).Methods("GET")
		imageTestsAPI.HandleFunc("", s.handleCreateImageTest).Methods("POST")
		imageTestsAPI.HandleFunc("/{id}", s.handleGetImageTest).Methods("GET")
		imageTestsAPI.HandleFunc("/{id}", s.handleUpdateImageTest).Methods("PUT")

		// Only admins can delete
		adminRoutes := machinesAPI.PathPrefix("").Subrouter()
		adminRoutes.Use(auth.RequireRole(models.RoleAdmin))
		adminRoutes.HandleFunc("/{id}", s.handleDeleteMachine).Methods("DELETE")

		// Build routes (authenticated)
		buildsAPI := api.PathPrefix("/builds").Subrouter()
		buildsAPI.Use(authMiddleware)
		buildsAPI.HandleFunc("/{id}", s.handleGetBuild).Methods("GET")

		// Group routes (authenticated)
		groupsAPI := api.PathPrefix("/groups").Subrouter()
		groupsAPI.Use(authMiddleware)

		// Viewers can read
		groupsAPI.HandleFunc("", s.handleListGroups).Methods("GET")
		groupsAPI.HandleFunc("/{id}", s.handleGetGroup).Methods("GET")
		groupsAPI.HandleFunc("/{id}/machines", s.handleGetGroupMachines).Methods("GET")

		// Operators and admins can modify
		groupOperatorRoutes := groupsAPI.PathPrefix("").Subrouter()
		groupOperatorRoutes.Use(auth.RequireRole(models.RoleOperator, models.RoleAdmin))
		groupOperatorRoutes.HandleFunc("", s.handleCreateGroup).Methods("POST")
		groupOperatorRoutes.HandleFunc("/{id}", s.handleUpdateGroup).Methods("PUT")
		groupOperatorRoutes.HandleFunc("/{id}/machines/{machine_id}", s.handleAddMachineToGroup).Methods("PUT")
		groupOperatorRoutes.HandleFunc("/{id}/machines/{machine_id}", s.handleRemoveMachineFromGroup).Methods("DELETE")

		// Only admins can delete groups
		groupAdminRoutes := groupsAPI.PathPrefix("").Subrouter()
		groupAdminRoutes.Use(auth.RequireRole(models.RoleAdmin))
		groupAdminRoutes.HandleFunc("/{id}", s.handleDeleteGroup).Methods("DELETE")

		// Bulk operations (operators and admins only)
		bulkAPI := api.PathPrefix("/bulk").Subrouter()
		bulkAPI.Use(authMiddleware)
		bulkAPI.Use(auth.RequireRole(models.RoleOperator, models.RoleAdmin))
		bulkAPI.HandleFunc("", s.handleBulkOperation).Methods("POST")
	} else {
		// No auth - all routes are public
		api.HandleFunc("/machines", s.handleListMachines).Methods("GET")
		api.HandleFunc("/machines/{id}", s.handleGetMachine).Methods("GET")
		api.HandleFunc("/machines/{id}", s.handleUpdateMachine).Methods("PUT")
		api.HandleFunc("/machines/{id}", s.handleDeleteMachine).Methods("DELETE")
		api.HandleFunc("/machines/{id}/build", s.handleBuildMachine).Methods("POST")
		api.HandleFunc("/machines/{id}/builds", s.handleListBuilds).Methods("GET")
		api.HandleFunc("/machines/{id}/groups", s.handleGetMachineGroups).Methods("GET")

		// Power control routes (no auth)
		api.HandleFunc("/machines/{id}/power", s.handlePowerControl).Methods("POST")
		api.HandleFunc("/machines/{id}/power/status", s.handleGetPowerStatus).Methods("GET")
		api.HandleFunc("/machines/{id}/power/operations", s.handleGetPowerOperations).Methods("GET")
		api.HandleFunc("/machines/{id}/bmc/test", s.handleTestBMC).Methods("POST")
		api.HandleFunc("/machines/{id}/bmc/info", s.handleGetBMCInfo).Methods("GET")
		api.HandleFunc("/machines/{id}/bmc/sensors", s.handleGetSensors).Methods("GET")

		// Metrics routes (no auth)
		api.HandleFunc("/machines/{id}/metrics", s.handleSubmitMetrics).Methods("POST")
		api.HandleFunc("/machines/{id}/metrics/latest", s.handleGetLatestMetrics).Methods("GET")
		api.HandleFunc("/machines/{id}/metrics/history", s.handleGetMetricsHistory).Methods("GET")
		api.HandleFunc("/metrics/machines", s.handleGetAllMachinesMetrics).Methods("GET")

		// Image testing routes (no auth)
		api.HandleFunc("/image-tests", s.handleListImageTests).Methods("GET")
		api.HandleFunc("/image-tests", s.handleCreateImageTest).Methods("POST")
		api.HandleFunc("/image-tests/{id}", s.handleGetImageTest).Methods("GET")
		api.HandleFunc("/image-tests/{id}", s.handleUpdateImageTest).Methods("PUT")

		api.HandleFunc("/builds/{id}", s.handleGetBuild).Methods("GET")

		// Groups
		api.HandleFunc("/groups", s.handleListGroups).Methods("GET")
		api.HandleFunc("/groups", s.handleCreateGroup).Methods("POST")
		api.HandleFunc("/groups/{id}", s.handleGetGroup).Methods("GET")
		api.HandleFunc("/groups/{id}", s.handleUpdateGroup).Methods("PUT")
		api.HandleFunc("/groups/{id}", s.handleDeleteGroup).Methods("DELETE")
		api.HandleFunc("/groups/{id}/machines", s.handleGetGroupMachines).Methods("GET")
		api.HandleFunc("/groups/{id}/machines/{machine_id}", s.handleAddMachineToGroup).Methods("PUT")
		api.HandleFunc("/groups/{id}/machines/{machine_id}", s.handleRemoveMachineFromGroup).Methods("DELETE")

		// Bulk operations
		api.HandleFunc("/bulk", s.handleBulkOperation).Methods("POST")
	}

	// Global middleware
	s.Router.Use(loggingMiddleware)
	s.Router.Use(corsMiddleware)
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting API server on %s", s.config.ListenAddr)
	return http.ListenAndServe(s.config.ListenAddr, s.Router)
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
