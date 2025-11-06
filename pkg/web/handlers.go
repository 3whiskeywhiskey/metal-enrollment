package web

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/database"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// Server represents the web server
type Server struct {
	db        *database.DB
	router    *mux.Router
	templates map[string]*template.Template
}

// NewServer creates a new web server
func NewServer(db *database.DB) *Server {
	s := &Server{
		db:     db,
		router: mux.NewRouter(),
		templates: map[string]*template.Template{
			"index":   template.Must(template.New("index").Parse(indexTemplate)),
			"machine": template.Must(template.New("machine").Parse(machineTemplate)),
		},
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
	s.router.HandleFunc("/machines/{id}", s.handleMachine).Methods("GET")
	s.router.HandleFunc("/machines/{id}/update", s.handleUpdateMachine).Methods("POST")
	s.router.HandleFunc("/machines/{id}/build", s.handleBuildMachine).Methods("GET")
}

// Router returns the HTTP router
func (s *Server) Router() *mux.Router {
	return s.router
}

// handleIndex shows the dashboard
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	machines, err := s.db.ListMachines()
	if err != nil {
		log.Printf("Error listing machines: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Calculate stats
	stats := struct {
		TotalMachines  int
		EnrolledCount  int
		ReadyCount     int
		BuildingCount  int
		Machines       []*models.Machine
	}{
		TotalMachines: len(machines),
		Machines:      machines,
	}

	for _, m := range machines {
		switch m.Status {
		case models.StatusEnrolled:
			stats.EnrolledCount++
		case models.StatusReady:
			stats.ReadyCount++
		case models.StatusBuilding:
			stats.BuildingCount++
		}
	}

	if err := s.templates["index"].Execute(w, stats); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleMachine shows machine details
func (s *Server) handleMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	machine, err := s.db.GetMachine(id)
	if err != nil {
		log.Printf("Error getting machine: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if machine == nil {
		http.NotFound(w, r)
		return
	}

	data := struct {
		Machine *models.Machine
	}{
		Machine: machine,
	}

	if err := s.templates["machine"].Execute(w, data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleUpdateMachine updates machine configuration
func (s *Server) handleUpdateMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	machine, err := s.db.GetMachine(id)
	if err != nil {
		log.Printf("Error getting machine: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if machine == nil {
		http.NotFound(w, r)
		return
	}

	// Parse form
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	// Update fields
	hostname := r.FormValue("hostname")
	description := r.FormValue("description")
	nixosConfig := r.FormValue("nixos_config")

	if hostname != "" {
		machine.Hostname = hostname
	}
	if description != "" {
		machine.Description = description
	}
	if nixosConfig != "" {
		machine.NixOSConfig = nixosConfig
		machine.Status = models.StatusConfigured
	}

	if err := s.db.UpdateMachine(machine); err != nil {
		log.Printf("Error updating machine: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Redirect back to machine page
	http.Redirect(w, r, "/machines/"+id, http.StatusSeeOther)
}

// handleBuildMachine triggers a build
func (s *Server) handleBuildMachine(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	machine, err := s.db.GetMachine(id)
	if err != nil {
		log.Printf("Error getting machine: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if machine == nil {
		http.NotFound(w, r)
		return
	}

	if machine.NixOSConfig == "" {
		http.Error(w, "Machine has no configuration", http.StatusBadRequest)
		return
	}

	// Create build request
	build, err := s.db.CreateBuild(machine.ID, machine.NixOSConfig)
	if err != nil {
		log.Printf("Error creating build: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Update machine status
	machine.Status = models.StatusBuilding
	machine.LastBuildID = &build.ID
	now := time.Now()
	machine.LastBuildTime = &now

	if err := s.db.UpdateMachine(machine); err != nil {
		log.Printf("Error updating machine: %v", err)
	}

	log.Printf("Build triggered for machine %s: build_id=%s", machine.ID, build.ID)

	// Redirect back to machine page
	http.Redirect(w, r, "/machines/"+id, http.StatusSeeOther)
}
