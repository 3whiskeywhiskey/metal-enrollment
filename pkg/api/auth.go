package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/3whiskeywhiskey/metal-enrollment/pkg/auth"
	"github.com/3whiskeywhiskey/metal-enrollment/pkg/models"
	"github.com/gorilla/mux"
)

// handleRegister handles user registration
func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	// Only admins can register new users
	claims, ok := auth.GetClaims(r)
	if !ok || claims.Role != models.RoleAdmin {
		respondError(w, http.StatusForbidden, "only admins can register new users")
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username, email, and password are required")
		return
	}

	// Validate role
	if req.Role == "" {
		req.Role = models.RoleViewer
	}
	if req.Role != models.RoleAdmin && req.Role != models.RoleOperator && req.Role != models.RoleViewer {
		respondError(w, http.StatusBadRequest, "invalid role")
		return
	}

	// Check if username already exists
	existing, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("Failed to check existing user: %v", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "username already exists")
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Create user
	user, err := s.db.CreateUser(req.Username, req.Email, passwordHash, req.Role)
	if err != nil {
		log.Printf("Failed to create user: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	log.Printf("Created user: %s (role: %s)", user.Username, user.Role)
	respondJSON(w, http.StatusCreated, user)
}

// handleLogin handles user login
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	// Get user
	user, err := s.db.GetUserByUsername(req.Username)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Check if user is active
	if !user.Active {
		respondError(w, http.StatusUnauthorized, "account is disabled")
		return
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	// Generate token
	token, expiresAt, err := s.jwtManager.GenerateToken(user)
	if err != nil {
		log.Printf("Failed to generate token: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	// Update last login
	if err := s.db.UpdateLastLogin(user.ID); err != nil {
		log.Printf("Failed to update last login: %v", err)
	}

	response := models.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      *user,
	}

	log.Printf("User logged in: %s", user.Username)
	respondJSON(w, http.StatusOK, response)
}

// handleRefreshToken handles token refresh
func (s *Server) handleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get current token from header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		respondError(w, http.StatusUnauthorized, "missing authorization header")
		return
	}

	// Extract token
	var token string
	if _, err := fmt.Sscanf(authHeader, "Bearer %s", &token); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid authorization header")
		return
	}

	// Refresh token
	newToken, expiresAt, err := s.jwtManager.RefreshToken(token)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	response := map[string]interface{}{
		"token":      newToken,
		"expires_at": expiresAt,
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetCurrentUser returns the currently authenticated user
func (s *Server) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.GetClaims(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := s.db.GetUser(claims.UserID)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// handleListUsers lists all users (admin only)
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers()
	if err != nil {
		log.Printf("Failed to list users: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to list users")
		return
	}

	respondJSON(w, http.StatusOK, users)
}

// handleGetUser retrieves a specific user (admin only)
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := s.db.GetUser(id)
	if err != nil {
		log.Printf("Failed to get user: %v", err)
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// handleUpdateUser updates a user (admin only)
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	user, err := s.db.GetUser(id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Password != "" {
		passwordHash, err := auth.HashPassword(req.Password)
		if err != nil {
			log.Printf("Failed to hash password: %v", err)
			respondError(w, http.StatusInternalServerError, "failed to update user")
			return
		}
		user.PasswordHash = passwordHash
	}
	if req.Role != "" {
		if req.Role != models.RoleAdmin && req.Role != models.RoleOperator && req.Role != models.RoleViewer {
			respondError(w, http.StatusBadRequest, "invalid role")
			return
		}
		user.Role = req.Role
	}
	user.Active = req.Active

	if err := s.db.UpdateUser(user); err != nil {
		log.Printf("Failed to update user: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to update user")
		return
	}

	respondJSON(w, http.StatusOK, user)
}

// handleDeleteUser deletes a user (admin only)
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Prevent deleting self
	claims, ok := auth.GetClaims(r)
	if ok && claims.UserID == id {
		respondError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}

	if err := s.db.DeleteUser(id); err != nil {
		log.Printf("Failed to delete user: %v", err)
		respondError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
