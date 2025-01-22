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
	cache  struct {
		sourceProject *ProjectV2
		targetProject *ProjectV2
		sourceNumber  int
		targetNumber  int
	}
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
			ID      string
			Name    string
			Options []struct {
				ID   string
				Name string
			} `graphql:"options"`
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

// constructMutationInput creates the input for the update mutation based on field type
func (c *GraphQLClient) constructMutationInput(projectID, itemID, fieldID string, field ProjectField, isDateField bool) (githubv4.UpdateProjectV2ItemFieldValueInput, error) {
	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: projectID,
		ItemID:    itemID,
		FieldID:   fieldID,
	}

	switch {
	case isDateField && field.Value.Date != nil:
		date := githubv4.Date{Time: *field.Value.Date}
		input.Value = githubv4.ProjectV2FieldValue{Date: &date}
	case !isDateField && field.Value.Text != nil:
		// Find the option ID for the single select value
		var optionID string
		for _, f := range c.cache.sourceProject.Fields.Nodes {
			if f.TypeName == "ProjectV2SingleSelectField" && f.SingleSelectField.Name == field.Name {
				for _, opt := range f.SingleSelectField.Options {
					if opt.Name == *field.Value.Text {
						optionID = opt.ID
						break
					}
				}
				break
			}
		}
		if optionID == "" {
			return input, fmt.Errorf("single select option %q not found for field %q", *field.Value.Text, field.Name)
		}
		optionIDv4 := githubv4.String(optionID)
		input.Value = githubv4.ProjectV2FieldValue{SingleSelectOptionID: &optionIDv4}
	default:
		return input, fmt.Errorf("unsupported field value type")
	}

	return input, nil
}

// updateCacheFieldValue updates the cached field value after a successful mutation
func (c *GraphQLClient) updateCacheFieldValue(project *ProjectV2, issueURL string, field ProjectField) {
	for i, item := range project.Items.Nodes {
		if item.Content.TypeName == "Issue" && item.Content.Issue.URL == issueURL {
			for j, fieldValue := range item.Fields.Nodes {
				switch fieldValue.TypeName {
				case "ProjectV2ItemFieldDateValue":
					if fieldValue.DateValue.Field.DateField.Name == field.Name {
						project.Items.Nodes[i].Fields.Nodes[j].DateValue.Date = &CustomDate{Time: *field.Value.Date}
					}
				case "ProjectV2ItemFieldSingleSelectValue":
					if fieldValue.SingleSelectValue.Field.SingleSelectField.Name == field.Name {
						project.Items.Nodes[i].Fields.Nodes[j].SingleSelectValue.Name = field.Value.Text
					}
				}
			}
			break
		}
	}
}

// UpdateProjectField implements the Client interface
func (c *GraphQLClient) UpdateProjectField(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int, issueURL string, field ProjectField) error {
	// Use cached data to find the project and item IDs
	var project *ProjectV2
	switch projectNumber {
	case c.cache.sourceNumber:
		project = c.cache.sourceProject
	case c.cache.targetNumber:
		project = c.cache.targetProject
	default:
		return fmt.Errorf("project number %d not found in cache", projectNumber)
	}

	// Find the item and its current field value
	itemID, currentValue, err := c.findProjectItem(project, issueURL, field.Name)
	if err != nil {
		return err
	}

	// Check if we need to update the value
	if c.valuesEqual(currentValue, field) {
		return nil
	}

	// Log the field update with previous and new values
	var oldValue, newValue string
	if currentValue != nil {
		switch currentValue.TypeName {
		case "ProjectV2ItemFieldDateValue":
			if currentValue.DateValue.Date != nil {
				oldValue = currentValue.DateValue.Date.Time.Format("2006-01-02")
			}
		case "ProjectV2ItemFieldSingleSelectValue":
			if currentValue.SingleSelectValue.Name != nil {
				oldValue = *currentValue.SingleSelectValue.Name
			}
		}
	}
	if field.Value.Date != nil {
		newValue = field.Value.Date.Format("2006-01-02")
	} else if field.Value.Text != nil {
		newValue = *field.Value.Text
	}
	slog.Info("updating field value",
		"field", field.Name,
		"old", oldValue,
		"new", newValue,
	)

	// Find the field configuration
	fieldID, isDateField, err := c.findProjectField(project, field.Name)
	if err != nil {
		return err
	}

	// Construct the mutation input
	input, err := c.constructMutationInput(project.ID, itemID, fieldID, field, isDateField)
	if err != nil {
		return err
	}

	// Execute the mutation
	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	if err := c.client.Mutate(ctx, &mutation, input, nil); err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	// Update the cache with the new value
	c.updateCacheFieldValue(project, issueURL, field)

	return nil
}

// GetProjectIssues implements the Client interface
func (c *GraphQLClient) GetProjectIssues(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int) ([]string, error) {
	// Use cached data if available
	if c.cache.sourceProject != nil && c.cache.targetProject != nil {
		var issues []string
		project := c.cache.sourceProject
		if projectNumber == c.cache.targetNumber {
			project = c.cache.targetProject
		}
		for _, item := range project.Items.Nodes {
			if item.Content.TypeName == "Issue" {
				issues = append(issues, item.Content.Issue.URL)
			}
		}
		return issues, nil
	}

	// Fall back to fetching data if not cached
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

	var issues []string
	for _, item := range project.Items.Nodes {
		if item.Content.TypeName == "Issue" {
			issues = append(issues, item.Content.Issue.URL)
		}
	}

	return issues, nil
}

// GetProjectFieldConfigsAndIssues retrieves field configurations and issues for both projects
func (c *GraphQLClient) GetProjectFieldConfigsAndIssues(ctx context.Context, ownerType OwnerType, ownerLogin string, sourceProjectNumber, targetProjectNumber int) (sourceConfigs []ProjectFieldConfig, targetConfigs []ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error) {
	// Store project numbers in cache
	c.cache.sourceNumber = sourceProjectNumber
	c.cache.targetNumber = targetProjectNumber

	var query struct {
		User struct {
			SourceProject struct {
				ID     string
				Fields struct {
					Nodes []ProjectV2FieldConfiguration
				} `graphql:"fields(first: 100)"`
				Items struct {
					Nodes []struct {
						ID      string
						Content struct {
							TypeName string `graphql:"__typename"`
							Issue    struct {
								URL string
							} `graphql:"... on Issue"`
						}
						Fields struct {
							Nodes []ProjectV2ItemFieldValue
						} `graphql:"fieldValues(first: 100)"`
					}
				} `graphql:"items(first: 100)"`
			} `graphql:"sourceProject: projectV2(number: $sourceProjectNumber)"`
			TargetProject struct {
				ID     string
				Fields struct {
					Nodes []ProjectV2FieldConfiguration
				} `graphql:"fields(first: 100)"`
				Items struct {
					Nodes []struct {
						ID      string
						Content struct {
							TypeName string `graphql:"__typename"`
							Issue    struct {
								URL string
							} `graphql:"... on Issue"`
						}
						Fields struct {
							Nodes []ProjectV2ItemFieldValue
						} `graphql:"fieldValues(first: 100)"`
					}
				} `graphql:"items(first: 100)"`
			} `graphql:"targetProject: projectV2(number: $targetProjectNumber)"`
		} `graphql:"user(login: $login)"`
	}

	// Execute the query with field aliases
	if err := c.client.Query(ctx, &query, map[string]interface{}{
		"login":               githubv4.String(ownerLogin),
		"sourceProjectNumber": githubv4.Int(sourceProjectNumber),
		"targetProjectNumber": githubv4.Int(targetProjectNumber),
	}); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to query projects: %w", err)
	}

	// Cache the project data
	c.cache.sourceProject = &ProjectV2{
		ID: query.User.SourceProject.ID,
		Fields: struct{ Nodes []ProjectV2FieldConfiguration }{
			Nodes: query.User.SourceProject.Fields.Nodes,
		},
		Items: struct{ Nodes []ProjectV2Item }{
			Nodes: make([]ProjectV2Item, len(query.User.SourceProject.Items.Nodes)),
		},
	}
	for i, item := range query.User.SourceProject.Items.Nodes {
		c.cache.sourceProject.Items.Nodes[i] = ProjectV2Item{
			ID:      item.ID,
			Content: item.Content,
			Fields: struct{ Nodes []ProjectV2ItemFieldValue }{
				Nodes: item.Fields.Nodes,
			},
		}
	}

	c.cache.targetProject = &ProjectV2{
		ID: query.User.TargetProject.ID,
		Fields: struct{ Nodes []ProjectV2FieldConfiguration }{
			Nodes: query.User.TargetProject.Fields.Nodes,
		},
		Items: struct{ Nodes []ProjectV2Item }{
			Nodes: make([]ProjectV2Item, len(query.User.TargetProject.Items.Nodes)),
		},
	}
	for i, item := range query.User.TargetProject.Items.Nodes {
		c.cache.targetProject.Items.Nodes[i] = ProjectV2Item{
			ID:      item.ID,
			Content: item.Content,
			Fields: struct{ Nodes []ProjectV2ItemFieldValue }{
				Nodes: item.Fields.Nodes,
			},
		}
	}

	// Extract field configurations and issues
	sourceConfigs = extractFieldConfigs(query.User.SourceProject.Fields.Nodes)
	targetConfigs = extractFieldConfigs(query.User.TargetProject.Fields.Nodes)

	// Extract issues with modified extractIssueURLs function
	sourceIssues = make([]string, 0)
	targetIssues = make([]string, 0)
	for _, node := range query.User.SourceProject.Items.Nodes {
		if node.Content.TypeName == "Issue" {
			sourceIssues = append(sourceIssues, node.Content.Issue.URL)
		}
	}
	for _, node := range query.User.TargetProject.Items.Nodes {
		if node.Content.TypeName == "Issue" {
			targetIssues = append(targetIssues, node.Content.Issue.URL)
		}
	}

	return sourceConfigs, targetConfigs, sourceIssues, targetIssues, nil
}

// extractFieldConfigs extracts field configurations from nodes
func extractFieldConfigs(nodes []ProjectV2FieldConfiguration) []ProjectFieldConfig {
	var configs []ProjectFieldConfig
	for _, field := range nodes {
		switch field.TypeName {
		case "ProjectV2Field":
			configs = append(configs, ProjectFieldConfig{
				ID:   field.DateField.ID,
				Name: field.DateField.Name,
				Type: field.TypeName,
			})
		case "ProjectV2SingleSelectField":
			configs = append(configs, ProjectFieldConfig{
				ID:   field.SingleSelectField.ID,
				Name: field.SingleSelectField.Name,
				Type: field.TypeName,
			})
		}
	}
	return configs
}

// convertFieldValue converts a ProjectV2ItemFieldValue to our internal ProjectField format
func (c *GraphQLClient) convertFieldValue(fieldValue ProjectV2ItemFieldValue) (ProjectField, bool) {
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

	return field, field.ID != ""
}

// findTargetItem finds an item in a project by its issue URL
func (c *GraphQLClient) findTargetItem(project *ProjectV2, issueURL string) (*ProjectV2Item, error) {
	for _, item := range project.Items.Nodes {
		if item.Content.TypeName == "Issue" && item.Content.Issue.URL == issueURL {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("issue %s not found in project", issueURL)
}

// GetProjectFieldValues implements the Client interface
func (c *GraphQLClient) GetProjectFieldValues(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int, issueURL string, fieldConfigs []ProjectFieldConfig) ([]ProjectField, error) {
	// Use cached data if available
	if c.cache.sourceProject != nil && c.cache.targetProject != nil {
		project := c.cache.sourceProject
		if projectNumber == c.cache.targetNumber {
			project = c.cache.targetProject
		}

		// Find the item (issue) in the project
		targetItem, err := c.findTargetItem(project, issueURL)
		if err != nil {
			return nil, err
		}

		// Convert field values to our internal format
		var fields []ProjectField
		for _, fieldValue := range targetItem.Fields.Nodes {
			if field, ok := c.convertFieldValue(fieldValue); ok {
				fields = append(fields, field)
			}
		}

		return fields, nil
	}

	// Fall back to fetching data if not cached
	var project *ProjectV2
	var err error

	switch ownerType {
	case OwnerTypeUser:
		project, err = c.getUserProjectItems(ctx, ownerLogin, projectNumber)
	case OwnerTypeOrg:
		project, err = c.getOrgProjectItems(ctx, ownerLogin, projectNumber)
	default:
		return nil, fmt.Errorf("invalid owner type")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Find the item (issue) in the project
	targetItem, err := c.findTargetItem(project, issueURL)
	if err != nil {
		return nil, err
	}

	// Convert field values to our internal format
	var fields []ProjectField
	for _, fieldValue := range targetItem.Fields.Nodes {
		if field, ok := c.convertFieldValue(fieldValue); ok {
			fields = append(fields, field)
		}
	}

	return fields, nil
}

// getUserProjectItems gets only the items and their field values for a user project
func (c *GraphQLClient) getUserProjectItems(ctx context.Context, username string, projectNumber int) (*ProjectV2, error) {
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

// getOrgProjectItems gets only the items and their field values for an org project
func (c *GraphQLClient) getOrgProjectItems(ctx context.Context, orgName string, projectNumber int) (*ProjectV2, error) {
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
