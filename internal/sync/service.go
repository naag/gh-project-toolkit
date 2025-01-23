package sync

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// Service provides functionality for syncing project fields
type Service struct {
	client github.Client
	dryRun bool
}

// NewService creates a new sync service
func NewService(client github.Client, dryRun bool) *Service {
	return &Service{
		client: client,
		dryRun: dryRun,
	}
}

// FieldMapping represents a mapping between source and target field names
type FieldMapping struct {
	SourceField string
	TargetField string
}

// SyncFields syncs field values from source project to target project
func (s *Service) SyncFields(ctx context.Context, ownerType github.OwnerType, ownerLogin string, sourceProject, targetProject int, issues []string, mappings []FieldMapping) error {
	// First, get the project IDs
	sourceProjectID, err := s.client.GetProjectID(ctx, ownerType, ownerLogin, sourceProject)
	if err != nil {
		return fmt.Errorf("failed to get source project ID: %w", err)
	}

	targetProjectID, err := s.client.GetProjectID(ctx, ownerType, ownerLogin, targetProject)
	if err != nil {
		return fmt.Errorf("failed to get target project ID: %w", err)
	}

	// Get field configurations and issues for both projects
	sourceFieldConfigs, targetFieldConfigs, sourceIssues, targetIssues, err := s.client.GetProjectFieldConfigsAndIssues(ctx, sourceProjectID, targetProjectID)
	if err != nil {
		return fmt.Errorf("failed to get project field configs and issues: %w", err)
	}

	// If no issues were provided, use the common issues from both projects
	if len(issues) == 0 {
		issues = findCommonIssues(sourceIssues, targetIssues)
		if len(issues) == 0 {
			return fmt.Errorf("no common issues found between source and target projects")
		}
		slog.Info("found common issues", slog.Int("count", len(issues)))
	}

	// Process issues in batches to avoid too many concurrent requests
	batchSize := 10
	for i := 0; i < len(issues); i += batchSize {
		end := i + batchSize
		if end > len(issues) {
			end = len(issues)
		}
		batch := issues[i:end]

		// Get field values for all issues in the batch from both projects
		sourceValues, targetValues, err := s.getFieldValuesForBatch(ctx, sourceProjectID, targetProjectID, batch, sourceFieldConfigs, targetFieldConfigs)
		if err != nil {
			return err
		}

		// Process all issues in the batch
		for _, issueURL := range batch {
			sourceFields := sourceValues[issueURL]
			targetFields := targetValues[issueURL]

			// Create a map of target fields by name for easy lookup
			targetFieldMap := make(map[string]github.ProjectField)
			for _, field := range targetFields {
				targetFieldMap[field.Name] = field
			}

			// Apply field mappings
			if err := s.applyFieldMappings(ctx, targetProjectID, issueURL, sourceFields, targetFieldMap, mappings); err != nil {
				return err
			}
		}
	}

	return nil
}

// findCommonIssues finds common issues between two lists
func findCommonIssues(sourceIssues, targetIssues []string) []string {
	issueMap := make(map[string]bool)
	for _, issue := range targetIssues {
		issueMap[issue] = true
	}

	var commonIssues []string
	for _, issue := range sourceIssues {
		if issueMap[issue] {
			commonIssues = append(commonIssues, issue)
		}
	}

	return commonIssues
}

// getFieldValuesForBatch retrieves field values for a batch of issues from both projects
func (s *Service) getFieldValuesForBatch(ctx context.Context, sourceProjectID string, targetProjectID string, batch []string, sourceFieldConfigs []github.ProjectFieldConfig, targetFieldConfigs []github.ProjectFieldConfig) (map[string][]github.ProjectField, map[string][]github.ProjectField, error) {
	sourceValues := make(map[string][]github.ProjectField)
	targetValues := make(map[string][]github.ProjectField)

	for _, issueURL := range batch {
		// Get source values using cached data
		sourceFields, err := s.client.GetProjectFieldValues(ctx, sourceProjectID, issueURL, sourceFieldConfigs)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get source field values for %s: %w", issueURL, err)
		}
		sourceValues[issueURL] = sourceFields

		// Get target values using cached data
		targetFields, err := s.client.GetProjectFieldValues(ctx, targetProjectID, issueURL, targetFieldConfigs)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get target field values for %s: %w", issueURL, err)
		}
		targetValues[issueURL] = targetFields
	}

	return sourceValues, targetValues, nil
}

// applyFieldMappings applies field mappings for an issue
func (s *Service) applyFieldMappings(ctx context.Context, targetProjectID string, issueURL string, sourceFields []github.ProjectField, targetFieldMap map[string]github.ProjectField, mappings []FieldMapping) error {
	for _, mapping := range mappings {
		for _, sourceField := range sourceFields {
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
				if err := s.client.UpdateProjectField(ctx, targetProjectID, issueURL, targetField, s.dryRun); err != nil {
					return fmt.Errorf("failed to update field for %s: %w", issueURL, err)
				}
				break
			}
		}
	}
	return nil
}

// fieldsEqual checks if two fields have equal values
func fieldsEqual(a, b github.ProjectField) bool {
	if a.Value.Date != nil && b.Value.Date != nil {
		return a.Value.Date.Equal(*b.Value.Date)
	}
	if a.Value.Text != nil && b.Value.Text != nil {
		return *a.Value.Text == *b.Value.Text
	}
	return false
}
