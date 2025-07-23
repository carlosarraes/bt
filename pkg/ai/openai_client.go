package ai

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
	cache  *CacheManager
}

type CacheManager struct {
	cacheDir string
}

type PRDescriptionSchema struct {
	Contexto              string   `json:"contexto" jsonschema:"description=Brief context description for the PR in the specified language"`
	Alteracoes            []string `json:"alteracoes" jsonschema:"description=List of specific changes made, each starting with bullet point"`
	ChecklistItems        []string `json:"checklist_items" jsonschema:"description=Dynamic checklist items based on change types, each starting with checkbox"`
	EvidencePlaceholders  []string `json:"evidence_placeholders" jsonschema:"description=Evidence placeholder items based on change types, each starting with checkbox"`
	Title                 string   `json:"title" jsonschema:"description=Concise PR title based on changes and branch name"`
	JiraTicket           string   `json:"jira_ticket,omitempty" jsonschema:"description=JIRA ticket ID if found in context"`
	ClientSpecific       string   `json:"client_specific,omitempty" jsonschema:"description=Client-specific information if found"`
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

	client := openai.NewClient(apiKey)
	
	cache, err := NewCacheManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create cache manager: %w", err)
	}

	return &OpenAIClient{
		client: client,
		cache:  cache,
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

func (c *OpenAIClient) GeneratePRDescription(ctx context.Context, input *PRAnalysisInput, language string) (*PRDescriptionSchema, error) {
	cacheKey := c.generateCacheKey(input, language)
	
	if cached, err := c.cache.Get(cacheKey); err == nil {
		return cached, nil
	}

	prompt := c.buildPrompt(input, language)

	schemaData := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"contexto": map[string]interface{}{
				"type":        "string",
				"description": fmt.Sprintf("Brief context description for the PR in %s", language),
			},
			"alteracoes": map[string]interface{}{
				"type":        "array",
				"description": "List of specific changes made, each starting with bullet point (•)",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"checklist_items": map[string]interface{}{
				"type":        "array",
				"description": "Dynamic checklist items based on change types, each starting with ✅",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"evidence_placeholders": map[string]interface{}{
				"type":        "array",
				"description": "Evidence placeholder items based on change types, each starting with - [ ]",
				"items": map[string]interface{}{
					"type": "string",
				},
			},
			"title": map[string]interface{}{
				"type":        "string",
				"description": "Concise PR title based on changes and branch name",
			},
			"jira_ticket": map[string]interface{}{
				"type":        "string",
				"description": "JIRA ticket ID if found in context (optional)",
			},
			"client_specific": map[string]interface{}{
				"type":        "string",
				"description": "Client-specific information if found (optional)",
			},
		},
		"required":             []string{"contexto", "alteracoes", "checklist_items", "evidence_placeholders", "title"},
		"additionalProperties": false,
	}

	schema := &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        "pr_description",
		Description: "Structured PR description schema",
		Schema:      &JSONSchemaMarshaler{Data: schemaData},
		Strict:      true,
	}

	req := openai.ChatCompletionRequest{
		Model: "o4-mini",
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: c.getSystemPrompt(language),
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
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

func (c *OpenAIClient) getSystemPrompt(language string) string {
	if language == "portuguese" {
		return `Você é um assistente especializado em análise de código e criação de descrições de Pull Requests. 

Sua tarefa é analisar mudanças de código e gerar descrições estruturadas e profissionais em português.

Diretrizes:
- Seja específico sobre as mudanças realizadas
- Use emojis e formatação markdown quando apropriado
- Crie checklists dinâmicos baseados no tipo de mudança
- Mantenha o tom profissional mas acessível
- Para alterações: use bullets (•) 
- Para checklist: use ✅ 
- Para evidências: use - [ ]
- Identifique tickets JIRA se presentes no contexto
- Extraia informações específicas do cliente quando relevante`
	}

	return `You are a code analysis assistant specialized in creating Pull Request descriptions.

Your task is to analyze code changes and generate structured, professional descriptions in English.

Guidelines:
- Be specific about the changes made
- Use emojis and markdown formatting when appropriate  
- Create dynamic checklists based on change types
- Maintain a professional but accessible tone
- For changes: use bullets (•)
- For checklist: use ✅
- For evidence: use - [ ]
- Identify JIRA tickets if present in context
- Extract client-specific information when relevant`
}

func (c *OpenAIClient) buildPrompt(input *PRAnalysisInput, language string) string {
	prompt := fmt.Sprintf(`Analyze the following PR information and generate a structured description:

**Branch Information:**
- Source: %s
- Target: %s

**Commit Messages:**
%s

**Files Changed:**
%s

**Git Diff (first 2000 chars):**
%s

**Statistics:**
- Files changed: %d
- Lines added: %d  
- Lines removed: %d
`, 
		input.SourceBranch,
		input.TargetBranch,
		formatCommits(input.CommitMessages),
		formatFiles(input.ChangedFiles),
		truncateString(input.GitDiff, 2000),
		input.FilesChanged,
		input.LinesAdded,
		input.LinesRemoved,
	)

	if input.JiraContext != "" {
		prompt += fmt.Sprintf("\n**JIRA Context:**\n%s\n", input.JiraContext)
	}

	if language == "portuguese" {
		prompt += "\nGere uma descrição de PR estruturada em português brasileiro."
	} else {
		prompt += "\nGenerate a structured PR description in English."
	}

	return prompt
}

func (c *OpenAIClient) generateCacheKey(input *PRAnalysisInput, language string) string {
	data := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%d",
		input.SourceBranch,
		input.TargetBranch,
		formatCommits(input.CommitMessages),
		formatFiles(input.ChangedFiles),
		truncateString(input.GitDiff, 1000),
		input.FilesChanged,
		input.LinesAdded,
		input.LinesRemoved,
	)
	
	if input.JiraContext != "" {
		data += "|" + input.JiraContext
	}
	data += "|" + language

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
	for _, commit := range commits {
		result += "- " + commit + "\n"
	}
	return result
}

func formatFiles(files []string) string {
	if len(files) == 0 {
		return "No files"
	}
	result := ""
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
	SourceBranch    string
	TargetBranch    string
	CommitMessages  []string
	ChangedFiles    []string
	GitDiff         string
	FilesChanged    int
	LinesAdded      int
	LinesRemoved    int
	JiraContext     string
}

type JSONSchemaMarshaler struct {
	Data map[string]interface{}
}

func (j *JSONSchemaMarshaler) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Data)
}
