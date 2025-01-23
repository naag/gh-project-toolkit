package sync_fields

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

// SyncFields synchronizes field values between two GitHub projects for the specified issues
// using the provided field mappings
func (s *Service) SyncFields(ctx context.Context, sourceProjectURL, targetProjectURL string, issues []string, fieldMappings []string) error {
	// Parse project URLs and field mappings
	sourceProject, targetProject, mappings, err := s.parseInputs(sourceProjectURL, targetProjectURL, fieldMappings)
	if err != nil {
		return err
	}

	// Get project IDs
	sourceProjectID, targetProjectID, err := s.getProjectIDs(ctx, sourceProject, targetProject)
	if err != nil {
		return err
	}

	// Get field configurations and issues
	sourceFieldConfigs, targetFieldConfigs, sourceIssues, targetIssues, err := s.client.GetProjectFieldConfigsAndIssues(ctx, sourceProjectID, targetProjectID)
	if err != nil {
		return fmt.Errorf("failed to get project field configs and issues: %w", err)
	}

	// If no issues were provided, find common issues
	if len(issues) == 0 {
		issues = findCommonIssues(sourceIssues, targetIssues)
		if len(issues) == 0 {
			return fmt.Errorf("no common issues found between source and target projects")
		}
		slog.Info("found common issues",
			"count", len(issues),
			"source_issues", len(sourceIssues),
			"target_issues", len(targetIssues),
		)
	}

	return s.processBatches(ctx, sourceProjectID, targetProjectID, issues, sourceFieldConfigs, targetFieldConfigs, mappings)
}

// parseInputs parses and validates the input URLs and field mappings
func (s *Service) parseInputs(sourceProjectURL, targetProjectURL string, fieldMappings []string) (*github.ProjectInfo, *github.ProjectInfo, []FieldMapping, error) {
	sourceProject, err := github.ParseProjectURL(sourceProjectURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid source project URL: %w", err)
	}

	targetProject, err := github.ParseProjectURL(targetProjectURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid target project URL: %w", err)
	}

	mappings, err := ParseFieldMappings(fieldMappings)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse field mappings: %w", err)
	}

	return sourceProject, targetProject, mappings, nil
}

// getProjectIDs retrieves the project IDs for both source and target projects
func (s *Service) getProjectIDs(ctx context.Context, sourceProject, targetProject *github.ProjectInfo) (string, string, error) {
	sourceProjectID, err := s.client.GetProjectID(ctx, sourceProject)
	if err != nil {
		return "", "", fmt.Errorf("failed to get source project ID: %w", err)
	}

	targetProjectID, err := s.client.GetProjectID(ctx, targetProject)
	if err != nil {
		return "", "", fmt.Errorf("failed to get target project ID: %w", err)
	}

	return sourceProjectID, targetProjectID, nil
}

// processBatches processes issues in batches to avoid too many concurrent requests
func (s *Service) processBatches(ctx context.Context, sourceProjectID, targetProjectID string, issues []string, sourceFieldConfigs, targetFieldConfigs []github.ProjectFieldConfig, mappings []FieldMapping) error {
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

			// Get issue title for logging
			title, err := s.client.GetIssueTitle(ctx, issueURL)
			if err != nil {
				slog.Warn("failed to get issue title", "issue", issueURL, "error", err)
				title = "<unknown>"
			}
			slog.Info("processing issue", "url", issueURL, "title", title)

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
