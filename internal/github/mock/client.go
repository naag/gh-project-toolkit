package mock

import (
	"context"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// Client implements github.Client interface for testing
type Client struct {
	GetProjectFieldsFunc   func(ctx context.Context, orgName string, projectNumber int, issueURL string) ([]github.ProjectField, error)
	UpdateProjectFieldFunc func(ctx context.Context, orgName string, projectNumber int, issueURL string, field github.ProjectField) error
}

func (c *Client) GetProjectFields(ctx context.Context, orgName string, projectNumber int, issueURL string) ([]github.ProjectField, error) {
	if c.GetProjectFieldsFunc != nil {
		return c.GetProjectFieldsFunc(ctx, orgName, projectNumber, issueURL)
	}
	return nil, nil
}

func (c *Client) UpdateProjectField(ctx context.Context, orgName string, projectNumber int, issueURL string, field github.ProjectField) error {
	if c.UpdateProjectFieldFunc != nil {
		return c.UpdateProjectFieldFunc(ctx, orgName, projectNumber, issueURL, field)
	}
	return nil
}
