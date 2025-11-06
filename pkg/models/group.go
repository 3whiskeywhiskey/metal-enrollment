package models

import (
	"time"
)

// MachineGroup represents a logical grouping of machines
type MachineGroup struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Tags        []string  `json:"tags,omitempty" db:"tags"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateGroupRequest represents a request to create a new group
type CreateGroupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

// UpdateGroupRequest represents a request to update a group
type UpdateGroupRequest struct {
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// GroupMembership represents the association between a machine and a group
type GroupMembership struct {
	GroupID   string    `json:"group_id" db:"group_id"`
	MachineID string    `json:"machine_id" db:"machine_id"`
	AddedAt   time.Time `json:"added_at" db:"added_at"`
}

// BulkOperationRequest represents a request to perform an operation on multiple machines
type BulkOperationRequest struct {
	MachineIDs []string               `json:"machine_ids,omitempty"`
	GroupID    string                 `json:"group_id,omitempty"`
	Operation  string                 `json:"operation"` // update, build, delete
	Data       map[string]interface{} `json:"data,omitempty"`
}

// BulkOperationResult represents the result of a bulk operation
type BulkOperationResult struct {
	TotalCount   int      `json:"total_count"`
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	Errors       []string `json:"errors,omitempty"`
}
