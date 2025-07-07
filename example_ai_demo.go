package main

import (
	"fmt"

	"github.com/carlosarraes/bt/pkg/ai"
	"github.com/carlosarraes/bt/pkg/utils"
)

func main() {
	fmt.Println("🚀 AI-Powered PR Description Generation Demo")
	fmt.Println("============================================")
	
	fmt.Println("\n📋 Portuguese Template Preview:")
	fmt.Println("-------------------------------")
	
	preview, err := ai.GetTemplatePreview("portuguese")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println(preview)
	
	fmt.Println("\n📋 English Template Preview:")
	fmt.Println("----------------------------")
	
	englishPreview, err := ai.GetTemplatePreview("english")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Println(englishPreview)
	
	fmt.Println("\n🔍 Template Validation:")
	fmt.Println("----------------------")
	
	languages := []string{"portuguese", "english", "spanish", "invalid"}
	for _, lang := range languages {
		err := ai.ValidateLanguage(lang)
		if err != nil {
			fmt.Printf("❌ %s: %v\n", lang, err)
		} else {
			fmt.Printf("✅ %s: Valid\n", lang)
		}
	}
	
	fmt.Println("\n🔧 Diff Analysis Demo:")
	fmt.Println("----------------------")
	
	analyzer := ai.NewDiffAnalyzer()
	
	diffData := &ai.DiffData{
		Content: `diff --git a/pkg/auth/oauth.go b/pkg/auth/oauth.go
index 1234567..abcdefg 100644
--- a/pkg/auth/oauth.go
+++ b/pkg/auth/oauth.go
@@ -1,5 +1,10 @@
 package auth
 
+import (
+    "golang.org/x/oauth2"
+)
+
 func AuthenticateUser() error {
+
     return nil
 }
diff --git a/web/login.html b/web/login.html
new file mode 100644
index 0000000..1234567
--- /dev/null
+++ b/web/login.html
@@ -0,0 +1,10 @@
+<!DOCTYPE html>
+<html>
+<head>
+    <title>Login</title>
+</head>
+<body>
+    <h1>OAuth Login</h1>
+    <button id="login">Login with OAuth</button>
+</body>
+</html>`,
		Files: []string{"pkg/auth/oauth.go", "web/login.html"},
		Stats: &utils.DiffStats{
			FilesChanged: 2,
			LinesAdded:   15,
			LinesRemoved: 0,
		},
	}
	
	analysis, err := analyzer.Analyze(diffData)
	if err != nil {
		fmt.Printf("Error analyzing diff: %v\n", err)
		return
	}
	
	fmt.Printf("Change Types: %v\n", analysis.ChangeTypes)
	fmt.Printf("Summary: %s\n", analysis.Summary)
	fmt.Printf("Complexity: %s\n", analysis.Complexity)
	fmt.Printf("Impact: %s\n", analysis.Impact)
	fmt.Printf("Tests Included: %v\n", analysis.TestsIncluded)
	fmt.Printf("Docs Included: %v\n", analysis.DocsIncluded)
	
	fmt.Println("\n✨ Demo completed! AI PR description generation is ready to use.")
	fmt.Println("\nTo use in real scenarios:")
	fmt.Println("bt pr create --ai --template portuguese")
	fmt.Println("bt pr create --ai --template english --jira context.md")
}
