package github

import (
	"time"
)

// ProjectInfo contains the parsed information from a GitHub project URL
type ProjectInfo struct {
	OwnerType     OwnerType
	OwnerLogin    string
	ProjectNumber int
}

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
