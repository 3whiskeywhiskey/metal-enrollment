package models

import (
	"time"
)

// UserRole represents the role of a user in the system
type UserRole string

const (
	RoleAdmin    UserRole = "admin"    // Full access to all resources
	RoleOperator UserRole = "operator" // Can manage machines and builds
	RoleViewer   UserRole = "viewer"   // Read-only access
)

// User represents a user in the system
type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Never expose in JSON
	Role         UserRole  `json:"role" db:"role"`
	Active       bool      `json:"active" db:"active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Password string   `json:"password"`
	Role     UserRole `json:"role"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	Email    string   `json:"email,omitempty"`
	Password string   `json:"password,omitempty"`
	Role     UserRole `json:"role,omitempty"`
	Active   bool     `json:"active"`
}

// APIKeyRequest represents an API key generation request
type APIKeyRequest struct {
	Name      string    `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID        string     `json:"id" db:"id"`
	UserID    string     `json:"user_id" db:"user_id"`
	Name      string     `json:"name" db:"name"`
	Key       string     `json:"key" db:"key"` // Hashed in database
	Active    bool       `json:"active" db:"active"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}
