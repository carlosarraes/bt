package ai

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/carlosarraes/bt/pkg/config"
	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	cache  *CacheManager
	model  string
}

type CacheManager struct {
	cacheDir string
}

type PRDescriptionSchema struct {
	Title          string `json:"title"`
	ChangeType     string `json:"change_type"`
	Summary        string `json:"summary"`
	JiraTicket     string `json:"jira_ticket,omitempty"`
	UIChanges      string `json:"ui_changes"`
	DBArchitecture string `json:"db_architecture"`
	Dependencies   string `json:"dependencies"`
	Documentation  string `json:"documentation"`
	TestCases      string `json:"test_cases"`
	BugFixDetails  string `json:"bug_fix_details,omitempty"`
	Security       string `json:"security"`
	RollbackSafety string `json:"rollback_safety"`
}

type CachedResponse struct {
	Response  *PRDescriptionSchema `json:"response"`
	Timestamp time.Time            `json:"timestamp"`
	Hash      string               `json:"hash"`
}

func NewOpenAIClient() (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-5-mini"
	}

	client := openai.NewClient(apiKey)

	cache, err := NewCacheManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	return &OpenAIClient{
		client: client,
		cache:  cache,
		model:  model,
	}, nil
}

func NewOpenAIClientWithConfig(cfg *config.Config) (*OpenAIClient, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	model := os.Getenv("BT_LLM_MODEL")
	if model == "" {
		model = os.Getenv("OPENAI_MODEL")
	}
	if model == "" && cfg != nil {
		model = cfg.LLM.Model
	}
	if model == "" {
		model = "gpt-5-mini"
	}

	client := openai.NewClient(apiKey)

	cache, err := NewCacheManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	return &OpenAIClient{
		client: client,
		cache:  cache,
		model:  model,
	}, nil
}

func NewCacheManager() (*CacheManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "bt", "ai")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, err
	}

	return &CacheManager{
		cacheDir: cacheDir,
	}, nil
}

func (c *OpenAIClient) GeneratePRDescription(ctx context.Context, input *PRAnalysisInput) (*PRDescriptionSchema, error) {
	cacheKey := c.generateCacheKey(input)

	if cached, err := c.cache.Get(cacheKey); err == nil {
		return cached, nil
	}

	prompt := c.buildPrompt(input)

	schemaData := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Concise PR title based on changes and branch name",
			},
			"change_type": map[string]interface{}{
				"type":        "string",
				"enum":        []string{"Bug Fix", "Feature", "Refactor", "Chore"},
				"description": "Type of change based on branch prefix and code analysis",
			},
			"summary": map[string]interface{}{
				"type":        "string",
				"description": "Brief summary of the PR purpose and what it accomplishes",
			},
			"jira_ticket": map[string]interface{}{
				"type":        "string",
				"description": "JIRA ticket ID if found in branch name or commits, or empty string",
				"default":     "",
			},
			"ui_changes": map[string]interface{}{
				"type":        "string",
				"description": "Description of UI/UX changes if any, or 'None' if no frontend changes detected",
			},
			"db_architecture": map[string]interface{}{
				"type":        "string",
				"description": "Database or architecture changes if any, including migration info, or 'None' if no DB changes",
			},
			"dependencies": map[string]interface{}{
				"type":        "string",
				"description": "New libraries, config changes, or dependency updates detected, or 'None'",
			},
			"documentation": map[string]interface{}{
				"type":        "string",
				"description": "Documentation changes detected (README, docs, etc.), or 'None'",
			},
			"test_cases": map[string]interface{}{
				"type":        "string",
				"description": "Testing scenarios and edge cases to verify, based on the changes",
			},
			"bug_fix_details": map[string]interface{}{
				"type":        "string",
				"description": "If this is a bug fix: severity and details. Empty string if not a bug fix",
				"default":     "",
			},
			"security": map[string]interface{}{
				"type":        "string",
				"description": "Security impact assessment: auth, data handling, permissions changes. 'No security impact detected' if none",
			},
			"rollback_safety": map[string]interface{}{
				"type":        "string",
				"description": "Assessment of whether the change is safe to revert and any rollback concerns",
			},
		},
		"required":             []string{"title", "change_type", "summary", "jira_ticket", "ui_changes", "db_architecture", "dependencies", "documentation", "test_cases", "bug_fix_details", "security", "rollback_safety"},
		"additionalProperties": false,
	}

	schema := &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "pr_description",
		Description: "Structured PR description schema",
		Schema:      &JSONSchemaMarshaler{Data: schemaData},
		Strict:      true,
	}

	req := openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: c.getSystemPrompt(),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type:       openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: schema,
		},
	}

	resp, err := c.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	var result PRDescriptionSchema
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if err := c.cache.Set(cacheKey, &result); err != nil {
		fmt.Printf("Warning: failed to cache result: %v\n", err)
	}

	return &result, nil
}

func (c *OpenAIClient) GetModel() string {
	return c.model
}

func (c *OpenAIClient) getSystemPrompt() string {
	return `You are a code analysis assistant specialized in creating Pull Request descriptions.

Your task is to analyze code changes and generate structured, professional PR descriptions.

For each PR, you must:
1. Classify the change type (Bug Fix, Feature, Refactor, Chore) based on branch prefix and diff patterns
2. Write a concise summary of what the PR does and why
3. Detect UI/UX changes from frontend file modifications
4. Detect database/architecture impact from migration files, schema changes, or SQL
5. Detect new dependencies from package manager files (go.mod, package.json, requirements.txt, etc.)
6. Detect documentation changes (README, docs/, etc.)
7. Suggest test cases and scenarios based on the changes
8. Fill bug_fix_details only if this is a bug fix (empty string otherwise)
9. Assess security impact: look for auth, token, password, permission patterns
10. Assess rollback safety: DB migrations make rollback risky, pure code changes are safe

Guidelines:
- Be specific about the changes made based on the actual diff
- If no changes detected for a category, use "None"
- Extract JIRA ticket IDs from branch names or commit messages if present
- Keep the summary concise but informative`
}

func (c *OpenAIClient) buildPrompt(input *PRAnalysisInput) string {
	prompt := fmt.Sprintf(`Analyze the following PR information and generate a structured description:

**Branch Information:**
- Source: %s
- Target: %s

**Commit Messages:**
%s

**Files Changed:**
%s

**Git Diff (ANALYZE THIS CAREFULLY):**
%s

**Statistics:**
- Files changed: %d
- Lines added: %d
- Lines removed: %d

CRITICAL INSTRUCTIONS:
1. Lines starting with '+' are ADDITIONS (new code being added)
2. Lines starting with '-' are DELETIONS (old code being removed)
3. Classify change_type from branch prefix: feature/feat/ -> Feature, fix/hotfix/bugfix/ -> Bug Fix, refactor/ -> Refactor, chore/ -> Chore
4. For each Technical Impact category (ui_changes, db_architecture, dependencies, documentation), analyze the changed files and diff content
5. If no JIRA ticket found, set jira_ticket to empty string
6. If this is NOT a bug fix, set bug_fix_details to empty string
`,
		input.SourceBranch,
		input.TargetBranch,
		formatCommits(input.CommitMessages),
		formatFiles(input.ChangedFiles),
		truncateString(input.GitDiff, 1500),
		input.FilesChanged,
		input.LinesAdded,
		input.LinesRemoved,
	)

	if input.JiraContext != "" {
		prompt += fmt.Sprintf("\n**JIRA Context:**\n%s\n", input.JiraContext)
	}

	return prompt
}

func (c *OpenAIClient) generateCacheKey(input *PRAnalysisInput) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%d",
		input.SourceBranch,
		input.TargetBranch,
		formatCommits(input.CommitMessages),
		formatFiles(input.ChangedFiles),
		truncateString(input.GitDiff, 500),
		input.FilesChanged,
		input.LinesAdded,
		input.LinesRemoved,
	)

	if input.JiraContext != "" {
		data += "|" + input.JiraContext
	}

	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}

func (cm *CacheManager) Get(key string) (*PRDescriptionSchema, error) {
	cachePath := filepath.Join(cm.cacheDir, key+".json")

	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, err
	}

	var cached CachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	if time.Since(cached.Timestamp) > 24*time.Hour {
		os.Remove(cachePath)
		return nil, fmt.Errorf("cache expired")
	}

	return cached.Response, nil
}

func (cm *CacheManager) Set(key string, response *PRDescriptionSchema) error {
	cachePath := filepath.Join(cm.cacheDir, key+".json")

	cached := CachedResponse{
		Response:  response,
		Timestamp: time.Now(),
		Hash:      key,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0644)
}

func formatCommits(commits []string) string {
	if len(commits) == 0 {
		return "No commits"
	}
	result := ""
	maxCommits := 10
	if len(commits) > maxCommits {
		commits = commits[:maxCommits]
		result += fmt.Sprintf("- ... showing first %d of %d commits\n", maxCommits, len(commits)+maxCommits)
	}
	for _, commit := range commits {
		if len(commit) > 100 {
			commit = commit[:100] + "..."
		}
		result += "- " + commit + "\n"
	}
	return result
}

func formatFiles(files []string) string {
	if len(files) == 0 {
		return "No files"
	}
	result := ""
	maxFiles := 20
	if len(files) > maxFiles {
		result += fmt.Sprintf("- ... showing first %d of %d files\n", maxFiles, len(files))
		files = files[:maxFiles]
	}
	for _, file := range files {
		result += "- " + file + "\n"
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... [truncated]"
}

type PRAnalysisInput struct {
	SourceBranch   string
	TargetBranch   string
	CommitMessages []string
	ChangedFiles   []string
	GitDiff        string
	FilesChanged   int
	LinesAdded     int
	LinesRemoved   int
	JiraContext    string
}

type JSONSchemaMarshaler struct {
	Data map[string]interface{}
}

func (j *JSONSchemaMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Data)
}
