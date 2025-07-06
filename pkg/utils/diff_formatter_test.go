package utils

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testDiff = `diff --git a/src/main.go b/src/main.go
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
diff --git a/deleted.txt b/deleted.txt
deleted file mode 100644
index abcdef..0000000
--- a/deleted.txt
+++ /dev/null
@@ -1,2 +0,0 @@
-This file will be deleted
-Second line
`

func TestExtractChangedFiles(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected []string
	}{
		{
			name:     "multiple files",
			diff:     testDiff,
			expected: []string{"src/main.go", "README.md", "deleted.txt"},
		},
		{
			name:     "single file",
			diff:     "diff --git a/test.go b/test.go\nindex 123..456 100644",
			expected: []string{"test.go"},
		},
		{
			name:     "new file",
			diff:     "diff --git a/dev/null b/new.txt\nindex 000..123 100644",
			expected: []string{"new.txt"},
		},
		{
			name:     "deleted file",
			diff:     "diff --git a/old.txt b/dev/null\nindex 123..000 100644",
expected: []string{"dev/null"},
		},
		{
			name:     "empty diff",
			diff:     "",
			expected: []string{},
		},
		{
			name:     "malformed diff",
			diff:     "some random text\nwithout proper diff format",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractChangedFiles(tt.diff)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFilterDiffByFile(t *testing.T) {
	tests := []struct {
		name       string
		diff       string
		targetFile string
		expected   string
	}{
		{
			name:       "filter specific file",
			diff:       testDiff,
			targetFile: "main.go",
			expected:   "diff --git a/src/main.go b/src/main.go\nindex 1234567..abcdefg 100644\n--- a/src/main.go\n+++ b/src/main.go\n@@ -1,7 +1,7 @@\n package main\n \n import (\n-\t\"fmt\"\n+\t\"log\"\n \t\"os\"\n )\n \n@@ -10,5 +10,5 @@ func main() {\n \t\tos.Exit(1)\n \t}\n \n-\tfmt.Println(\"Hello, World!\")\n+\tlog.Println(\"Hello, World!\")\n }\n",
		},
		{
			name:       "filter non-existent file",
			diff:       testDiff,
			targetFile: "nonexistent.go",
			expected:   "",
		},
		{
			name:       "filter README",
			diff:       testDiff,
			targetFile: "README",
			expected:   "diff --git a/README.md b/README.md\nindex 9876543..fedcba9 100644\n--- a/README.md\n+++ b/README.md\n@@ -1,3 +1,4 @@\n # Sample Project\n \n This is a sample project.\n+Added a new line here.\n",
		},
		{
			name:       "empty diff",
			diff:       "",
			targetFile: "any.file",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterDiffByFile(tt.diff, tt.targetFile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanDiffForPatch(t *testing.T) {
	dirtyDiff := `HTTP/1.1 200 OK
Content-Type: text/plain
X-Custom-Header: value

diff --git a/test.go b/test.go
index 123..456 100644
--- a/test.go
+++ b/test.go
@@ -1,3 +1,3 @@
 package main
-fmt.Println("old")
+fmt.Println("new")
`

	expected := `diff --git a/test.go b/test.go
index 123..456 100644
--- a/test.go
+++ b/test.go
@@ -1,3 +1,3 @@
 package main
-fmt.Println("old")
+fmt.Println("new")
`

	result := CleanDiffForPatch(dirtyDiff)
	assert.Equal(t, expected, result)
}

func TestCalculateDiffStats(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected DiffStats
	}{
		{
			name: "basic diff",
			diff: testDiff,
			expected: DiffStats{
				FilesChanged: 3,
				LinesAdded:   3,
				LinesRemoved: 4,
			},
		},
		{
			name: "only additions",
			diff: `diff --git a/new.txt b/new.txt
+++ b/new.txt
@@ -0,0 +1,3 @@
+Line 1
+Line 2
+Line 3`,
			expected: DiffStats{
				FilesChanged: 1,
				LinesAdded:   3,
				LinesRemoved: 0,
			},
		},
		{
			name: "only deletions",
			diff: `diff --git a/old.txt b/old.txt
--- a/old.txt
@@ -1,2 +0,0 @@
-Line 1
-Line 2`,
			expected: DiffStats{
				FilesChanged: 1,
				LinesAdded:   0,
				LinesRemoved: 2,
			},
		},
		{
			name: "empty diff",
			diff: "",
			expected: DiffStats{
				FilesChanged: 0,
				LinesAdded:   0,
				LinesRemoved: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateDiffStats(tt.diff)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDiff(t *testing.T) {
	simpleDiff := `diff --git a/test.go b/test.go
index 123..456 100644
--- a/test.go
+++ b/test.go
@@ -1,2 +1,2 @@
-old line
+new line
`


	result := FormatDiff(simpleDiff, false)
	assert.Equal(t, simpleDiff, result)


	colorResult := FormatDiff(simpleDiff, true)
	assert.NotEqual(t, simpleDiff, colorResult)
assert.Contains(t, colorResult, "diff --git")
}

func TestFormatFilePath(t *testing.T) {
	tests := []struct {
		name     string
		fromPath string
		toPath   string
		expected string
	}{
		{
			name:     "file rename",
			fromPath: "old.txt",
			toPath:   "new.txt",
			expected: "old.txt → new.txt",
		},
		{
			name:     "new file",
			fromPath: "/dev/null",
			toPath:   "new.txt",
			expected: "new.txt (new)",
		},
		{
			name:     "deleted file",
			fromPath: "old.txt",
			toPath:   "/dev/null",
			expected: "old.txt (deleted)",
		},
		{
			name:     "same file",
			fromPath: "same.txt",
			toPath:   "same.txt",
			expected: "same.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatFilePath(tt.fromPath, tt.toPath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseHunkHeader(t *testing.T) {
	tests := []struct {
		name               string
		line               string
		fromStart, toStart int
		fromCount, toCount int
		wantErr            bool
	}{
		{
			name:      "basic hunk",
			line:      "@@ -1,5 +1,7 @@ func main() {",
			fromStart: 1,
			fromCount: 5,
			toStart:   1,
			toCount:   7,
		},
		{
			name:      "single line changes",
			line:      "@@ -10 +10 @@",
			fromStart: 10,
			fromCount: 1,
			toStart:   10,
			toCount:   1,
		},
		{
			name:    "invalid format",
			line:    "not a hunk header",
			wantErr: true,
		},
		{
			name:      "context in header",
			line:      "@@ -15,3 +15,4 @@ package main",
			fromStart: 15,
			fromCount: 3,
			toStart:   15,
			toCount:   4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fromStart, fromCount, toStart, toCount, err := ParseHunkHeader(tt.line)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.fromStart, fromStart)
			assert.Equal(t, tt.fromCount, fromCount)
			assert.Equal(t, tt.toStart, toStart)
			assert.Equal(t, tt.toCount, toCount)
		})
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		name     string
		diff     string
		expected bool
	}{
		{
			name:     "binary file marker",
			diff:     "Binary files a/image.png and b/image.png differ\n",
			expected: true,
		},
		{
			name:     "git binary patch",
			diff:     "GIT binary patch\ndelta 123\n",
			expected: true,
		},
		{
			name:     "text file",
			diff:     "diff --git a/text.txt b/text.txt\n+line addition",
			expected: false,
		},
		{
			name:     "empty diff",
			diff:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsBinaryFile(tt.diff)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSplitDiffIntoFiles(t *testing.T) {
	result := SplitDiffIntoFiles(testDiff)


	assert.Len(t, result, 3)


	assert.Contains(t, result, "src/main.go")
	assert.Contains(t, result, "README.md")
	assert.Contains(t, result, "deleted.txt")


	mainGoDiff := result["src/main.go"]
	assert.Contains(t, mainGoDiff, "diff --git a/src/main.go b/src/main.go")
	assert.Contains(t, mainGoDiff, "-\t\"fmt\"")
	assert.Contains(t, mainGoDiff, "+\t\"log\"")

	readmeDiff := result["README.md"]
	assert.Contains(t, readmeDiff, "diff --git a/README.md b/README.md")
	assert.Contains(t, readmeDiff, "+Added a new line here.")
}

func TestIsTerminal(t *testing.T) {


	

	result := IsTerminal(os.Stdin)
assert.IsType(t, bool(false), result)


	result = IsTerminal(os.Stdout)
assert.IsType(t, bool(false), result)


	result = IsTerminal(os.Stderr)
assert.IsType(t, bool(false), result)
}


func BenchmarkExtractChangedFiles(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ExtractChangedFiles(testDiff)
	}
}

func BenchmarkFilterDiffByFile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FilterDiffByFile(testDiff, "main.go")
	}
}

func BenchmarkCalculateDiffStats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CalculateDiffStats(testDiff)
	}
}

func BenchmarkFormatDiff(b *testing.B) {
	for i := 0; i < b.N; i++ {
		FormatDiff(testDiff, true)
	}
}


func TestEdgeCases(t *testing.T) {
	t.Run("very large diff", func(t *testing.T) {

		var largeDiff strings.Builder
		largeDiff.WriteString("diff --git a/large.txt b/large.txt\n")
		largeDiff.WriteString("--- a/large.txt\n")
		largeDiff.WriteString("+++ b/large.txt\n")
		largeDiff.WriteString("@@ -1,1000 +1,1001 @@\n")
		
		for i := 0; i < 1000; i++ {
			largeDiff.WriteString("-line " + string(rune(i)) + "\n")
		}
		for i := 0; i < 1001; i++ {
			largeDiff.WriteString("+line " + string(rune(i)) + "\n")
		}

		files := ExtractChangedFiles(largeDiff.String())
		assert.Equal(t, []string{"large.txt"}, files)

		stats := CalculateDiffStats(largeDiff.String())
		assert.Equal(t, 1, stats.FilesChanged)
		assert.Equal(t, 1001, stats.LinesAdded)
		assert.Equal(t, 1000, stats.LinesRemoved)
	})

	t.Run("malformed diff headers", func(t *testing.T) {
		malformedDiff := `diff --git
index incomplete
--- a/file.txt
not a proper header
+++ b/file.txt
@@ invalid hunk @@
+some content
`
		files := ExtractChangedFiles(malformedDiff)
assert.Empty(t, files)
	})

	t.Run("unicode content", func(t *testing.T) {
		unicodeDiff := `diff --git a/unicode.txt b/unicode.txt
--- a/unicode.txt
+++ b/unicode.txt
@@ -1,2 +1,2 @@
-Hello 世界
+Hello 世界! 🌍
`
		files := ExtractChangedFiles(unicodeDiff)
		assert.Equal(t, []string{"unicode.txt"}, files)

		stats := CalculateDiffStats(unicodeDiff)
		assert.Equal(t, 1, stats.FilesChanged)
		assert.Equal(t, 1, stats.LinesAdded)
		assert.Equal(t, 1, stats.LinesRemoved)
	})
}
