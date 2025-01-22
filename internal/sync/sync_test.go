package sync

import (
	"context"
	"testing"
	"time"

	"github.com/naag/gh-project-toolkit/internal/github"
	"github.com/naag/gh-project-toolkit/internal/github/mock"
)

func TestSyncFields(t *testing.T) {
	now := time.Now()
	mockClient := &mock.Client{
		GetProjectFieldsFunc: func(ctx context.Context, orgName string, projectNumber int, issueURL string) ([]github.ProjectField, error) {
			return []github.ProjectField{
				{
					ID:   "1",
					Name: "start",
					Value: github.FieldValue{
						Date: &now,
					},
				},
			}, nil
		},
		UpdateProjectFieldFunc: func(ctx context.Context, orgName string, projectNumber int, issueURL string, field github.ProjectField) error {
			if field.Name != "Start date" {
				t.Errorf("expected field name 'Start date', got %s", field.Name)
			}
			if field.Value.Date != &now {
				t.Error("expected date value to match")
			}
			return nil
		},
	}

	service := NewService(mockClient)

	err := service.SyncFields(
		context.Background(),
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
