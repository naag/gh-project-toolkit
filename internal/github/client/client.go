package client

import (
	"context"

	"github.com/naag/gh-project-toolkit/internal/github"
)

type Client interface {
	GetProjectID(ctx context.Context, projectInfo *github.ProjectInfo) (string, error)

	GetProjectFields(ctx context.Context, projectID string, issueURL string) ([]github.ProjectField, error)

	UpdateProjectField(ctx context.Context, projectID string, issueURL string, field github.ProjectField, dryRun bool) error

	GetProjectIssues(ctx context.Context, projectID string) ([]string, error)

	GetProjectFieldConfigsAndIssues(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []github.ProjectFieldConfig, targetConfigs []github.ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error)

	GetProjectFieldValues(ctx context.Context, projectID string, issueURL string, fieldConfigs []github.ProjectFieldConfig) ([]github.ProjectField, error)

	GetIssueTitle(ctx context.Context, issueURL string) (string, error)
}
