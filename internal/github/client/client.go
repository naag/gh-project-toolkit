package client

import (
	"context"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// Client defines the interface for interacting with GitHub
type Client interface {
	// GetProjectID retrieves the globally unique node ID for a project
	GetProjectID(ctx context.Context, projectInfo *github.ProjectInfo) (string, error)

	// GetProjectFields retrieves field values for an issue in a project
	GetProjectFields(ctx context.Context, projectID string, issueURL string) ([]github.ProjectField, error)

	// UpdateProjectField updates a field value for an issue in a project
	UpdateProjectField(ctx context.Context, projectID string, issueURL string, field github.ProjectField, dryRun bool) error

	// GetProjectIssues retrieves all issue URLs from a project
	GetProjectIssues(ctx context.Context, projectID string) ([]string, error)

	// GetProjectFieldConfigsAndIssues retrieves field configurations and issues for both projects
	GetProjectFieldConfigsAndIssues(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []github.ProjectFieldConfig, targetConfigs []github.ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error)

	// GetProjectFieldValues retrieves field values for an issue in a project, using pre-fetched field configurations
	GetProjectFieldValues(ctx context.Context, projectID string, issueURL string, fieldConfigs []github.ProjectFieldConfig) ([]github.ProjectField, error)

	// GetIssueTitle retrieves the title of an issue by its URL
	GetIssueTitle(ctx context.Context, issueURL string) (string, error)
}
