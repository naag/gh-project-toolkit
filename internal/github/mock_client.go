package github

import (
	"context"
)

// MockClient implements MockClient interface for testing
type MockClient struct {
	GetProjectIDFunc                    func(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int) (string, error)
	GetProjectFieldsFunc                func(ctx context.Context, projectID string, issueURL string) ([]ProjectField, error)
	UpdateProjectFieldFunc              func(ctx context.Context, projectID string, issueURL string, field ProjectField, dryRun bool) error
	GetProjectIssuesFunc                func(ctx context.Context, projectID string) ([]string, error)
	GetProjectFieldConfigsAndIssuesFunc func(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []ProjectFieldConfig, targetConfigs []ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error)
	GetProjectFieldValuesFunc           func(ctx context.Context, projectID string, issueURL string, fieldConfigs []ProjectFieldConfig) ([]ProjectField, error)
	GetIssueTitleFunc                   func(ctx context.Context, issueURL string) (string, error)
}

// GetProjectID implements the Client interface
func (c *MockClient) GetProjectID(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int) (string, error) {
	if c.GetProjectIDFunc != nil {
		return c.GetProjectIDFunc(ctx, ownerType, ownerLogin, projectNumber)
	}
	return "", nil
}

// GetProjectFields implements the Client interface
func (c *MockClient) GetProjectFields(ctx context.Context, projectID string, issueURL string) ([]ProjectField, error) {
	if c.GetProjectFieldsFunc != nil {
		return c.GetProjectFieldsFunc(ctx, projectID, issueURL)
	}
	return nil, nil
}

// UpdateProjectField implements the Client interface
func (c *MockClient) UpdateProjectField(ctx context.Context, projectID string, issueURL string, field ProjectField, dryRun bool) error {
	if c.UpdateProjectFieldFunc != nil {
		return c.UpdateProjectFieldFunc(ctx, projectID, issueURL, field, dryRun)
	}
	return nil
}

// GetProjectIssues implements the Client interface
func (c *MockClient) GetProjectIssues(ctx context.Context, projectID string) ([]string, error) {
	if c.GetProjectIssuesFunc != nil {
		return c.GetProjectIssuesFunc(ctx, projectID)
	}
	return nil, nil
}

// GetProjectFieldConfigsAndIssues implements the Client interface
func (c *MockClient) GetProjectFieldConfigsAndIssues(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []ProjectFieldConfig, targetConfigs []ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error) {
	if c.GetProjectFieldConfigsAndIssuesFunc != nil {
		return c.GetProjectFieldConfigsAndIssuesFunc(ctx, sourceProjectID, targetProjectID)
	}
	return nil, nil, nil, nil, nil
}

// GetProjectFieldValues implements the Client interface
func (c *MockClient) GetProjectFieldValues(ctx context.Context, projectID string, issueURL string, fieldConfigs []ProjectFieldConfig) ([]ProjectField, error) {
	if c.GetProjectFieldValuesFunc != nil {
		return c.GetProjectFieldValuesFunc(ctx, projectID, issueURL, fieldConfigs)
	}
	return nil, nil
}

// GetIssueTitle implements the Client interface
func (c *MockClient) GetIssueTitle(ctx context.Context, issueURL string) (string, error) {
	if c.GetIssueTitleFunc != nil {
		return c.GetIssueTitleFunc(ctx, issueURL)
	}
	return "", nil
}
