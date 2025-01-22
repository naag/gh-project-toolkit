package sync

import (
	"context"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// Service provides functionality for syncing project fields
type Service struct {
	client github.Client
}

// NewService creates a new sync service
func NewService(client github.Client) *Service {
	return &Service{client: client}
}

// GetProjectIssues retrieves all issue URLs from a project
func (s *Service) GetProjectIssues(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) ([]string, error) {
	return s.client.GetProjectIssues(ctx, ownerType, ownerLogin, projectNumber)
}

// FieldMapping represents a mapping between source and target field names
type FieldMapping struct {
	SourceField string
	TargetField string
}

// GetProjectFieldsForIssues retrieves field values for multiple issues in a project
func (s *Service) GetProjectFieldsForIssues(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int, issues []string) (map[string][]github.ProjectField, error) {
	result := make(map[string][]github.ProjectField)
	for _, issueURL := range issues {
		fields, err := s.client.GetProjectFields(ctx, ownerType, ownerLogin, projectNumber, issueURL)
		if err != nil {
			return nil, err
		}
		result[issueURL] = fields
	}
	return result, nil
}

// SyncFields syncs field values from source project to target project
func (s *Service) SyncFields(ctx context.Context, ownerType github.OwnerType, ownerLogin string, sourceProject, targetProject int, issues []string, mappings []FieldMapping) error {
	// Get all fields from source project
	sourceFields, err := s.GetProjectFieldsForIssues(ctx, ownerType, ownerLogin, sourceProject, issues)
	if err != nil {
		return err
	}

	// Get all fields from target project
	targetFields, err := s.GetProjectFieldsForIssues(ctx, ownerType, ownerLogin, targetProject, issues)
	if err != nil {
		return err
	}

	// For each issue, compare and update fields as needed
	for _, issueURL := range issues {
		sourceIssueFields := sourceFields[issueURL]
		targetIssueFields := targetFields[issueURL]

		// Create a map of target field names to their current values for quick lookup
		targetFieldMap := make(map[string]github.ProjectField)
		for _, field := range targetIssueFields {
			targetFieldMap[field.Name] = field
		}

		// Apply field mappings
		for _, mapping := range mappings {
			for _, sourceField := range sourceIssueFields {
				if sourceField.Name == mapping.SourceField {
					// Check if we need to update the target field
					targetField := github.ProjectField{
						Name:  mapping.TargetField,
						Value: sourceField.Value,
					}

					// If the field exists in target and has the same value, skip the update
					if existingField, ok := targetFieldMap[mapping.TargetField]; ok {
						if fieldsEqual(existingField, targetField) {
							continue
						}
					}

					// Update field in target project
					if err := s.client.UpdateProjectField(ctx, ownerType, ownerLogin, targetProject, issueURL, targetField); err != nil {
						return err
					}
					break
				}
			}
		}
	}

	return nil
}

// fieldsEqual checks if two fields have the same value
func fieldsEqual(a, b github.ProjectField) bool {
	if a.Value.Date != nil && b.Value.Date != nil {
		return a.Value.Date.Equal(*b.Value.Date)
	}
	if a.Value.Text != nil && b.Value.Text != nil {
		return *a.Value.Text == *b.Value.Text
	}
	return false
}
