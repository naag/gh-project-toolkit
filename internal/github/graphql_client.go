package github

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
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
		issueTitles   map[string]string // map of issue URL to title
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

	client := &GraphQLClient{
		client: githubv4.NewClient(httpClient),
	}
	client.cache.issueTitles = make(map[string]string)
	return client, nil
}

// ProjectV2 represents a GitHub project (v2)
type ProjectV2 struct {
	ID     string
	Fields struct {
		Nodes []ProjectV2FieldConfiguration
	} `graphql:"fields(first: 100)"`
	Items struct {
		Nodes    []ProjectV2Item
		PageInfo struct {
			HasNextPage bool
			EndCursor   string
		}
	} `graphql:"items(first: 100, after: $afterCursor)"`
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
				URL   string
				Title string
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

	slog.Debug("loading organization project ID", "org", orgName, "number", projectNumber)

	var query struct {
		Organization struct {
			Project struct {
				ID string
			} `graphql:"projectV2(number: $projectNumber)"`
		} `graphql:"organization(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":         githubv4.String(orgName),
		"projectNumber": githubv4.Int(projectNumber),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query organization project: %w", err)
	}

	return &ProjectV2{ID: query.Organization.Project.ID}, nil
}

func (c *GraphQLClient) getUserProject(ctx context.Context, username string, projectNumber int) (*ProjectV2, error) {
	if projectNumber <= 0 {
		return nil, fmt.Errorf("invalid project number: %d", projectNumber)
	}

	slog.Debug("loading user project ID", "user", username, "number", projectNumber)

	var query struct {
		User struct {
			Project struct {
				ID string
			} `graphql:"projectV2(number: $projectNumber)"`
		} `graphql:"user(login: $login)"`
	}

	variables := map[string]interface{}{
		"login":         githubv4.String(username),
		"projectNumber": githubv4.Int(projectNumber),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query user project: %w", err)
	}

	return &ProjectV2{ID: query.User.Project.ID}, nil
}

// GetProjectFields implements the Client interface
func (c *GraphQLClient) GetProjectFields(ctx context.Context, projectID string, issueURL string) ([]ProjectField, error) {
	var query struct {
		Node struct {
			Project ProjectV2 `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectID)"`
	}

	variables := map[string]interface{}{
		"projectID": githubv4.ID(projectID),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	project := &query.Node.Project

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
			field = ProjectField{
				ID:   fieldValue.DateValue.Field.DateField.ID,
				Name: fieldValue.DateValue.Field.DateField.Name,
				Value: FieldValue{
					Date: &fieldValue.DateValue.Date.Time,
				},
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
		// Find the option ID for the single select value in the target project
		var optionID string
		var project *ProjectV2
		if c.cache.targetProject != nil && c.cache.targetProject.ID == projectID {
			project = c.cache.targetProject
		}
		if project == nil {
			return input, fmt.Errorf("target project not found in cache")
		}

		for _, f := range project.Fields.Nodes {
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
			return input, fmt.Errorf("single select option %q not found in target field %q", *field.Value.Text, field.Name)
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

// getProjectFromCache retrieves a project from cache if available
func (c *GraphQLClient) getProjectFromCache(projectID string) *ProjectV2 {
	if c.cache.sourceProject != nil && c.cache.sourceProject.ID == projectID {
		return c.cache.sourceProject
	}
	if c.cache.targetProject != nil && c.cache.targetProject.ID == projectID {
		return c.cache.targetProject
	}
	return nil
}

// fetchProject fetches a project by ID
func (c *GraphQLClient) fetchProject(ctx context.Context, projectID string) (*ProjectV2, error) {
	var query struct {
		Node struct {
			Project ProjectV2 `graphql:"... on ProjectV2"`
		} `graphql:"node(id: $projectID)"`
	}

	variables := map[string]interface{}{
		"projectID": githubv4.ID(projectID),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("failed to query project: %w", err)
	}

	return &query.Node.Project, nil
}

// getFieldUpdateValues gets the old and new values for logging
func (c *GraphQLClient) getFieldUpdateValues(currentValue *ProjectV2ItemFieldValue, field ProjectField) (string, string) {
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
	return oldValue, newValue
}

// executeFieldUpdate executes the field update mutation
func (c *GraphQLClient) executeFieldUpdate(ctx context.Context, input githubv4.UpdateProjectV2ItemFieldValueInput) error {
	var mutation struct {
		UpdateProjectV2ItemFieldValue struct {
			ClientMutationID string
		} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
	}

	if err := c.client.Mutate(ctx, &mutation, input, nil); err != nil {
		return fmt.Errorf("failed to update field: %w", err)
	}

	return nil
}

// UpdateProjectField implements the Client interface
func (c *GraphQLClient) UpdateProjectField(ctx context.Context, projectID string, issueURL string, field ProjectField, dryRun bool) error {
	// Get project from cache or fetch it
	project := c.getProjectFromCache(projectID)
	if project == nil {
		var err error
		project, err = c.fetchProject(ctx, projectID)
		if err != nil {
			return err
		}
	}

	// Find the item and its current field value
	itemID, currentValue, err := c.findProjectItem(project, issueURL, field.Name)
	if err != nil {
		return err
	}

	// Skip update if values are equal
	if c.valuesEqual(currentValue, field) {
		return nil
	}

	// Find the field configuration
	fieldID, isDateField, err := c.findProjectField(project, field.Name)
	if err != nil {
		return err
	}

	// Log the field update
	oldValue, newValue := c.getFieldUpdateValues(currentValue, field)
	slog.Debug("updating field value",
		"field", field.Name,
		"old", oldValue,
		"new", newValue,
		"dry_run", dryRun,
	)

	if !dryRun {
		// Construct and execute the mutation
		input, err := c.constructMutationInput(project.ID, itemID, fieldID, field, isDateField)
		if err != nil {
			return err
		}

		if err := c.executeFieldUpdate(ctx, input); err != nil {
			return err
		}

		// Update the cache with the new value
		c.updateCacheFieldValue(project, issueURL, field)
	}

	return nil
}

// GetProjectIssues implements the Client interface
func (c *GraphQLClient) GetProjectIssues(ctx context.Context, projectID string) ([]string, error) {
	type projectQuery struct {
		Project struct {
			Items struct {
				Nodes    []ProjectV2Item
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
			} `graphql:"items(first: 100, after: $afterCursor)"`
		} `graphql:"... on ProjectV2"`
	}

	var query struct {
		Node projectQuery `graphql:"node(id: $projectID)"`
	}

	var items []ProjectV2Item
	var afterCursor *string
	var page int

	slog.Info("loading project issues from GitHub")

	// Fetch items with pagination
	for {
		page++
		slog.Debug("loading page of issues", "page", page)

		variables := map[string]interface{}{
			"projectID":   githubv4.ID(projectID),
			"afterCursor": (*githubv4.String)(afterCursor),
		}

		if err := c.client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("failed to query project: %w", err)
		}

		items = append(items, query.Node.Project.Items.Nodes...)

		if !query.Node.Project.Items.PageInfo.HasNextPage {
			break
		}

		cursor := query.Node.Project.Items.PageInfo.EndCursor
		afterCursor = &cursor
	}

	var issues []string
	for _, item := range items {
		if item.Content.TypeName == "Issue" {
			issues = append(issues, item.Content.Issue.URL)
			c.cache.issueTitles[item.Content.Issue.URL] = item.Content.Issue.Title
		}
	}

	slog.Info("completed loading project issues", "total_issues", len(issues), "pages_loaded", page)
	return issues, nil
}

// GetProjectFieldConfigsAndIssues implements the Client interface
func (c *GraphQLClient) GetProjectFieldConfigsAndIssues(ctx context.Context, sourceProjectID string, targetProjectID string) (sourceConfigs []ProjectFieldConfig, targetConfigs []ProjectFieldConfig, sourceIssues []string, targetIssues []string, err error) {
	type projectQuery struct {
		Project struct {
			ID     string
			Fields struct {
				Nodes []ProjectV2FieldConfiguration
			} `graphql:"fields(first: 100)"`
			Items struct {
				Nodes    []ProjectV2Item
				PageInfo struct {
					HasNextPage bool
					EndCursor   string
				}
			} `graphql:"items(first: 100, after: $afterCursor)"`
		} `graphql:"... on ProjectV2"`
	}

	var query struct {
		SourceProject projectQuery `graphql:"sourceProject: node(id: $sourceProjectID)"`
		TargetProject projectQuery `graphql:"targetProject: node(id: $targetProjectID)"`
	}

	// Initialize variables for pagination
	var sourceItems []ProjectV2Item
	var targetItems []ProjectV2Item
	var afterCursor *string
	var page int

	slog.Info("loading project data from GitHub")

	// Fetch source project items with pagination
	for {
		page++
		slog.Debug("loading page", "page", page)

		variables := map[string]interface{}{
			"sourceProjectID": githubv4.ID(sourceProjectID),
			"targetProjectID": githubv4.ID(targetProjectID),
			"afterCursor":     (*githubv4.String)(afterCursor),
		}

		if err := c.client.Query(ctx, &query, variables); err != nil {
			return nil, nil, nil, nil, fmt.Errorf("failed to query projects: %w", err)
		}

		sourceItems = append(sourceItems, query.SourceProject.Project.Items.Nodes...)
		targetItems = append(targetItems, query.TargetProject.Project.Items.Nodes...)

		if !query.SourceProject.Project.Items.PageInfo.HasNextPage && !query.TargetProject.Project.Items.PageInfo.HasNextPage {
			break
		}

		// Use the end cursor from either project that has more pages
		if query.SourceProject.Project.Items.PageInfo.HasNextPage {
			cursor := query.SourceProject.Project.Items.PageInfo.EndCursor
			afterCursor = &cursor
		} else if query.TargetProject.Project.Items.PageInfo.HasNextPage {
			cursor := query.TargetProject.Project.Items.PageInfo.EndCursor
			afterCursor = &cursor
		}
	}

	// Cache the project data with all items
	c.cache.sourceProject = &ProjectV2{
		ID: query.SourceProject.Project.ID,
		Fields: struct {
			Nodes []ProjectV2FieldConfiguration
		}{
			Nodes: query.SourceProject.Project.Fields.Nodes,
		},
		Items: struct {
			Nodes    []ProjectV2Item
			PageInfo struct {
				HasNextPage bool
				EndCursor   string
			}
		}{
			Nodes: sourceItems,
		},
	}

	c.cache.targetProject = &ProjectV2{
		ID: query.TargetProject.Project.ID,
		Fields: struct {
			Nodes []ProjectV2FieldConfiguration
		}{
			Nodes: query.TargetProject.Project.Fields.Nodes,
		},
		Items: struct {
			Nodes    []ProjectV2Item
			PageInfo struct {
				HasNextPage bool
				EndCursor   string
			}
		}{
			Nodes: targetItems,
		},
	}

	// Convert field configurations
	for _, field := range query.SourceProject.Project.Fields.Nodes {
		config := ProjectFieldConfig{
			ID:   field.DateField.ID,
			Name: field.DateField.Name,
			Type: field.TypeName,
		}
		sourceConfigs = append(sourceConfigs, config)
	}

	for _, field := range query.TargetProject.Project.Fields.Nodes {
		config := ProjectFieldConfig{
			ID:   field.DateField.ID,
			Name: field.DateField.Name,
			Type: field.TypeName,
		}
		targetConfigs = append(targetConfigs, config)
	}

	// Get issues from all fetched items
	for _, item := range sourceItems {
		if item.Content.TypeName == "Issue" {
			sourceIssues = append(sourceIssues, item.Content.Issue.URL)
			c.cache.issueTitles[item.Content.Issue.URL] = item.Content.Issue.Title
		}
	}

	for _, item := range targetItems {
		if item.Content.TypeName == "Issue" {
			targetIssues = append(targetIssues, item.Content.Issue.URL)
			c.cache.issueTitles[item.Content.Issue.URL] = item.Content.Issue.Title
		}
	}

	slog.Info("completed loading project data",
		"source_issues", len(sourceIssues),
		"target_issues", len(targetIssues),
		"pages_loaded", page,
	)

	return sourceConfigs, targetConfigs, sourceIssues, targetIssues, nil
}

// GetProjectFieldValues implements the Client interface
func (c *GraphQLClient) GetProjectFieldValues(ctx context.Context, projectID string, issueURL string, fieldConfigs []ProjectFieldConfig) ([]ProjectField, error) {
	// Use cached data if available
	var project *ProjectV2
	if c.cache.sourceProject != nil && c.cache.sourceProject.ID == projectID {
		project = c.cache.sourceProject
	} else if c.cache.targetProject != nil && c.cache.targetProject.ID == projectID {
		project = c.cache.targetProject
	}

	if project == nil {
		// Fall back to fetching data if not cached
		var query struct {
			Node struct {
				Project ProjectV2 `graphql:"... on ProjectV2"`
			} `graphql:"node(id: $projectID)"`
		}

		variables := map[string]interface{}{
			"projectID": githubv4.ID(projectID),
		}

		if err := c.client.Query(ctx, &query, variables); err != nil {
			return nil, fmt.Errorf("failed to query project: %w", err)
		}

		project = &query.Node.Project
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
			field = ProjectField{
				ID:   fieldValue.DateValue.Field.DateField.ID,
				Name: fieldValue.DateValue.Field.DateField.Name,
				Value: FieldValue{
					Date: &fieldValue.DateValue.Date.Time,
				},
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

// GetProjectID implements the Client interface
func (c *GraphQLClient) GetProjectID(ctx context.Context, ownerType OwnerType, ownerLogin string, projectNumber int) (string, error) {
	slog.Info("loading project metadata from GitHub")

	var project *ProjectV2
	var err error

	switch ownerType {
	case OwnerTypeUser:
		project, err = c.getUserProject(ctx, ownerLogin, projectNumber)
	case OwnerTypeOrg:
		project, err = c.getOrgProject(ctx, ownerLogin, projectNumber)
	default:
		return "", fmt.Errorf("invalid owner type")
	}

	if err != nil {
		return "", fmt.Errorf("failed to get project: %w", err)
	}

	return project.ID, nil
}

// GetIssueTitle implements the Client interface
func (c *GraphQLClient) GetIssueTitle(ctx context.Context, issueURL string) (string, error) {
	// Check cache first
	if title, ok := c.cache.issueTitles[issueURL]; ok {
		return title, nil
	}

	// If not in cache, fall back to querying GitHub
	parts := strings.Split(issueURL, "/")
	if len(parts) < 7 {
		return "", fmt.Errorf("invalid issue URL format: %s", issueURL)
	}

	owner := parts[3]
	repo := parts[4]
	number := parts[6]

	issueNumber, err := strconv.Atoi(number)
	if err != nil {
		return "", fmt.Errorf("invalid issue number: %s", number)
	}

	slog.Debug("loading issue title from GitHub (cache miss)",
		"owner", owner,
		"repo", repo,
		"number", issueNumber,
	)

	var query struct {
		Repository struct {
			Issue struct {
				Title string
			} `graphql:"issue(number: $issueNumber)"`
		} `graphql:"repository(owner: $owner, name: $repo)"`
	}

	variables := map[string]interface{}{
		"owner":       githubv4.String(owner),
		"repo":        githubv4.String(repo),
		"issueNumber": githubv4.Int(issueNumber),
	}

	if err := c.client.Query(ctx, &query, variables); err != nil {
		return "", fmt.Errorf("failed to query issue: %w", err)
	}

	// Cache the result
	title := query.Repository.Issue.Title
	c.cache.issueTitles[issueURL] = title
	return title, nil
}
