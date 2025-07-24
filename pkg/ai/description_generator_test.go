package ai

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDiff_CorrectParameterOrder(t *testing.T) {
	
	tests := []struct {
		name         string
		targetBranch string
		sourceBranch string
		description  string
	}{
		{
			name:         "PR from feature to main",
			targetBranch: "main",
			sourceBranch: "feature-branch",
			description:  "Shows what changes when merging feature-branch INTO main",
		},
		{
			name:         "PR from develop to staging",
			targetBranch: "staging",
			sourceBranch: "develop",
			description:  "Shows what changes when merging develop INTO staging",
		},
		{
			name:         "PR from ZUP-55-hml to homolog",
			targetBranch: "homolog",
			sourceBranch: "ZUP-55-hml",
			description:  "Shows what changes when merging ZUP-55-hml INTO homolog",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Correct order: GetDiff(%q, %q)", tt.targetBranch, tt.sourceBranch)
			t.Logf("Description: %s", tt.description)
			
			t.Logf("WRONG order would be: GetDiff(%q, %q) - DON'T DO THIS!", tt.sourceBranch, tt.targetBranch)
		})
	}
}

func TestDiffInterpretation_AdditionsVsDeletions(t *testing.T) {
	
	testDiff := `diff --git a/app.py b/app.py
index abc123..def456 100644
--- a/app.py
+++ b/app.py
@@ -50,6 +50,15 @@ def index():
     return render_template('index.html')
 
+def get_api_base_url():
+    """Determine API base URL based on environment"""
+    host = request.headers.get('Host', '')
+    if 'localhost' in host:
+        return 'http://localhost:8000'
+    elif 'hml' in host:
+        return 'https://api-hml.example.com'
+    else:
+        return 'https://api.example.com'
+
 @app.route('/validate_pdf', methods=['POST'])
 def validate_pdf():`
	
	assert.True(t, strings.Contains(testDiff, "+def get_api_base_url():"), "Should contain added function")
	assert.True(t, strings.Contains(testDiff, "+    \"\"\"Determine API base URL based on environment\"\"\""), "Should contain added docstring")
	
	lines := strings.Split(testDiff, "\n")
	additionCount := 0
	deletionCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additionCount++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletionCount++
		}
	}
	assert.Equal(t, 10, additionCount, "Should have 10 lines added (9 function lines + 1 blank line)")
	assert.Equal(t, 0, deletionCount, "Should have 0 lines deleted")
}

func TestDiffInterpretation_WrongDirection(t *testing.T) {
	
	wrongDiff := `diff --git a/app.py b/app.py
index def456..abc123 100644
--- a/app.py
+++ b/app.py
@@ -50,15 +50,6 @@ def index():
     return render_template('index.html')
 
-def get_api_base_url():
-    """Determine API base URL based on environment"""
-    host = request.headers.get('Host', '')
-    if 'localhost' in host:
-        return 'http://localhost:8000'
-    elif 'hml' in host:
-        return 'https://api-hml.example.com'
-    else:
-        return 'https://api.example.com'
-
 @app.route('/validate_pdf', methods=['POST'])
 def validate_pdf():`
	
	assert.True(t, strings.Contains(wrongDiff, "-def get_api_base_url():"), "Function appears as deletion with wrong order")
	
	t.Log("With wrong parameter order, the AI sees additions as deletions!")
	t.Log("This is why it said 'removes functionality' when code was being added")
}
