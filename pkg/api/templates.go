package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/auth"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// handleCreateTemplate creates a new machine template
func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template models.MachineTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if template.Name == "" || template.NixOSConfig == "" {
		respondError(w, http.StatusBadRequest, "name and nixos_config are required")
		return
	}

	// Get user from context
	if s.config.EnableAuth {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.Claims)
		if ok {
			template.CreatedBy = claims.UserID
		}
	}

	if template.CreatedBy == "" {
		template.CreatedBy = "system"
	}

	// Check if template with same name already exists
	existing, err := s.db.GetTemplateByName(template.Name)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "template with this name already exists")
		return
	}

	if err := s.db.CreateTemplate(&template); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create template")
		return
	}

	respondJSON(w, http.StatusCreated, template)
}

// handleListTemplates lists all templates
func (s *Server) handleListTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := s.db.ListTemplates()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list templates")
		return
	}

	respondJSON(w, http.StatusOK, templates)
}

// handleGetTemplate retrieves a single template
func (s *Server) handleGetTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	template, err := s.db.GetTemplate(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if template == nil {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}

	respondJSON(w, http.StatusOK, template)
}

// handleUpdateTemplate updates a template
func (s *Server) handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	template, err := s.db.GetTemplate(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if template == nil {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}

	var updates models.MachineTemplate
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if updates.Name != "" && updates.Name != template.Name {
		// Check if new name conflicts
		existing, err := s.db.GetTemplateByName(updates.Name)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "database error")
			return
		}
		if existing != nil {
			respondError(w, http.StatusConflict, "template with this name already exists")
			return
		}
		template.Name = updates.Name
	}
	if updates.Description != "" {
		template.Description = updates.Description
	}
	if updates.NixOSConfig != "" {
		template.NixOSConfig = updates.NixOSConfig
	}
	if updates.BMCConfig != nil {
		template.BMCConfig = updates.BMCConfig
	}
	if updates.Tags != nil {
		template.Tags = updates.Tags
	}
	if updates.Variables != nil {
		template.Variables = updates.Variables
	}

	if err := s.db.UpdateTemplate(template); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update template")
		return
	}

	respondJSON(w, http.StatusOK, template)
}

// handleDeleteTemplate deletes a template
func (s *Server) handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.db.DeleteTemplate(id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete template")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleApplyTemplate applies a template to a machine
func (s *Server) handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	machineID := vars["id"]
	templateID := vars["template_id"]

	// Get machine
	machine, err := s.db.GetMachine(machineID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if machine == nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	// Get template
	template, err := s.db.GetTemplate(templateID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if template == nil {
		respondError(w, http.StatusNotFound, "template not found")
		return
	}

	// Apply template configuration
	config := template.NixOSConfig

	// Replace variables if present
	if template.Variables != nil {
		var variables map[string]string
		if err := json.Unmarshal(template.Variables, &variables); err == nil {
			// Replace placeholders in config
			for key, value := range variables {
				// Check if machine has this variable in its hardware info or hostname
				actualValue := value
				switch key {
				case "hostname":
					if machine.Hostname != "" {
						actualValue = machine.Hostname
					}
				case "service_tag":
					actualValue = machine.ServiceTag
				case "mac_address":
					actualValue = machine.MACAddress
				}
				config = strings.ReplaceAll(config, "{{"+key+"}}", actualValue)
			}
		}
	}

	// Update machine configuration
	machine.NixOSConfig = config
	machine.Status = models.StatusConfigured

	// Apply BMC config if template has it and machine doesn't
	if template.BMCConfig != nil && machine.BMCInfo == nil {
		machine.BMCInfo = template.BMCConfig
	}

	if err := s.db.UpdateMachine(machine); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update machine")
		return
	}

	// Trigger event
	if s.webhookService != nil {
		s.webhookService.TriggerEvent("machine.template_applied", map[string]interface{}{
			"machine_id":  machine.ID,
			"template_id": template.ID,
		})
	}

	respondJSON(w, http.StatusOK, machine)
}
