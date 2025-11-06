package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// handleCreateWebhook creates a new webhook
func (s *Server) handleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	var webhook models.Webhook
	if err := json.NewDecoder(r.Body).Decode(&webhook); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if webhook.Name == "" || webhook.URL == "" || len(webhook.Events) == 0 {
		respondError(w, http.StatusBadRequest, "name, url, and events are required")
		return
	}

	// Set defaults
	if webhook.Timeout == 0 {
		webhook.Timeout = 30
	}
	if webhook.MaxRetries == 0 {
		webhook.MaxRetries = 3
	}

	if err := s.db.CreateWebhook(&webhook); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create webhook")
		return
	}

	respondJSON(w, http.StatusCreated, webhook)
}

// handleListWebhooks lists all webhooks
func (s *Server) handleListWebhooks(w http.ResponseWriter, r *http.Request) {
	webhooks, err := s.db.ListWebhooks()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list webhooks")
		return
	}

	respondJSON(w, http.StatusOK, webhooks)
}

// handleGetWebhook retrieves a single webhook
func (s *Server) handleGetWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	webhook, err := s.db.GetWebhook(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if webhook == nil {
		respondError(w, http.StatusNotFound, "webhook not found")
		return
	}

	respondJSON(w, http.StatusOK, webhook)
}

// handleUpdateWebhook updates a webhook
func (s *Server) handleUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	webhook, err := s.db.GetWebhook(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if webhook == nil {
		respondError(w, http.StatusNotFound, "webhook not found")
		return
	}

	var updates models.Webhook
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if updates.Name != "" {
		webhook.Name = updates.Name
	}
	if updates.URL != "" {
		webhook.URL = updates.URL
	}
	if len(updates.Events) > 0 {
		webhook.Events = updates.Events
	}
	if updates.Secret != "" {
		webhook.Secret = updates.Secret
	}
	webhook.Active = updates.Active
	if updates.Headers != nil {
		webhook.Headers = updates.Headers
	}
	if updates.Timeout > 0 {
		webhook.Timeout = updates.Timeout
	}
	if updates.MaxRetries > 0 {
		webhook.MaxRetries = updates.MaxRetries
	}

	if err := s.db.UpdateWebhook(webhook); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update webhook")
		return
	}

	respondJSON(w, http.StatusOK, webhook)
}

// handleDeleteWebhook deletes a webhook
func (s *Server) handleDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.db.DeleteWebhook(id); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete webhook")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListWebhookDeliveries lists deliveries for a webhook
func (s *Server) handleListWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	deliveries, err := s.db.ListWebhookDeliveries(id, limit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list deliveries")
		return
	}

	respondJSON(w, http.StatusOK, deliveries)
}
