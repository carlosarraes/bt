package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PipelineService provides pipeline-related API operations
type PipelineService struct {
	client *Client
}

// NewPipelineService creates a new pipeline service
func NewPipelineService(client *Client) *PipelineService {
	return &PipelineService{
		client: client,
	}
}

// ListPipelines retrieves a paginated list of pipelines for a repository
func (p *PipelineService) ListPipelines(ctx context.Context, workspace, repoSlug string, options *PipelineListOptions) (*PaginatedResponse, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines", workspace, repoSlug)
	
	// Add query parameters if options are provided
	if options != nil {
		params := make([]string, 0)
		
		if options.Status != "" {
			params = append(params, fmt.Sprintf("state.name=%s", url.QueryEscape(options.Status)))
		}
		
		if options.Branch != "" {
			params = append(params, fmt.Sprintf("target.ref_name=%s", url.QueryEscape(options.Branch)))
		}
		
		if options.Sort != "" {
			params = append(params, fmt.Sprintf("sort=%s", url.QueryEscape(options.Sort)))
		}
		
		if options.Page > 0 {
			params = append(params, fmt.Sprintf("page=%d", options.Page))
		}
		
		if options.PageLen > 0 {
			params = append(params, fmt.Sprintf("pagelen=%d", options.PageLen))
		}
		
		if len(params) > 0 {
			endpoint += "?" + strings.Join(params, "&")
		}
	}

	var result PaginatedResponse
	if err := p.client.GetJSON(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("failed to decode pipelines response: %w", err)
	}

	return &result, nil
}

// GetPipeline retrieves detailed information about a specific pipeline
func (p *PipelineService) GetPipeline(ctx context.Context, workspace, repoSlug, pipelineUUID string) (*Pipeline, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" {
		return nil, NewValidationError("workspace, repository slug, and pipeline UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s", workspace, repoSlug, pipelineUUID)
	
	var pipeline Pipeline
	if err := p.client.GetJSON(ctx, endpoint, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to get pipeline: %w", err)
	}

	return &pipeline, nil
}

// GetPipelineSteps retrieves all steps for a specific pipeline
func (p *PipelineService) GetPipelineSteps(ctx context.Context, workspace, repoSlug, pipelineUUID string) ([]*PipelineStep, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" {
		return nil, NewValidationError("workspace, repository slug, and pipeline UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps", workspace, repoSlug, pipelineUUID)
	
	var result PaginatedResponse
	if err := p.client.GetJSON(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("failed to get pipeline steps: %w", err)
	}

	// Parse the Values field (raw JSON) into PipelineStep structs  
	steps, err := parsePipelineStepsResults(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pipeline steps: %w", err)
	}

	return steps, nil
}

// parsePipelineStepsResults parses the paginated response into PipelineStep structs
func parsePipelineStepsResults(result *PaginatedResponse) ([]*PipelineStep, error) {
	var steps []*PipelineStep

	// Parse the Values field (raw JSON) into PipelineStep structs
	if result.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(result.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal step values: %w", err)
		}

		steps = make([]*PipelineStep, len(values))
		for i, rawStep := range values {
			var step PipelineStep
			if err := json.Unmarshal(rawStep, &step); err != nil {
				return nil, fmt.Errorf("failed to unmarshal step %d: %w", i, err)
			}
			steps[i] = &step
		}
	}

	return steps, nil
}

// GetStepLogs retrieves logs for a specific pipeline step
func (p *PipelineService) GetStepLogs(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) (io.ReadCloser, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" || stepUUID == "" {
		return nil, NewValidationError("workspace, repository slug, pipeline UUID, and step UUID are required", "")
	}

	// Format UUIDs for the endpoint (based on user's successful manual test)
	formattedPipelineUUID := pipelineUUID
	formattedStepUUID := stepUUID
	
	if !strings.HasPrefix(formattedPipelineUUID, "{") {
		formattedPipelineUUID = "{" + formattedPipelineUUID + "}"
	}
	if !strings.HasPrefix(formattedStepUUID, "{") {
		formattedStepUUID = "{" + formattedStepUUID + "}"
	}
	
	// Use the exact working endpoint format from user's successful test
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/%s/log", workspace, repoSlug, formattedPipelineUUID, formattedStepUUID)
	
	var resp *http.Response
	resp, err := p.client.getLogsRequest(ctx, endpoint)
	if err == nil {
		return resp.Body, nil
	}
	
	originalErr := err
	
	// If direct endpoint fails, try to get step details and use the logs link (use original UUID for this)
	steps, err := p.GetPipelineSteps(ctx, workspace, repoSlug, pipelineUUID)
	if err != nil {
		return nil, fmt.Errorf("all direct log endpoints failed (first error: %v), and failed to get pipeline steps: %w", originalErr, err)
	}

	// Find the specific step
	var targetStep *PipelineStep
	for _, step := range steps {
		if step.UUID == stepUUID {
			targetStep = step
			break
		}
	}

	if targetStep == nil {
		return nil, fmt.Errorf("all direct log endpoints failed (first error: %v), step with UUID %s not found in pipeline", originalErr, stepUUID)
	}

	// Check if the step has logs available via the logs link
	if targetStep.Logs == nil || targetStep.Logs.Href == "" {
		stepStatus := "unknown"
		if targetStep.State != nil {
			stepStatus = targetStep.State.Name
		}
		return nil, fmt.Errorf("no logs available: direct endpoints failed (%v), and no logs link for step '%s' (status: %s)", originalErr, targetStep.Name, stepStatus)
	}

	// Try the container-specific logs URL
	logURL := targetStep.Logs.Href
	resp, err = p.getLogsFromURL(ctx, logURL)
	if err != nil {
		return nil, fmt.Errorf("all direct log endpoints failed (first error: %v), and failed to get logs from step logs URL (%s): %w", originalErr, logURL, err)
	}

	return resp.Body, nil
}

// getLogsFromURL makes a request to a full logs URL
func (p *PipelineService) getLogsFromURL(ctx context.Context, logURL string) (*http.Response, error) {
	// Create the request
	req, err := http.NewRequestWithContext(ctx, "GET", logURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers for log requests
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", p.client.config.UserAgent)

	// Add authentication headers
	if p.client.authManager != nil {
		if err := p.client.authManager.SetHTTPHeaders(req); err != nil {
			return nil, fmt.Errorf("failed to set auth headers: %w", err)
		}
	}

	// Perform the request
	resp, err := p.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, ParseError(resp)
	}

	return resp, nil
}

// GetStepLogsWithRange retrieves logs for a specific pipeline step with HTTP Range support
// This is useful for large log files and supports resumable downloads
func (p *PipelineService) GetStepLogsWithRange(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string, rangeStart, rangeEnd int64) (io.ReadCloser, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" || stepUUID == "" {
		return nil, NewValidationError("workspace, repository slug, pipeline UUID, and step UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/%s/log", workspace, repoSlug, pipelineUUID, stepUUID)
	
	// Build the full URL
	fullURL, err := p.client.buildURL(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// Create request with Range header
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Range header for partial content
	if rangeEnd > rangeStart {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))
	} else if rangeStart > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", rangeStart))
	}

	// Set standard headers
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("User-Agent", p.client.config.UserAgent)

	// Add authentication headers
	if p.client.authManager != nil {
		if err := p.client.authManager.SetHTTPHeaders(req); err != nil {
			return nil, fmt.Errorf("failed to set auth headers: %w", err)
		}
	}

	// Perform the request
	resp, err := p.client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Check for HTTP errors (but allow 206 Partial Content)
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		return nil, ParseError(resp)
	}

	return resp.Body, nil
}

// StreamStepLogs streams logs for a specific pipeline step line by line
// Returns a channel that yields log lines
func (p *PipelineService) StreamStepLogs(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) (<-chan string, <-chan error) {
	logChan := make(chan string, 100)  // Buffer for log lines
	errChan := make(chan error, 1)     // Error channel

	go func() {
		defer close(logChan)
		defer close(errChan)

		// Get the log stream
		logReader, err := p.GetStepLogs(ctx, workspace, repoSlug, pipelineUUID, stepUUID)
		if err != nil {
			errChan <- fmt.Errorf("failed to get log stream: %w", err)
			return
		}
		defer logReader.Close()

		// Create a scanner to read line by line
		scanner := bufio.NewScanner(logReader)
		
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			case logChan <- scanner.Text():
				// Line sent successfully
			}
		}

		if err := scanner.Err(); err != nil {
			errChan <- fmt.Errorf("error reading log stream: %w", err)
		}
	}()

	return logChan, errChan
}

// CancelPipeline stops a running pipeline
func (p *PipelineService) CancelPipeline(ctx context.Context, workspace, repoSlug, pipelineUUID string) error {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" {
		return NewValidationError("workspace, repository slug, and pipeline UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/stopPipeline", workspace, repoSlug, pipelineUUID)
	
	resp, err := p.client.Post(ctx, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to cancel pipeline: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ParseError(resp)
	}

	return nil
}

// TriggerPipeline creates and starts a new pipeline
func (p *PipelineService) TriggerPipeline(ctx context.Context, workspace, repoSlug string, request *TriggerPipelineRequest) (*Pipeline, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}

	if request == nil || request.Target == nil {
		return nil, NewValidationError("trigger request with target is required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines", workspace, repoSlug)
	
	var pipeline Pipeline
	if err := p.client.PostJSON(ctx, endpoint, request, &pipeline); err != nil {
		return nil, fmt.Errorf("failed to trigger pipeline: %w", err)
	}

	return &pipeline, nil
}

// ListArtifacts retrieves a list of artifacts for a repository
func (p *PipelineService) ListArtifacts(ctx context.Context, workspace, repoSlug string) ([]*Artifact, error) {
	if workspace == "" || repoSlug == "" {
		return nil, NewValidationError("workspace and repository slug are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/downloads", workspace, repoSlug)
	
	var artifacts []*Artifact
	paginator := p.client.Paginate(endpoint, nil)
	if err := paginator.FetchAllTyped(ctx, &artifacts); err != nil {
		return nil, fmt.Errorf("failed to fetch artifacts: %w", err)
	}

	return artifacts, nil
}

// DownloadArtifact downloads a specific artifact
func (p *PipelineService) DownloadArtifact(ctx context.Context, workspace, repoSlug, artifactUUID string) (io.ReadCloser, error) {
	if workspace == "" || repoSlug == "" || artifactUUID == "" {
		return nil, NewValidationError("workspace, repository slug, and artifact UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/downloads/%s", workspace, repoSlug, artifactUUID)
	
	resp, err := p.client.Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to download artifact: %w", err)
	}

	// Don't close the response body here - the caller is responsible for closing it
	return resp.Body, nil
}

// GetPipelinesByBranch is a convenience method to get pipelines for a specific branch
func (p *PipelineService) GetPipelinesByBranch(ctx context.Context, workspace, repoSlug, branch string, limit int) ([]*Pipeline, error) {
	options := &PipelineListOptions{
		Branch:  branch,
		Sort:    "-created_on", // Most recent first
		PageLen: 50,
	}

	if limit > 0 && limit < 50 {
		options.PageLen = limit
	}

	// Get the first page to validate parameters
	_, err := p.ListPipelines(ctx, workspace, repoSlug, options)
	if err != nil {
		return nil, err
	}

	// Parse pipelines from the response
	var pipelines []*Pipeline
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines", workspace, repoSlug)
	
	// Build query parameters
	params := make([]string, 0)
	if branch != "" {
		params = append(params, fmt.Sprintf("target.ref_name=%s", url.QueryEscape(branch)))
	}
	params = append(params, "sort=-created_on")
	
	if len(params) > 0 {
		endpoint += "?" + strings.Join(params, "&")
	}

	pageOptions := &PageOptions{
		PageLen: options.PageLen,
		Limit:   limit,
	}

	paginator := p.client.Paginate(endpoint, pageOptions)
	if err := paginator.FetchAllTyped(ctx, &pipelines); err != nil {
		return nil, fmt.Errorf("failed to fetch pipelines for branch: %w", err)
	}

	return pipelines, nil
}

// GetStepTestReports retrieves test reports summary for a specific pipeline step
func (p *PipelineService) GetStepTestReports(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) ([]*TestReport, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" || stepUUID == "" {
		return nil, NewValidationError("workspace, repository slug, pipeline UUID, and step UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/%s/test_reports", workspace, repoSlug, pipelineUUID, stepUUID)
	
	var result PaginatedResponse
	if err := p.client.GetJSON(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("failed to get test reports: %w", err)
	}

	// Parse the Values field into TestReport structs
	var reports []*TestReport
	if result.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(result.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test report values: %w", err)
		}

		reports = make([]*TestReport, len(values))
		for i, rawReport := range values {
			var report TestReport
			if err := json.Unmarshal(rawReport, &report); err != nil {
				return nil, fmt.Errorf("failed to unmarshal test report %d: %w", i, err)
			}
			reports[i] = &report
		}
	}

	return reports, nil
}

// GetStepTestCases retrieves test cases for a specific pipeline step
func (p *PipelineService) GetStepTestCases(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) ([]*TestCase, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" || stepUUID == "" {
		return nil, NewValidationError("workspace, repository slug, pipeline UUID, and step UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/%s/test_reports/test_cases", workspace, repoSlug, pipelineUUID, stepUUID)
	
	var result PaginatedResponse
	if err := p.client.GetJSON(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("failed to get test cases: %w", err)
	}

	// Parse the Values field into TestCase structs
	var testCases []*TestCase
	if result.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(result.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test case values: %w", err)
		}

		testCases = make([]*TestCase, len(values))
		for i, rawCase := range values {
			var testCase TestCase
			if err := json.Unmarshal(rawCase, &testCase); err != nil {
				return nil, fmt.Errorf("failed to unmarshal test case %d: %w", i, err)
			}
			testCases[i] = &testCase
		}
	}

	return testCases, nil
}

// GetTestCaseReasons retrieves the output/reasons for a specific test case
func (p *PipelineService) GetTestCaseReasons(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID, testCaseUUID string) ([]*TestCaseReason, error) {
	if workspace == "" || repoSlug == "" || pipelineUUID == "" || stepUUID == "" || testCaseUUID == "" {
		return nil, NewValidationError("workspace, repository slug, pipeline UUID, step UUID, and test case UUID are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines/%s/steps/%s/test_reports/test_cases/%s/test_case_reasons", 
		workspace, repoSlug, pipelineUUID, stepUUID, testCaseUUID)
	
	var result PaginatedResponse
	if err := p.client.GetJSON(ctx, endpoint, &result); err != nil {
		return nil, fmt.Errorf("failed to get test case reasons: %w", err)
	}

	// Parse the Values field into TestCaseReason structs
	var reasons []*TestCaseReason
	if result.Values != nil {
		var values []json.RawMessage
		if err := json.Unmarshal(result.Values, &values); err != nil {
			return nil, fmt.Errorf("failed to unmarshal test case reason values: %w", err)
		}

		reasons = make([]*TestCaseReason, len(values))
		for i, rawReason := range values {
			var reason TestCaseReason
			if err := json.Unmarshal(rawReason, &reason); err != nil {
				return nil, fmt.Errorf("failed to unmarshal test case reason %d: %w", i, err)
			}
			reasons[i] = &reason
		}
	}

	return reasons, nil
}

// GetFailedPipelines is a convenience method to get recently failed pipelines
func (p *PipelineService) GetFailedPipelines(ctx context.Context, workspace, repoSlug string, limit int) ([]*Pipeline, error) {
	options := &PipelineListOptions{
		Status:  "FAILED",
		Sort:    "-created_on", // Most recent first
		PageLen: 50,
	}

	if limit > 0 && limit < 50 {
		options.PageLen = limit
	}

	// Use the existing ListPipelines method and parse results
	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines", workspace, repoSlug)
	
	// Build query parameters for failed pipelines
	params := []string{
		"sort=-created_on",
	}
	endpoint += "?" + strings.Join(params, "&")

	pageOptions := &PageOptions{
		PageLen: options.PageLen,
		Limit:   limit,
	}

	var pipelines []*Pipeline
	paginator := p.client.Paginate(endpoint, pageOptions)
	if err := paginator.FetchAllTyped(ctx, &pipelines); err != nil {
		return nil, fmt.Errorf("failed to fetch failed pipelines: %w", err)
	}

	// Filter for failed pipelines (API filtering might not be exact)
	var failedPipelines []*Pipeline
	for _, pipeline := range pipelines {
		if pipeline.State != nil && pipeline.State.Name == "FAILED" {
			failedPipelines = append(failedPipelines, pipeline)
		}
	}

	return failedPipelines, nil
}

// WaitForPipelineCompletion polls a pipeline until it completes or fails
func (p *PipelineService) WaitForPipelineCompletion(ctx context.Context, workspace, repoSlug, pipelineUUID string, pollInterval int) (*Pipeline, error) {
	if pollInterval <= 0 {
		pollInterval = 5 // Default 5 seconds
	}

	ticker := time.NewTicker(time.Duration(pollInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			pipeline, err := p.GetPipeline(ctx, workspace, repoSlug, pipelineUUID)
			if err != nil {
				return nil, fmt.Errorf("failed to check pipeline status: %w", err)
			}

			// Check if pipeline is completed
			if pipeline.State != nil {
				switch pipeline.State.Name {
				case "SUCCESSFUL", "FAILED", "ERROR", "STOPPED":
					return pipeline, nil
				case "PENDING", "IN_PROGRESS":
					// Continue polling
					continue
				}
			}
		}
	}
}

func (p *PipelineService) GetPipelinesByCommit(ctx context.Context, workspace, repoSlug, commitSHA string) ([]*Pipeline, error) {
	if workspace == "" || repoSlug == "" || commitSHA == "" {
		return nil, NewValidationError("workspace, repository slug, and commit SHA are required", "")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/pipelines", workspace, repoSlug)
	
	params := []string{
		fmt.Sprintf("target.commit.hash=%s", url.QueryEscape(commitSHA)),
		"sort=-created_on",
	}
	endpoint += "?" + strings.Join(params, "&")

	pageOptions := &PageOptions{
		PageLen: 50,
	}

	var pipelines []*Pipeline
	paginator := p.client.Paginate(endpoint, pageOptions)
	if err := paginator.FetchAllTyped(ctx, &pipelines); err != nil {
		return nil, fmt.Errorf("failed to fetch pipelines for commit: %w", err)
	}

	return pipelines, nil
}
