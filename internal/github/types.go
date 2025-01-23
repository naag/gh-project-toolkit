package github

import (
	"time"
)

// ProjectInfo contains the parsed information from a GitHub project URL
type ProjectInfo struct {
	OwnerType     ProjectOwnerType
	OwnerLogin    string
	ProjectNumber int
}

// ProjectFieldValue represents a project field value
type ProjectFieldValue struct {
	Date *time.Time
	Text *string
}

// ProjectField represents a field in a GitHub project
type ProjectField struct {
	ID    string
	Name  string
	Value ProjectFieldValue
}

// ProjectFieldConfig represents a field configuration in a GitHub project
type ProjectFieldConfig struct {
	ID   string
	Name string
	Type string // e.g., "ProjectV2Field", "ProjectV2SingleSelectField"
}

// ProjectOwnerType represents the type of project owner (user or organization)
type ProjectOwnerType string

const (
	// ProjectOwnerTypeUser represents a user-owned project
	ProjectOwnerTypeUser ProjectOwnerType = "user"
	// ProjectOwnerTypeOrg represents an organization-owned project
	ProjectOwnerTypeOrg ProjectOwnerType = "org"
)
