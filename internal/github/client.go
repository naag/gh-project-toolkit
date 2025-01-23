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

// ProjectFieldConfig represents a field configuration in a GitHub project
type ProjectFieldConfig struct {
	ID   string
	Name string
	Type string // e.g., "ProjectV2Field", "ProjectV2SingleSelectField"
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
	// GetProjectID retrieves the globally unique node ID for a project
	GetProjectID(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int) (string, error)

	// GetProjectFields retrieves field values for an issue in a project
	GetProjectFields(ctx context.Context, projectID string, issueURL string) ([]ProjectField, error)

	// UpdateProjectField updates a field value for an issue in a project
	UpdateProjectField(ctx context.Context, projectID string, issueURL string, field ProjectField, dryRun bool) error

	// GetProjectIssues retrieves all issue URLs from a project
	GetProjectIssues(ctx context.Context, projectID string) ([]string, error)

	// GetProjectFieldConfigsAndIssues retrieves field configurations and issues for both projects
	GetProjectFieldConfigsAndIssues(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []ProjectFieldConfig, targetConfigs []ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error)

	// GetProjectFieldValues retrieves field values for an issue in a project, using pre-fetched field configurations
	GetProjectFieldValues(ctx context.Context, projectID string, issueURL string, fieldConfigs []ProjectFieldConfig) ([]ProjectField, error)

	// GetIssueTitle retrieves the title of an issue by its URL
	GetIssueTitle(ctx context.Context, issueURL string) (string, error)
}
