package projecturl

import (
	"testing"

	"github.com/naag/gh-project-toolkit/internal/github"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *ProjectInfo
		wantErr string
	}{
		{
			name: "valid org project URL",
			url:  "https://github.com/orgs/testorg/projects/123",
			want: &ProjectInfo{
				OwnerType:     github.OwnerTypeOrg,
				OwnerLogin:    "testorg",
				ProjectNumber: 123,
			},
		},
		{
			name: "valid user project URL",
			url:  "https://github.com/users/testuser/projects/456",
			want: &ProjectInfo{
				OwnerType:     github.OwnerTypeUser,
				OwnerLogin:    "testuser",
				ProjectNumber: 456,
			},
		},
		{
			name:    "invalid URL",
			url:     ":", // Using a clearly invalid URL
			wantErr: "invalid URL",
		},
		{
			name:    "non-GitHub URL",
			url:     "https://gitlab.com/orgs/test/projects/123",
			wantErr: "not a GitHub URL",
		},
		{
			name:    "invalid path format",
			url:     "https://github.com/orgs/test/wrong/123",
			wantErr: "invalid URL format",
		},
		{
			name:    "invalid owner type",
			url:     "https://github.com/wrong/test/projects/123",
			wantErr: "invalid owner type",
		},
		{
			name:    "invalid project number",
			url:     "https://github.com/orgs/test/projects/abc",
			wantErr: "invalid project number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.url)
			if tt.wantErr != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
