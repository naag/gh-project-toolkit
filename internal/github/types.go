package github

import (
	"time"
)

type ProjectInfo struct {
	OwnerType     ProjectOwnerType
	OwnerLogin    string
	ProjectNumber int
}

type ProjectFieldValue struct {
	Date *time.Time
	Text *string
}

type ProjectField struct {
	ID    string
	Name  string
	Value ProjectFieldValue
}

type ProjectFieldConfig struct {
	ID   string
	Name string
	Type string // e.g., "ProjectV2Field", "ProjectV2SingleSelectField"
}

type ProjectOwnerType string

const (
	// ProjectOwnerTypeUser represents a user-owned project
	ProjectOwnerTypeUser ProjectOwnerType = "user"
	// ProjectOwnerTypeOrg represents an organization-owned project
	ProjectOwnerTypeOrg ProjectOwnerType = "org"
)
