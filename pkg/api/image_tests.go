package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// handleCreateImageTest creates a new image test
func (s *Server) handleCreateImageTest(w http.ResponseWriter, r *http.Request) {
	var test models.ImageTest
	if err := json.NewDecoder(r.Body).Decode(&test); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if test.ImagePath == "" || test.ImageType == "" || test.TestType == "" {
		http.Error(w, "image_path, image_type, and test_type are required", http.StatusBadRequest)
		return
	}

	// Set initial status
	test.Status = "pending"

	// Create test
	if err := s.db.CreateImageTest(&test); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create image test: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(test)
}

// handleGetImageTest retrieves an image test by ID
func (s *Server) handleGetImageTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	test, err := s.db.GetImageTest(testID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get image test: %v", err), http.StatusInternalServerError)
		return
	}
	if test == nil {
		http.Error(w, "Image test not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(test)
}

// handleListImageTests retrieves image tests
func (s *Server) handleListImageTests(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	imageType := r.URL.Query().Get("image_type")
	limitStr := r.URL.Query().Get("limit")

	// Default limit to 50
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	tests, err := s.db.ListImageTests(imageType, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list image tests: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tests)
}

// handleUpdateImageTest updates an image test
func (s *Server) handleUpdateImageTest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	testID := vars["id"]

	// Get existing test
	test, err := s.db.GetImageTest(testID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get image test: %v", err), http.StatusInternalServerError)
		return
	}
	if test == nil {
		http.Error(w, "Image test not found", http.StatusNotFound)
		return
	}

	// Parse update
	var update models.ImageTest
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update fields
	if update.Status != "" {
		test.Status = update.Status
	}
	if update.Result != "" {
		test.Result = update.Result
	}
	if update.Error != "" {
		test.Error = update.Error
	}
	if update.CompletedAt != nil {
		test.CompletedAt = update.CompletedAt
	}

	// Update in database
	if err := s.db.UpdateImageTest(test); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update image test: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(test)
}
