package mock

import (
	"context"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// Client implements github.Client interface for testing
type Client struct {
	GetProjectIDFunc                    func(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) (string, error)
	GetProjectFieldsFunc                func(ctx context.Context, projectID string, issueURL string) ([]github.ProjectField, error)
	UpdateProjectFieldFunc              func(ctx context.Context, projectID string, issueURL string, field github.ProjectField, dryRun bool) error
	GetProjectIssuesFunc                func(ctx context.Context, projectID string) ([]string, error)
	GetProjectFieldConfigsAndIssuesFunc func(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []github.ProjectFieldConfig, targetConfigs []github.ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error)
	GetProjectFieldValuesFunc           func(ctx context.Context, projectID string, issueURL string, fieldConfigs []github.ProjectFieldConfig) ([]github.ProjectField, error)
}

// GetProjectID implements the github.Client interface
func (c *Client) GetProjectID(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) (string, error) {
	if c.GetProjectIDFunc != nil {
		return c.GetProjectIDFunc(ctx, ownerType, ownerLogin, projectNumber)
	}
	return "", nil
}

// GetProjectFields implements the github.Client interface
func (c *Client) GetProjectFields(ctx context.Context, projectID string, issueURL string) ([]github.ProjectField, error) {
	if c.GetProjectFieldsFunc != nil {
		return c.GetProjectFieldsFunc(ctx, projectID, issueURL)
	}
	return nil, nil
}

// UpdateProjectField implements the github.Client interface
func (c *Client) UpdateProjectField(ctx context.Context, projectID string, issueURL string, field github.ProjectField, dryRun bool) error {
	if c.UpdateProjectFieldFunc != nil {
		return c.UpdateProjectFieldFunc(ctx, projectID, issueURL, field, dryRun)
	}
	return nil
}

// GetProjectIssues implements the github.Client interface
func (c *Client) GetProjectIssues(ctx context.Context, projectID string) ([]string, error) {
	if c.GetProjectIssuesFunc != nil {
		return c.GetProjectIssuesFunc(ctx, projectID)
	}
	return nil, nil
}

// GetProjectFieldConfigsAndIssues implements the github.Client interface
func (c *Client) GetProjectFieldConfigsAndIssues(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []github.ProjectFieldConfig, targetConfigs []github.ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error) {
	if c.GetProjectFieldConfigsAndIssuesFunc != nil {
		return c.GetProjectFieldConfigsAndIssuesFunc(ctx, sourceProjectID, targetProjectID)
	}
	return nil, nil, nil, nil, nil
}

// GetProjectFieldValues implements the github.Client interface
func (c *Client) GetProjectFieldValues(ctx context.Context, projectID string, issueURL string, fieldConfigs []github.ProjectFieldConfig) ([]github.ProjectField, error) {
	if c.GetProjectFieldValuesFunc != nil {
		return c.GetProjectFieldValuesFunc(ctx, projectID, issueURL, fieldConfigs)
	}
	return nil, nil
}
