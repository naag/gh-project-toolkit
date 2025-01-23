package projecturl

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/naag/gh-project-toolkit/internal/github"
)

// ProjectInfo contains the parsed information from a GitHub project URL
type ProjectInfo struct {
	OwnerType     github.OwnerType
	OwnerLogin    string
	ProjectNumber int
}

// Parse takes a GitHub project URL and returns the parsed ProjectInfo
func Parse(projectURL string) (*ProjectInfo, error) {
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
	var ownerType github.OwnerType
	switch parts[0] {
	case "orgs":
		ownerType = github.OwnerTypeOrg
	case "users":
		ownerType = github.OwnerTypeUser
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

	return &ProjectInfo{
		OwnerType:     ownerType,
		OwnerLogin:    parts[1],
		ProjectNumber: projectNum,
	}, nil
}
