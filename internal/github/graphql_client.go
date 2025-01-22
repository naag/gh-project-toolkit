package github

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// GraphQLClient implements the Client interface using GitHub's GraphQL API
type GraphQLClient struct {
	client *githubv4.Client
}

// CustomDate is a custom date type that can parse GitHub's date format
type CustomDate struct {
	time.Time
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (d *CustomDate) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := string(data)
	s = s[1 : len(s)-1]

	// Parse date in YYYY-MM-DD format
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return err
	}

	d.Time = t
	return nil
}

// NewGraphQLClient creates a new GitHub GraphQL client using the token from GITHUB_TOKEN env var
func NewGraphQLClient(verbose bool) (*GraphQLClient, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable not set")
	}

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	if verbose {
		httpClient.Transport = &debugTransport{
			transport: httpClient.Transport,
		}
	}

	client := githubv4.NewClient(httpClient)
	return &GraphQLClient{client: client}, nil
}

// ProjectV2 represents a GitHub project (v2)
type ProjectV2 struct {
	ID     string
	Fields struct {
		Nodes []ProjectV2FieldConfiguration
	} `graphql:"fields(first: 100)"`
	Items struct {
		Nodes []ProjectV2Item
	} `graphql:"items(first: 100)"`
}

// GraphQL query types for GitHub's API
type (
	// ProjectV2FieldConfiguration represents a field configuration in a project
	ProjectV2FieldConfiguration struct {
		TypeName string `graphql:"__typename"`
		// Common fields for all field types
		DateField struct {
			ID   string
			Name string
		} `graphql:"... on ProjectV2Field"`
		SingleSelectField struct {
			ID   string
			Name string
		} `graphql:"... on ProjectV2SingleSelectField"`
	}

	// ProjectV2Item represents an item (issue) in a GitHub project
	ProjectV2Item struct {
		ID     string
		Fields struct {
			Nodes []ProjectV2ItemFieldValue
		} `graphql:"fieldValues(first: 100)"`
		Content struct {
			TypeName string `graphql:"__typename"`
			Issue    struct {
				URL string
			} `graphql:"... on Issue"`
		}
	}

	// ProjectV2ItemFieldValue represents a field value for an item
	ProjectV2ItemFieldValue struct {
		TypeName string `graphql:"__typename"`
		// Date field value
		DateValue struct {
			Field struct {
				TypeName  string `graphql:"__typename"`
				DateField struct {
					ID   string
					Name string
				} `graphql:"... on ProjectV2Field"`
			}
			Date *CustomDate `graphql:"date"`
		} `graphql:"... on ProjectV2ItemFieldDateValue"`
		// Single select field value
		SingleSelectValue struct {
			Field struct {
				TypeName          string `graphql:"__typename"`
				SingleSelectField struct {
					ID   string
					Name string
				} `graphql:"... on ProjectV2SingleSelectField"`
			}
			Name *string
		} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
	}
)

func (c *GraphQLClient) getOrgProject(ctx context.Context, orgName string, projectNumber int) (*ProjectV2, error) {
	if projectNumber <= 0 {
		return nil, fmt.Errorf("invalid project number: %d", projectNumber)
	}

	var query struct {
		Organization struct {
			ProjectV2 ProjectV2 `graphql:"projectV2(number: $projectNumber)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":         githubv4.String(orgName),
		"projectNumber": githubv4.Int(projectNumber),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query organization project: %w", err)
	}

	return &query.Organization.ProjectV2, nil
}

func (c *GraphQLClient) getUserProject(ctx context.Context, username string, projectNumber int) (*ProjectV2, error) {
	if projectNumber <= 0 {
		return nil, fmt.Errorf("invalid project number: %d", projectNumber)
	}

	var query struct {
		User struct {
			ProjectV2 ProjectV2 `graphql:"projectV2(number: $projectNumber)"`
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":         githubv4.String(username),
		"projectNumber": githubv4.Int(projectNumber),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query user project: %w", err)
	}

	return &query.User.ProjectV2, nil
}

// GetProjectFields implements the Client interface
func (c *GraphQLClient) GetProjectFields(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int, issueURL string) ([]ProjectField, error) {
	var project *ProjectV2
	var err error

	switch ownerType {
	case OwnerTypeUser:
		project, err = c.getUserProject(ctx, ownerLogin, projectNumber)
	case OwnerTypeOrg:
		project, err = c.getOrgProject(ctx, ownerLogin, projectNumber)
	default:
		return nil, fmt.Errorf("invalid owner type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Find the item (issue) in the project
	var targetItem *ProjectV2Item
	for _, item := range project.Items.Nodes {
		if item.Content.TypeName == "Issue" && item.Content.Issue.URL == issueURL {
			targetItem = &item
			break
		}
	}

	if targetItem == nil {
		return nil, fmt.Errorf("issue %s not found in project", issueURL)
	}

	// Convert field values to our internal format
	var fields []ProjectField
	for _, fieldValue := range targetItem.Fields.Nodes {
		var field ProjectField

		switch fieldValue.TypeName {
		case "ProjectV2ItemFieldDateValue":
			if fieldValue.DateValue.Date != nil {
				field = ProjectField{
					ID:   fieldValue.DateValue.Field.DateField.ID,
					Name: fieldValue.DateValue.Field.DateField.Name,
					Value: FieldValue{
						Date: &fieldValue.DateValue.Date.Time,
					},
				}
			}
		case "ProjectV2ItemFieldSingleSelectValue":
			field = ProjectField{
				ID:   fieldValue.SingleSelectValue.Field.SingleSelectField.ID,
				Name: fieldValue.SingleSelectValue.Field.SingleSelectField.Name,
				Value: FieldValue{
					Text: fieldValue.SingleSelectValue.Name,
				},
			}
		}

		if field.ID != "" { // Only add if we handled this field type
			fields = append(fields, field)
		}
	}

	return fields, nil
}

// findProjectItem finds an item in a project by its issue URL and field name
func (c *GraphQLClient) findProjectItem(project *ProjectV2, issueURL string, fieldName string) (string, *ProjectV2ItemFieldValue, error) {
	for _, item := range project.Items.Nodes {
		if item.Content.TypeName == "Issue" && item.Content.Issue.URL == issueURL {
			// Find current value of the field we want to update
			for _, fieldValue := range item.Fields.Nodes {
				switch fieldValue.TypeName {
				case "ProjectV2ItemFieldDateValue":
					if fieldValue.DateValue.Field.DateField.Name == fieldName {
						return item.ID, &fieldValue, nil
					}
				case "ProjectV2ItemFieldSingleSelectValue":
					if fieldValue.SingleSelectValue.Field.SingleSelectField.Name == fieldName {
						return item.ID, &fieldValue, nil
					}
				}
			}
			return item.ID, nil, nil
		}
	}
	return "", nil, fmt.Errorf("issue %s not found in project", issueURL)
}

// findProjectField finds a field configuration in a project by its name
func (c *GraphQLClient) findProjectField(project *ProjectV2, fieldName string) (string, bool, error) {
	for _, f := range project.Fields.Nodes {
		switch f.TypeName {
		case "ProjectV2Field":
			if f.DateField.Name == fieldName {
				return f.DateField.ID, true, nil
			}
		case "ProjectV2SingleSelectField":
			if f.SingleSelectField.Name == fieldName {
				return f.SingleSelectField.ID, false, nil
			}
		}
	}
	return "", false, fmt.Errorf("field %s not found in project", fieldName)
}

// valuesEqual checks if the current field value equals the new value
func (c *GraphQLClient) valuesEqual(currentValue *ProjectV2ItemFieldValue, field ProjectField) bool {
	if currentValue == nil {
		return false
	}

	switch currentValue.TypeName {
	case "ProjectV2ItemFieldDateValue":
		if currentValue.DateValue.Date != nil && field.Value.Date != nil {
			return currentValue.DateValue.Date.Time.Equal(*field.Value.Date)
		}
	case "ProjectV2ItemFieldSingleSelectValue":
		if currentValue.SingleSelectValue.Name != nil && field.Value.Text != nil {
			return *currentValue.SingleSelectValue.Name == *field.Value.Text
		}
	}
	return false
}

// UpdateProjectField implements the Client interface
func (c *GraphQLClient) UpdateProjectField(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int, issueURL string, field ProjectField) error {
	var project *ProjectV2
	var err error

	switch ownerType {
	case OwnerTypeUser:
		project, err = c.getUserProject(ctx, ownerLogin, projectNumber)
	case OwnerTypeOrg:
		project, err = c.getOrgProject(ctx, ownerLogin, projectNumber)
	default:
		return fmt.Errorf("invalid owner type")
	}

	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	// Find the item and its current field value
	itemID, currentValue, err := c.findProjectItem(project, issueURL, field.Name)
	if err != nil {
		return err
	}

	// Check if we need to update the value
	if c.valuesEqual(currentValue, field) {
		slog.Info("skipping field update",
			"message", "field already up to date",
			"field", field.Name,
			"value", field.Value.Date,
		)
		return nil
	}

	// Find the field configuration
	fieldID, isDateField, err := c.findProjectField(project, field.Name)
	if err != nil {
		return err
	}

	// If we get here, we need to update the field
	slog.Info("syncing field",
		"message", "updating field value",
		"field", field.Name,
	)

	// Update the field value
	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	// Construct the input based on field type
	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: project.ID,
		ItemID:    itemID,
		FieldID:   fieldID,
	}

	switch {
	case isDateField && field.Value.Date != nil:
		date := githubv4.Date{Time: *field.Value.Date}
		input.Value = githubv4.ProjectV2FieldValue{Date: &date}
	case !isDateField && field.Value.Text != nil:
		text := githubv4.String(*field.Value.Text)
		input.Value = githubv4.ProjectV2FieldValue{Text: &text}
	default:
		return fmt.Errorf("unsupported field value type")
	}

	return c.client.Mutate(ctx, &mutation, input, nil)
}
