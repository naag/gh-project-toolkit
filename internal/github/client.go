package github

import (
	"context"
	"time"
)

// FieldValue represents a project field value
type FieldValue struct {
	Date *time.Time
	Text *string
	// Add other field types as needed
}

// ProjectField represents a field in a GitHub project
type ProjectField struct {
	ID    string
	Name  string
	Value FieldValue
}

// OwnerType represents the type of project owner (user or organization)
type OwnerType int

const (
	// OwnerTypeUser represents a user-owned project
	OwnerTypeUser OwnerType = iota
	// OwnerTypeOrg represents an organization-owned project
	OwnerTypeOrg
)

// Client defines the interface for interacting with GitHub
type Client interface {
	// GetProjectFields retrieves field values for an issue in a project
	GetProjectFields(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int, issueURL string) ([]ProjectField, error)

	// UpdateProjectField updates a field value for an issue in a project
	UpdateProjectField(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int, issueURL string, field ProjectField) error

	// GetProjectIssues retrieves all issue URLs from a project
	GetProjectIssues(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int) ([]string, error)
}
