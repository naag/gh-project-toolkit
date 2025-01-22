package sync

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// FieldMapping represents a mapping between source and target field names
type FieldMapping struct {
	SourceField string
	TargetField string
}

// Service handles field synchronization between projects
type Service struct {
	client github.Client
}

// NewService creates a new sync service
func NewService(client github.Client) *Service {
	return &Service{client: client}
}

// SyncFields synchronizes fields from source project to target project for given issues
func (s *Service) SyncFields(ctx context.Context, ownerType github.OwnerType, ownerLogin string, sourceProject, targetProject int, issueURLs []string, fieldMappings []FieldMapping) error {
	var hasErrors bool
	for _, issueURL := range issueURLs {
		if err := s.syncFieldsForIssue(ctx, ownerType, ownerLogin, sourceProject, targetProject, issueURL, fieldMappings); err != nil {
			slog.Error("failed to sync fields for issue", "error", err, "issue", issueURL)
			hasErrors = true
		}
	}
	if hasErrors {
		return fmt.Errorf("one or more sync operations failed")
	}
	return nil
}

func (s *Service) syncFieldsForIssue(ctx context.Context, ownerType github.OwnerType, ownerLogin string, sourceProject, targetProject int, issueURL string, fieldMappings []FieldMapping) error {
	var sourceProjectURL, targetProjectURL string
	switch ownerType {
	case github.OwnerTypeUser:
		sourceProjectURL = fmt.Sprintf("https://github.com/users/%s/projects/%d", ownerLogin, sourceProject)
		targetProjectURL = fmt.Sprintf("https://github.com/users/%s/projects/%d", ownerLogin, targetProject)
	case github.OwnerTypeOrg:
		sourceProjectURL = fmt.Sprintf("https://github.com/orgs/%s/projects/%d", ownerLogin, sourceProject)
		targetProjectURL = fmt.Sprintf("https://github.com/orgs/%s/projects/%d", ownerLogin, targetProject)
	}
	slog.Info("processing issue",
		"message", "syncing fields",
		"issue", issueURL,
		"source_project", sourceProjectURL,
		"target_project", targetProjectURL,
	)

	sourceFields, err := s.client.GetProjectFields(ctx, ownerType, ownerLogin, sourceProject, issueURL)
	if err != nil {
		return fmt.Errorf("failed to get source fields: %w", err)
	}

	var hasErrors bool
	for _, mapping := range fieldMappings {
		var sourceField *github.ProjectField
		for _, field := range sourceFields {
			if field.Name == mapping.SourceField {
				sourceField = &field
				break
			}
		}

		if sourceField == nil {
			slog.Warn("source field not found",
				"message", "field not found",
				"source_field", mapping.SourceField,
				"issue", issueURL,
			)
			hasErrors = true
			continue
		}

		targetField := github.ProjectField{
			Name:  mapping.TargetField,
			Value: sourceField.Value,
		}

		if err := s.client.UpdateProjectField(ctx, ownerType, ownerLogin, targetProject, issueURL, targetField); err != nil {
			slog.Error("failed to update target field",
				"message", "field update failed",
				"target_field", mapping.TargetField,
				"issue", issueURL,
				"error", err,
			)
			hasErrors = true
			continue
		}
	}

	if hasErrors {
		return fmt.Errorf("one or more field updates failed")
	}
	return nil
}
