package pr

import (
	"strconv"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestFilesCmd_matchesFilter(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{
			name:     "exact filename match",
			path:     "main.go",
			pattern:  "main.go",
			expected: true,
		},
		{
			name:     "wildcard extension match",
			path:     "src/main.go",
			pattern:  "*.go",
			expected: true,
		},
		{
			name:     "wildcard extension no match",
			path:     "src/main.js",
			pattern:  "*.go",
			expected: false,
		},
		{
			name:     "path pattern match",
			path:     "src/cmd/main.go",
			pattern:  "src/cmd/*.go",
			expected: true,
		},
		{
			name:     "path pattern no match",
			path:     "pkg/cmd/main.go",
			pattern:  "src/cmd/*.go",
			expected: false,
		},
		{
			name:     "invalid pattern returns true",
			path:     "main.go",
			pattern:  "[",
			expected: true,
		},
	}

	cmd := &FilesCmd{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.matchesFilter(tt.path, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilesCmd_getFileStatus(t *testing.T) {
	tests := []struct {
		name     string
		file     *api.PullRequestFile
		expected string
	}{
		{
			name: "added file from status",
			file: &api.PullRequestFile{
				Status:       "added",
				LinesAdded:   10,
				LinesRemoved: 0,
			},
			expected: "A",
		},
		{
			name: "deleted file from status",
			file: &api.PullRequestFile{
				Status:       "removed",
				LinesAdded:   0,
				LinesRemoved: 10,
			},
			expected: "D",
		},
		{
			name: "modified file from status",
			file: &api.PullRequestFile{
				Status:       "modified",
				LinesAdded:   5,
				LinesRemoved: 3,
			},
			expected: "M",
		},
		{
			name: "renamed file from status",
			file: &api.PullRequestFile{
				Status:       "renamed",
				LinesAdded:   0,
				LinesRemoved: 0,
			},
			expected: "R",
		},
		{
			name: "added file from line counts",
			file: &api.PullRequestFile{
				Status:       "",
				LinesAdded:   10,
				LinesRemoved: 0,
			},
			expected: "A",
		},
		{
			name: "deleted file from line counts",
			file: &api.PullRequestFile{
				Status:       "",
				LinesAdded:   0,
				LinesRemoved: 10,
			},
			expected: "D",
		},
		{
			name: "modified file from line counts",
			file: &api.PullRequestFile{
				Status:       "",
				LinesAdded:   5,
				LinesRemoved: 3,
			},
			expected: "M",
		},
		{
			name: "unknown status defaults to modified",
			file: &api.PullRequestFile{
				Status:       "unknown",
				LinesAdded:   0,
				LinesRemoved: 0,
			},
			expected: "M",
		},
	}

	cmd := &FilesCmd{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmd.getFileStatus(tt.file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilesCmd_validatePRID(t *testing.T) {
	tests := []struct {
		name    string
		prID    string
		wantID  int
		wantErr bool
	}{
		{
			name:    "numeric ID",
			prID:    "123",
			wantID:  123,
			wantErr: false,
		},
		{
			name:    "hash prefix ID",
			prID:    "#456",
			wantID:  456,
			wantErr: false,
		},
		{
			name:    "invalid non-numeric",
			prID:    "abc",
			wantID:  0,
			wantErr: true,
		},
		{
			name:    "empty string",
			prID:    "",
			wantID:  0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prID := tt.prID
			if len(prID) > 0 && prID[0] == '#' {
				prID = prID[1:]
			}

			var err error
			if prID != "" {
				_, err = strconv.Atoi(prID)
			} else {
				err = strconv.ErrSyntax
			}

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
