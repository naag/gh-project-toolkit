package util

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/naag/gh-project-toolkit/internal/github"
)

func ParseProjectURL(projectURL string) (*github.ProjectInfo, error) {
	u, err := url.Parse(projectURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if u.Host != "github.com" {
		return nil, fmt.Errorf("not a GitHub URL")
	}

	// Split path into components
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid project URL format")
	}

	// Check if it's an org or user project
	var ownerType github.ProjectOwnerType
	switch parts[0] {
	case "orgs":
		ownerType = github.ProjectOwnerTypeOrg
	case "users":
		ownerType = github.ProjectOwnerTypeUser
	default:
		return nil, fmt.Errorf("invalid owner type in URL: %s", parts[0])
	}

	// Parse project number
	if parts[2] != "projects" {
		return nil, fmt.Errorf("invalid URL format: expected 'projects' as third component")
	}

	projectNum, err := strconv.Atoi(parts[3])
	if err != nil {
		return nil, fmt.Errorf("invalid project number: %w", err)
	}

	return &github.ProjectInfo{
		OwnerType:     ownerType,
		OwnerLogin:    parts[1],
		ProjectNumber: projectNum,
	}, nil
}
