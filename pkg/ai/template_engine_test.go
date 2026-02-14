package ai

import (
	"strings"
	"testing"
)

func newTestVars() map[string]interface{} {
	return map[string]interface{}{
		"change_type":           "Feature",
		"summary":               "Implementation of new functionality",
		"jira_ticket":           "PROJ-123",
		"design_doc":            "N/A",
		"ui_changes":            "None",
		"db_architecture":       "None",
		"dependencies":          "None",
		"documentation":         "README updated",
		"testing_env":           "Local",
		"test_cases":            "Unit tests for new handler",
		"bug_fix_details":       "",
		"feature_flags":         "N/A",
		"security":              "No security impact detected",
		"monitoring":            "[Datadog dashboards]",
		"rollback_safety":       "Safe to revert",
		"production_validation": "Health check endpoint",
		"files_changed":         5,
		"additions":             127,
		"deletions":             45,
		"branch_name":           "feature/new-functionality",
		"target_branch":         "main",
	}
}

func TestTemplateEngine_Apply(t *testing.T) {
	engine := NewTemplateEngine()
	vars := newTestVars()

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "# 🚀 Pull Request") {
		t.Error("Template should contain PR header")
	}
	if !strings.Contains(result, "## 📝 1. Context & Description") {
		t.Error("Template should contain Context section")
	}
	if !strings.Contains(result, "## 🛠️ 2. Technical Impact & UI") {
		t.Error("Template should contain Technical Impact section")
	}
	if !strings.Contains(result, "## ✅ 3. Testing & Quality") {
		t.Error("Template should contain Testing section")
	}
	if !strings.Contains(result, "## 🛡️ 4. Safety, Observability & Risk") {
		t.Error("Template should contain Safety section")
	}
}

func TestTemplateEngine_VariableSubstitution(t *testing.T) {
	engine := NewTemplateEngine()
	vars := newTestVars()

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "Implementation of new functionality") {
		t.Error("Template should contain the summary")
	}
	if !strings.Contains(result, "PROJ-123") {
		t.Error("Template should contain the JIRA ticket")
	}
	if !strings.Contains(result, "feature/new-functionality → main") {
		t.Error("Template should contain branch information")
	}
	if !strings.Contains(result, "5 file(s) changed") {
		t.Error("Template should contain file statistics")
	}
	if !strings.Contains(result, "+127 -45") {
		t.Error("Template should contain line statistics")
	}
	if !strings.Contains(result, "`Feature`") {
		t.Error("Template should contain change type")
	}
}

func TestTemplateEngine_BugFixDetailsConditional_Shown(t *testing.T) {
	engine := NewTemplateEngine()
	vars := newTestVars()
	vars["bug_fix_details"] = "Severity: 3 | Introduced in PR #42"

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "Bug Fix Details") {
		t.Error("Template should contain bug fix details section when provided")
	}
	if !strings.Contains(result, "Severity: 3 | Introduced in PR #42") {
		t.Error("Template should contain the bug fix details content")
	}
}

func TestTemplateEngine_BugFixDetailsConditional_Hidden(t *testing.T) {
	engine := NewTemplateEngine()
	vars := newTestVars()
	vars["bug_fix_details"] = ""

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if strings.Contains(result, "Bug Fix Details") {
		t.Error("Template should not contain bug fix details section when empty")
	}
	if strings.Contains(result, "{{if bug_fix_details}}") {
		t.Error("Template should not contain unprocessed conditional syntax")
	}
}

func TestTemplateEngine_NoUnprocessedPlaceholders(t *testing.T) {
	engine := NewTemplateEngine()
	vars := newTestVars()

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if strings.Contains(result, "{{") && strings.Contains(result, "}}") {
		t.Error("Template should not contain unprocessed placeholders")
	}
}

func TestGetStaticTemplate(t *testing.T) {
	result := GetStaticTemplate()

	if result == "" {
		t.Fatal("Static template should not be empty")
	}
	if !strings.Contains(result, "[Bug Fix / Feature / Refactor / Chore]") {
		t.Error("Static template should contain change type placeholder")
	}
	if !strings.Contains(result, "[Link]") {
		t.Error("Static template should contain JIRA ticket placeholder")
	}
	if !strings.Contains(result, "Context & Description") {
		t.Error("Static template should contain Context section")
	}
}
