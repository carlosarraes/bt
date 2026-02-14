package ai

import (
	"fmt"
	"strings"
)

type TemplateEngine struct{}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{}
}

func (t *TemplateEngine) Apply(vars map[string]interface{}) (string, error) {
	return t.renderTemplate(t.getTemplate(), vars)
}

func (t *TemplateEngine) getTemplate() string {
	return `# 🚀 Pull Request

## 📝 1. Context & Description
> *What are we doing and why?*

- **Change Type:** ` + "`{{change_type}}`" + `
- **Jira Ticket:** {{jira_ticket}}
- **Proposal/Design Doc:** {{design_doc}}
- **Summary:** {{summary}}

---

## 🛠️ 2. Technical Impact & UI
> *Check only what applies. Leave empty if not applicable.*

- [ ] **UI/UX Changes:** {{ui_changes}}
- [ ] **Database & Architecture:** {{db_architecture}}
- [ ] **Dependencies:** {{dependencies}}
- [ ] **Documentation:** {{documentation}}

---

## ✅ 3. Testing & Quality
> *How did you verify this works?*

- **Manual Testing:** Tested in ` + "`{{testing_env}}`" + `
- **Test Cases:** {{test_cases}}
{{if bug_fix_details}}- **Bug Fix Details:**
    - {{bug_fix_details}}
{{end}}
---

## 🛡️ 4. Safety, Observability & Risk
> *Mitigation and monitoring for production. Leave empty if not applicable.*

- **Feature Flags:** {{feature_flags}}
- **Security:** {{security}}
- **Monitoring:** {{monitoring}}
- **Rollback Safety:** {{rollback_safety}}
- **Production Validation:** {{production_validation}}

---
**Statistics:** {{files_changed}} file(s) changed | +{{additions}} -{{deletions}} lines
**Branch:** {{branch_name}} → {{target_branch}}`
}

func GetStaticTemplate() string {
	engine := NewTemplateEngine()
	vars := map[string]interface{}{
		"change_type":           "[Bug Fix / Feature / Refactor / Chore]",
		"jira_ticket":           "[Link]",
		"design_doc":            "[Link/NA]",
		"summary":               "(Briefly describe the problem and your solution)",
		"ui_changes":            "(Attach screenshots or screen recordings here)",
		"db_architecture":       "Performance/Locking impact? `[Yes / No]`",
		"dependencies":          "(List any new libraries or required config changes)",
		"documentation":         "Does this require a README or Confluence update? `[Yes / No]`",
		"testing_env":           "[Local / Homolog / N/A]",
		"test_cases":            "(Add a list with scenarios, edge cases, and failure cases tested)",
		"bug_fix_details":       "",
		"feature_flags":         "(List new feature flags added and how to enable them)",
		"security":              "Any impact on Auth, sensitive data, or permissions? `[Yes / No]`",
		"monitoring":            "(List Datadog dashboards, new logs, or specific alerts to watch)",
		"rollback_safety":       "Is it safe to revert without data inconsistency? `[Yes / No]`",
		"production_validation": "How will you confirm success after deployment?",
		"files_changed":         0,
		"additions":             0,
		"deletions":             0,
		"branch_name":           "",
		"target_branch":         "",
	}
	result, _ := engine.Apply(vars)
	return result
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
