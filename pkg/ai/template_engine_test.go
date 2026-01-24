package ai

import (
	"strings"
	"testing"
)

func TestTemplateEngine_Portuguese(t *testing.T) {
	engine := NewTemplateEngine("portuguese")

	vars := map[string]interface{}{
		"contexto":              "Implementação de nova funcionalidade",
		"alteracoes":            "• Alterações no backend\n• Modificações na interface do usuário",
		"client_specific":       "",
		"jira_ticket":           "",
		"checklist":             []string{"✅ Testado localmente", "✅ Código revisado"},
		"evidence_placeholders": "- [ ] Screenshots da interface\n- [ ] Logs de teste",
		"files_changed":         5,
		"additions":             127,
		"deletions":             45,
		"branch_name":           "feature/new-functionality",
		"target_branch":         "main",
	}

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "## Descrição da Pull Request") {
		t.Error("Portuguese template should contain Portuguese header")
	}

	if !strings.Contains(result, "Implementação de nova funcionalidade") {
		t.Error("Template should contain the context")
	}

	if !strings.Contains(result, "feature/new-functionality → main") {
		t.Error("Template should contain branch information")
	}

	if !strings.Contains(result, "5 arquivo(s) alterado(s)") {
		t.Error("Template should contain file statistics")
	}

}

func TestTemplateEngine_English(t *testing.T) {
	engine := NewTemplateEngine("english")

	vars := map[string]interface{}{
		"contexto":              "Implementation of new functionality",
		"alteracoes":            "• Backend changes\n• User interface modifications",
		"checklist":             []string{"✅ Tested locally", "✅ Code reviewed"},
		"evidence_placeholders": "- [ ] Interface screenshots\n- [ ] Test logs",
		"files_changed":         3,
		"additions":             45,
		"deletions":             12,
		"branch_name":           "feature/test",
		"target_branch":         "main",
	}

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "## Pull Request Description") {
		t.Error("English template should contain English header")
	}

	if !strings.Contains(result, "Implementation of new functionality") {
		t.Error("Template should contain the context")
	}

	if !strings.Contains(result, "3 file(s) changed") {
		t.Error("Template should contain file statistics")
	}
}

func TestTemplateEngine_ValidateLanguage(t *testing.T) {
	tests := []struct {
		language    string
		shouldError bool
	}{
		{"portuguese", false},
		{"english", false},
		{"spanish", true},
		{"invalid", true},
		{"", true},
	}

	for _, test := range tests {
		err := ValidateLanguage(test.language)
		if test.shouldError && err == nil {
			t.Errorf("Expected error for language %s, but got none", test.language)
		}
		if !test.shouldError && err != nil {
			t.Errorf("Expected no error for language %s, but got: %v", test.language, err)
		}
	}
}

func TestTemplateEngine_Conditionals(t *testing.T) {
	engine := NewTemplateEngine("portuguese")

	vars := map[string]interface{}{
		"contexto":              "Test context",
		"alteracoes":            "Test changes",
		"client_specific":       "TestClient",
		"jira_ticket":           "PROJ-123",
		"checklist":             []string{"✅ Test"},
		"evidence_placeholders": "- [ ] Test evidence",
		"files_changed":         1,
		"additions":             5,
		"deletions":             2,
		"branch_name":           "test",
		"target_branch":         "main",
	}

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "Cliente Específico") {
		t.Error("Template should contain client-specific section when provided")
	}

	if !strings.Contains(result, "TestClient") {
		t.Error("Template should contain client name")
	}

	if !strings.Contains(result, "PROJ-123") {
		t.Error("Template should contain JIRA ticket")
	}
}

func TestTemplateEngine_ConditionalEmpty(t *testing.T) {
	engine := NewTemplateEngine("portuguese")

	vars := map[string]interface{}{
		"contexto":              "Test context",
		"alteracoes":            "Test changes",
		"client_specific":       "",
		"jira_ticket":           "PROJ-123",
		"checklist":             []string{"✅ Test"},
		"evidence_placeholders": "- [ ] Test evidence",
		"files_changed":         1,
		"additions":             5,
		"deletions":             2,
		"branch_name":           "test",
		"target_branch":         "main",
	}

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if strings.Contains(result, "Cliente Específico") {
		t.Error("Template should not contain client-specific section when empty")
	}

	if strings.Contains(result, "{{if client_specific}}") {
		t.Error("Template should not contain unprocessed conditional syntax")
	}
}

func TestTemplateEngine_ConditionalInlineContent(t *testing.T) {
	engine := NewTemplateEngine("portuguese")

	vars := map[string]interface{}{
		"contexto":              "Test context",
		"alteracoes":            "Test changes",
		"client_specific":       "TestClient",
		"jira_ticket":           "PROJ-123",
		"checklist":             []string{"✅ Test"},
		"evidence_placeholders": "- [ ] Test evidence",
		"files_changed":         1,
		"additions":             5,
		"deletions":             2,
		"branch_name":           "test",
		"target_branch":         "main",
	}

	result, err := engine.Apply(vars)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	if !strings.Contains(result, "### Cliente Específico") {
		t.Error("Template should contain client-specific section header when provided")
	}

	if !strings.Contains(result, "[TestClient] PROJ-123") {
		t.Error("Template should contain formatted client and JIRA ticket")
	}

	if strings.Contains(result, "{{if client_specific}}") {
		t.Error("Template should not contain unprocessed conditional syntax")
	}
}
