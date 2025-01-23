package sync

import (
	"context"
	"testing"
	"time"

	"github.com/naag/gh-project-toolkit/internal/github"
)

func TestSyncFieldsWithoutDryRun(t *testing.T) {
	now := time.Now()
	mockClient := &github.MockClient{
		GetProjectIDFunc: func(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) (string, error) {
			if projectNumber == 824 {
				return "project_1", nil
			}
			return "project_2", nil
		},
		GetProjectFieldConfigsAndIssuesFunc: func(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []github.ProjectFieldConfig, targetConfigs []github.ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error) {
			return []github.ProjectFieldConfig{
					{ID: "1", Name: "start", Type: "ProjectV2Field"},
				},
				[]github.ProjectFieldConfig{
					{ID: "2", Name: "Start date", Type: "ProjectV2Field"},
				},
				[]string{"https://github.com/org/repo/issues/1"},
				[]string{"https://github.com/org/repo/issues/1"},
				nil
		},
		GetProjectFieldValuesFunc: func(ctx context.Context, projectID string, issueURL string, fieldConfigs []github.ProjectFieldConfig) ([]github.ProjectField, error) {
			if projectID == "project_1" {
				return []github.ProjectField{
					{
						ID:   "1",
						Name: "start",
						Value: github.FieldValue{
							Date: &now,
						},
					},
				}, nil
			}
			return []github.ProjectField{}, nil
		},
		UpdateProjectFieldFunc: func(ctx context.Context, projectID string, issueURL string, field github.ProjectField, dryRun bool) error {
			if field.Name != "Start date" {
				t.Errorf("expected field name 'Start date', got %s", field.Name)
			}
			if field.Value.Date != &now {
				t.Error("expected date value to match")
			}
			if dryRun {
				t.Error("expected dryRun to be false")
			}
			return nil
		},
		GetIssueTitleFunc: func(ctx context.Context, issueURL string) (string, error) {
			return "Test Issue", nil
		},
	}

	service := NewService(mockClient, false)

	err := service.SyncFields(
		context.Background(),
		github.OwnerTypeOrg,
		"myorg",
		824,
		825,
		[]string{"https://github.com/org/repo/issues/1"},
		[]FieldMapping{{SourceField: "start", TargetField: "Start date"}},
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSyncFieldsWithDryRun(t *testing.T) {
	now := time.Now()
	mockClient := &github.MockClient{
		GetProjectIDFunc: func(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) (string, error) {
			if projectNumber == 824 {
				return "project_1", nil
			}
			return "project_2", nil
		},
		GetProjectFieldConfigsAndIssuesFunc: func(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []github.ProjectFieldConfig, targetConfigs []github.ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error) {
			return []github.ProjectFieldConfig{
					{ID: "1", Name: "start", Type: "ProjectV2Field"},
				},
				[]github.ProjectFieldConfig{
					{ID: "2", Name: "Start date", Type: "ProjectV2Field"},
				},
				[]string{"https://github.com/org/repo/issues/1"},
				[]string{"https://github.com/org/repo/issues/1"},
				nil
		},
		GetProjectFieldValuesFunc: func(ctx context.Context, projectID string, issueURL string, fieldConfigs []github.ProjectFieldConfig) ([]github.ProjectField, error) {
			if projectID == "project_1" {
				return []github.ProjectField{
					{
						ID:   "1",
						Name: "start",
						Value: github.FieldValue{
							Date: &now,
						},
					},
				}, nil
			}
			return []github.ProjectField{}, nil
		},
		UpdateProjectFieldFunc: func(ctx context.Context, projectID string, issueURL string, field github.ProjectField, dryRun bool) error {
			if field.Name != "Start date" {
				t.Errorf("expected field name 'Start date', got %s", field.Name)
			}
			if field.Value.Date != &now {
				t.Error("expected date value to match")
			}
			if !dryRun {
				t.Error("expected dryRun to be true")
			}
			return nil
		},
		GetIssueTitleFunc: func(ctx context.Context, issueURL string) (string, error) {
			return "Test Issue", nil
		},
	}

	service := NewService(mockClient, true)

	err := service.SyncFields(
		context.Background(),
		github.OwnerTypeOrg,
		"myorg",
		824,
		825,
		[]string{"https://github.com/org/repo/issues/1"},
		[]FieldMapping{{SourceField: "start", TargetField: "Start date"}},
	)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
