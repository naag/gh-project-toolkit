package mock

import (
	"context"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// Client implements github.Client interface for testing
type Client struct {
	GetProjectFieldsFunc   func(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int, issueURL string) ([]github.ProjectField, error)
	UpdateProjectFieldFunc func(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int, issueURL string, field github.ProjectField) error
	GetProjectIssuesFunc   func(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) ([]string, error)
}

// GetProjectFields implements the github.Client interface
func (c *Client) GetProjectFields(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int, issueURL string) ([]github.ProjectField, error) {
	if c.GetProjectFieldsFunc != nil {
		return c.GetProjectFieldsFunc(ctx, ownerType, ownerLogin, projectNumber, issueURL)
	}
	return nil, nil
}

// UpdateProjectField implements the github.Client interface
func (c *Client) UpdateProjectField(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int, issueURL string, field github.ProjectField) error {
	if c.UpdateProjectFieldFunc != nil {
		return c.UpdateProjectFieldFunc(ctx, ownerType, ownerLogin, projectNumber, issueURL, field)
	}
	return nil
}

// GetProjectIssues implements the github.Client interface
func (c *Client) GetProjectIssues(ctx context.Context, ownerType github.OwnerType, ownerLogin string, projectNumber int) ([]string, error) {
	if c.GetProjectIssuesFunc != nil {
		return c.GetProjectIssuesFunc(ctx, ownerType, ownerLogin, projectNumber)
	}
	return nil, nil
}
