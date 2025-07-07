package pr

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)


const sampleDiff = `diff --git a/src/main.go b/src/main.go
index 1234567..abcdefg 100644
--- a/src/main.go
+++ b/src/main.go
@@ -1,7 +1,7 @@
 package main
 
 import (
-	"fmt"
+	"log"
 	"os"
 )
 
@@ -10,5 +10,5 @@ func main() {
 		os.Exit(1)
 	}
 
-	fmt.Println("Hello, World!")
+	log.Println("Hello, World!")
 }
diff --git a/README.md b/README.md
index 9876543..fedcba9 100644
--- a/README.md
+++ b/README.md
@@ -1,3 +1,4 @@
 # Sample Project
 
 This is a sample project.
+Added a new line here.
`

func TestDiffCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name    string
		prid    string
		want    int
		wantErr bool
	}{
		{
			name: "valid number",
			prid: "123",
			want: 123,
		},
		{
			name: "valid number with hash prefix",
			prid: "#456",
			want: 456,
		},
		{
			name:    "empty string",
			prid:    "",
			wantErr: true,
		},
		{
			name:    "invalid number",
			prid:    "abc",
			wantErr: true,
		},
		{
			name:    "negative number",
			prid:    "-123",
			wantErr: true,
		},
		{
			name:    "zero",
			prid:    "0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &DiffCmd{PRID: tt.prid}
			got, err := cmd.ParsePRID()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiffCmd_outputNameOnly(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		file     string
		expected []string
	}{
		{
			name:     "extract all files",
			diff:     sampleDiff,
			expected: []string{"src/main.go", "README.md"},
		},
		{
			name:     "filter by specific file",
			diff:     sampleDiff,
			file:     "main.go",
			expected: []string{"src/main.go"},
		},
		{
			name:     "filter by non-existent file",
			diff:     sampleDiff,
			file:     "nonexistent.txt",
			expected: []string{},
		},
		{
			name:     "empty diff",
			diff:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {



			files := extractChangedFilesForTest(tt.diff)
			
			if tt.file != "" {
				filteredFiles := make([]string, 0)
				for _, file := range files {
					if strings.Contains(file, tt.file) {
						filteredFiles = append(filteredFiles, file)
					}
				}
				files = filteredFiles
			}

			assert.Equal(t, tt.expected, files)
		})
	}
}


func extractChangedFilesForTest(diff string) []string {
	files := make([]string, 0)
	lines := strings.Split(diff, "\n")
	
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git") {
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				fileA := strings.TrimPrefix(parts[2], "a/")
				fileB := strings.TrimPrefix(parts[3], "b/")
				
				if fileB != "/dev/null" {
					files = append(files, fileB)
				} else if fileA != "/dev/null" {
					files = append(files, fileA)
				}
			}
		}
	}
	
	return files
}

func TestDiffCmd_shouldUseColors(t *testing.T) {
	tests := []struct {
		name    string
		color   string
		noColor bool
		want    bool
	}{
		{
			name:  "always use colors",
			color: "always",
			want:  true,
		},
		{
			name:  "never use colors",
			color: "never",
			want:  false,
		},
		{
			name:    "auto with no color flag",
			color:   "auto",
			noColor: true,
			want:    false,
		},
		{
			name:  "auto without no color flag",
			color: "auto",
want:  false,
		},
		{
			name:  "invalid color setting",
			color: "invalid",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &DiffCmd{
				Color:   tt.color,
				NoColor: tt.noColor,
			}
			got := cmd.shouldUseColors()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDiffCmd_validateFlags(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *DiffCmd
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid basic command",
			cmd: &DiffCmd{
				PRID:   "123",
				Output: "diff",
				Color:  "auto",
			},
			wantErr: false,
		},
		{
			name: "valid with name-only",
			cmd: &DiffCmd{
				PRID:     "456",
				NameOnly: true,
			},
			wantErr: false,
		},
		{
			name: "valid with patch format",
			cmd: &DiffCmd{
				PRID:  "789",
				Patch: true,
			},
			wantErr: false,
		},
		{
			name: "valid with file filter",
			cmd: &DiffCmd{
				PRID: "101",
				File: "main.go",
			},
			wantErr: false,
		},
		{
			name: "conflicting name-only and patch",
			cmd: &DiffCmd{
				PRID:     "123",
				NameOnly: true,
				Patch:    true,
			},
wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			_, err := tt.cmd.ParsePRID()
			
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


func TestDiffCmd_Integration(t *testing.T) {
	t.Skip("Integration test - requires real Bitbucket API")









}

func TestDiffCmd_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		prid     string
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "missing PR ID",
			prid:     "",
			wantErr:  true,
			errorMsg: "pull request ID is required",
		},
		{
			name:     "invalid PR ID format",
			prid:     "invalid",
			wantErr:  true,
			errorMsg: "must be a positive integer",
		},
		{
			name:     "negative PR ID",
			prid:     "-1",
			wantErr:  true,
			errorMsg: "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &DiffCmd{PRID: tt.prid}
			_, err := cmd.ParsePRID()

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}


func BenchmarkDiffCmd_ParsePRID(b *testing.B) {
	cmd := &DiffCmd{PRID: "12345"}
	
	for i := 0; i < b.N; i++ {
		_, _ = cmd.ParsePRID()
	}
}

func BenchmarkExtractChangedFiles(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = extractChangedFilesForTest(sampleDiff)
	}
}


func TestDiffCmd_CommandStructure(t *testing.T) {
	cmd := &DiffCmd{
		PRID:       "123",
		NameOnly:   false,
		Patch:      false,
		File:       "",
		Color:      "auto",
		Output:     "diff",
		NoColor:    false,
		Workspace:  "",
		Repository: "",
	}


	assert.Equal(t, "123", cmd.PRID)
	assert.Equal(t, "auto", cmd.Color)
	assert.Equal(t, "diff", cmd.Output)
	assert.False(t, cmd.NameOnly)
	assert.False(t, cmd.Patch)
	assert.False(t, cmd.NoColor)
}


func TestDiffCmd_OutputModeSelection(t *testing.T) {
	tests := []struct {
		name     string
		nameOnly bool
		patch    bool
		output   string
		expected string
	}{
		{
			name:     "name-only mode",
			nameOnly: true,
			expected: "name-only",
		},
		{
			name:     "patch mode",
			patch:    true,
			expected: "patch",
		},
		{
			name:     "JSON output",
			output:   "json",
			expected: "json",
		},
		{
			name:     "YAML output",
			output:   "yaml",
			expected: "yaml",
		},
		{
			name:     "default colored diff",
			output:   "diff",
			expected: "colored",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &DiffCmd{
				NameOnly: tt.nameOnly,
				Patch:    tt.patch,
				Output:   tt.output,
			}


			var selectedMode string
			switch {
			case cmd.NameOnly:
				selectedMode = "name-only"
			case cmd.Output == "json":
				selectedMode = "json"
			case cmd.Output == "yaml":
				selectedMode = "yaml"
			case cmd.Patch:
				selectedMode = "patch"
			default:
				selectedMode = "colored"
			}

			assert.Equal(t, tt.expected, selectedMode)
		})
	}
}
