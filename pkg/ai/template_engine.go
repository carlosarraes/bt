package ai

import (
	"fmt"
	"strings"
)

type TemplateEngine struct {
	language string
}

func NewTemplateEngine(language string) *TemplateEngine {
	return &TemplateEngine{
		language: language,
	}
}

func (t *TemplateEngine) Apply(vars map[string]interface{}) (string, error) {
	var template string

	switch t.language {
	case "portuguese":
		template = t.getPortugueseTemplate()
	case "english":
		template = t.getEnglishTemplate()
	default:
		return "", fmt.Errorf("unsupported template language: %s", t.language)
	}

	return t.renderTemplate(template, vars)
}

func (t *TemplateEngine) getPortugueseTemplate() string {
	return `## Descrição da Pull Request



### Contexto

{{contexto}}



### Alterações Realizadas

{{alteracoes}}



{{if client_specific}}### Cliente Específico

[{{client_specific}}] {{jira_ticket}}



{{end}}### Checklist

{{checklist}}



### Evidências

{{evidence_placeholders}}



---
**Estatísticas:** {{files_changed}} arquivo(s) alterado(s) • +{{additions}} -{{deletions}} linhas  
**Branch:** {{branch_name}} → {{target_branch}}`
}

func (t *TemplateEngine) getEnglishTemplate() string {
	return `## Pull Request Description



### Context

{{contexto}}



### Changes Made

{{alteracoes}}



{{if client_specific}}### Client Specific

[{{client_specific}}] {{jira_ticket}}



{{end}}### Checklist

{{checklist}}



### Evidence

{{evidence_placeholders}}



---
**Statistics:** {{files_changed}} file(s) changed • +{{additions}} -{{deletions}} lines  
**Branch:** {{branch_name}} → {{target_branch}}`
}

func (t *TemplateEngine) renderTemplate(template string, vars map[string]interface{}) (string, error) {
	result := template

	result = t.processConditionals(result, vars)

	for key, value := range vars {
		placeholder := fmt.Sprintf("{{%s}}", key)

		var replacement string
		switch v := value.(type) {
		case string:
			replacement = v
		case []string:
			replacement = t.formatListAsString(v)
		case int:
			replacement = fmt.Sprintf("%d", v)
		default:
			replacement = fmt.Sprintf("%v", v)
		}

		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	result = t.cleanupTemplate(result)

	return result, nil
}

func (t *TemplateEngine) processConditionals(template string, vars map[string]interface{}) string {
	lines := strings.Split(template, "\n")
	var result []string
	var inIf bool
	var ifCondition string
	var ifContent []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "{{if ") {
			ifStart := strings.Index(trimmed, "{{if ")
			ifEndOffset := strings.Index(trimmed[ifStart:], "}}")
			if ifEndOffset != -1 {
				ifEnd := ifStart + ifEndOffset

				condition := trimmed[ifStart+5 : ifEnd]
				condition = strings.TrimSpace(condition)

				contentAfterIf := strings.TrimSpace(trimmed[ifEnd+2:])

				inIf = true
				ifCondition = condition
				ifContent = []string{}

				if contentAfterIf != "" {
					ifContent = append(ifContent, contentAfterIf)
				}
				continue
			}
		}

		if strings.HasPrefix(trimmed, "{{end}}") {
			if inIf {
				if t.evaluateCondition(ifCondition, vars) {
					result = append(result, ifContent...)
				}
				inIf = false
				ifCondition = ""
				ifContent = []string{}
			}

			contentAfterEnd := strings.TrimSpace(trimmed[7:])
			if contentAfterEnd != "" {
				result = append(result, contentAfterEnd)
			}
			continue
		}

		if inIf {
			ifContent = append(ifContent, line)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func (t *TemplateEngine) evaluateCondition(condition string, vars map[string]interface{}) bool {
	value, exists := vars[condition]
	if !exists {
		return false
	}

	switch v := value.(type) {
	case string:
		return v != ""
	case bool:
		return v
	case int:
		return v != 0
	case []string:
		return len(v) > 0
	default:
		return value != nil
	}
}

func (t *TemplateEngine) formatListAsString(items []string) string {
	if len(items) == 0 {
		return ""
	}

	if len(items) > 0 && (strings.HasPrefix(items[0], "✅") || strings.HasPrefix(items[0], "- [")) {
		return strings.Join(items, "\n")
	}

	var formatted []string
	for _, item := range items {
		if !strings.HasPrefix(item, "•") && !strings.HasPrefix(item, "-") && !strings.HasPrefix(item, "*") {
			formatted = append(formatted, "• "+item)
		} else {
			formatted = append(formatted, item)
		}
	}

	return strings.Join(formatted, "\n")
}

func (t *TemplateEngine) cleanupTemplate(template string) string {
	lines := strings.Split(template, "\n")
	var cleaned []string

	for _, line := range lines {
		if strings.Contains(line, "{{") && strings.Contains(line, "}}") {
			withoutPlaceholders := line
			for strings.Contains(withoutPlaceholders, "{{") {
				start := strings.Index(withoutPlaceholders, "{{")
				end := strings.Index(withoutPlaceholders, "}}")
				if start >= 0 && end >= 0 && end > start {
					withoutPlaceholders = withoutPlaceholders[:start] + withoutPlaceholders[end+2:]
				} else {
					break
				}
			}

			if strings.TrimSpace(withoutPlaceholders) == "" {
				continue
			}
		}

		cleaned = append(cleaned, line)
	}

	result := strings.Join(cleaned, "\n")

	for strings.Contains(result, "\n\n\n") {
		result = strings.ReplaceAll(result, "\n\n\n", "\n\n")
	}

	result = strings.TrimSpace(result)

	return result
}

func GetSupportedLanguages() []string {
	return []string{"portuguese", "english"}
}

func ValidateLanguage(language string) error {
	supported := GetSupportedLanguages()
	for _, lang := range supported {
		if lang == language {
			return nil
		}
	}

	return fmt.Errorf("unsupported template language '%s', supported languages: %s",
		language, strings.Join(supported, ", "))
}

func GetTemplatePreview(language string) (string, error) {
	engine := NewTemplateEngine(language)

	sampleVars := map[string]interface{}{
		"contexto":              "Implementação de nova funcionalidade",
		"alteracoes":            "• Alterações no backend\n• Modificações na interface do usuário",
		"client_specific":       "ClienteXYZ",
		"jira_ticket":           "PROJ-123",
		"checklist":             []string{"✅ Testado localmente", "✅ Código revisado"},
		"evidence_placeholders": "- [ ] Screenshots da interface\n- [ ] Logs de teste",
		"files_changed":         5,
		"additions":             127,
		"deletions":             45,
		"branch_name":           "feature/new-functionality",
		"target_branch":         "main",
	}

	return engine.Apply(sampleVars)
}
