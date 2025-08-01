package sonarcloud

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
)

type Service struct {
	client    *Client
	discovery *ProjectKeyDiscovery
}

func NewService(client *Client, bitbucketClient *api.Client) *Service {
	return &Service{
		client:    client,
		discovery: NewProjectKeyDiscovery(bitbucketClient),
	}
}

func (s *Service) buildAPIContext(pipeline *api.Pipeline, projectKey string) APIContext {
	context := APIContext{
		ProjectKey: projectKey,
		BaseParams: map[string]string{"component": projectKey},
	}

	if pipeline.Target != nil && pipeline.Target.Type == "pipeline_pullrequest_target" {
		context.IsPullRequest = true
		context.PreferredMetrics = []string{
			"new_coverage", "new_uncovered_lines", "new_bugs",
			"new_vulnerabilities", "new_code_smells",
		}
	} else {
		context.IsPullRequest = false
		context.PreferredMetrics = []string{
			"coverage", "bugs", "vulnerabilities", "code_smells",
			"security_hotspots", "duplicated_lines_density",
		}
	}

	if pipeline.State != nil {
		switch pipeline.State.Name {
		case "COMPLETED", "SUCCESSFUL", "FAILED", "ERROR", "STOPPED":
			context.PipelineCompleted = true
		case "IN_PROGRESS", "PENDING":
			context.PipelineRunning = true
		}
	}

	return context
}

func (s *Service) GenerateReportForPR(ctx context.Context, prID int, workspace, repo string, filters FilterOptions) (*Report, error) {
	if filters.Debug {
		fmt.Printf("DEBUG: SonarCloud GenerateReportForPR called for PR #%d\n", prID)
	}

	discoveryResult, err := s.discovery.DiscoverProjectKey(ctx, workspace, repo, "")
	if err != nil {
		return nil, fmt.Errorf("failed to discover SonarCloud project key: %w", err)
	}

	apiContext := APIContext{
		ProjectKey:        discoveryResult.ProjectKey,
		BaseParams:        map[string]string{"component": discoveryResult.ProjectKey},
		IsPullRequest:     true,
		PullRequestID:     prID,
		PreferredMetrics:  []string{"new_coverage", "new_uncovered_lines", "new_bugs", "new_vulnerabilities", "new_code_smells"},
		PipelineCompleted: true,
	}
	
	apiContext.BaseParams["pullRequest"] = fmt.Sprintf("%d", prID)
	
	if filters.Debug {
		fmt.Printf("DEBUG: API Context - IsPullRequest: %t, PullRequestID: %d\n", 
			apiContext.IsPullRequest, apiContext.PullRequestID)
		fmt.Printf("DEBUG: Project Key: %s\n", discoveryResult.ProjectKey)
	}

	report := &Report{
		ProjectKey:    discoveryResult.ProjectKey,
		Timestamp:     time.Now(),
		PullRequestID: &prID,
	}

	var errors []error

	if qualityGate, err := s.GetQualityGate(ctx, apiContext); err != nil {
		errors = append(errors, fmt.Errorf("quality gate: %w", err))
		report.QualityGate = &QualityGateInfo{Status: "UNKNOWN", Error: err.Error()}
	} else {
		report.QualityGate = qualityGate
	}

	if filters.IncludeCoverage {
		if coverage, err := s.GetCoverageData(ctx, apiContext, filters); err != nil {
			errors = append(errors, fmt.Errorf("coverage: %w", err))
			report.Coverage = &CoverageData{Available: false, Error: err.Error()}
		} else {
			report.Coverage = coverage
		}
	}

	if filters.IncludeIssues {
		if issues, err := s.GetIssuesData(ctx, apiContext, filters); err != nil {
			errors = append(errors, fmt.Errorf("issues: %w", err))
			report.Issues = &IssuesData{Available: false, Error: err.Error()}
		} else {
			report.Issues = issues
		}
	}

	if metrics, err := s.GetMetricsData(ctx, apiContext); err != nil {
		errors = append(errors, fmt.Errorf("metrics: %w", err))
		report.Metrics = &MetricsData{Available: false, Error: err.Error()}
	} else {
		report.Metrics = metrics
	}

	if len(errors) > 0 {
		report.Warnings = errors
	}

	return report, nil
}

func (s *Service) GenerateReport(ctx context.Context, pipeline *api.Pipeline, workspace, repo string, filters FilterOptions) (*Report, error) {
	if filters.Debug {
		fmt.Printf("DEBUG: SonarCloud GenerateReport called\n")
	}
	commitHash := ""
	if pipeline.Target != nil && pipeline.Target.Commit != nil {
		commitHash = pipeline.Target.Commit.Hash
	}

	discoveryResult, err := s.discovery.DiscoverProjectKey(ctx, workspace, repo, commitHash)
	if err != nil {
		return nil, fmt.Errorf("failed to discover SonarCloud project key: %w", err)
	}

	apiContext := s.buildAPIContext(pipeline, discoveryResult.ProjectKey)
	
	if filters.Debug {
		fmt.Printf("DEBUG: API Context - IsPullRequest: %t, PullRequestID: %d\n", 
			apiContext.IsPullRequest, apiContext.PullRequestID)
		fmt.Printf("DEBUG: Project Key: %s\n", discoveryResult.ProjectKey)
	}

	report := &Report{
		ProjectKey:    discoveryResult.ProjectKey,
		Timestamp:     time.Now(),
		PullRequestID: nil,
	}

	if apiContext.IsPullRequest {
		report.PullRequestID = &apiContext.PullRequestID
	}

	var errors []error

	if qualityGate, err := s.GetQualityGate(ctx, apiContext); err != nil {
		errors = append(errors, fmt.Errorf("quality gate: %w", err))
		report.QualityGate = &QualityGateInfo{Status: "UNKNOWN", Error: err.Error()}
	} else {
		report.QualityGate = qualityGate
	}

	if filters.IncludeCoverage {
		if coverage, err := s.GetCoverageData(ctx, apiContext, filters); err != nil {
			errors = append(errors, fmt.Errorf("coverage: %w", err))
			report.Coverage = &CoverageData{Available: false, Error: err.Error()}
		} else {
			report.Coverage = coverage
		}
	}

	if filters.IncludeIssues {
		if issues, err := s.GetIssuesData(ctx, apiContext, filters); err != nil {
			errors = append(errors, fmt.Errorf("issues: %w", err))
			report.Issues = &IssuesData{Available: false, Error: err.Error()}
		} else {
			report.Issues = issues
		}
	}

	if metrics, err := s.GetMetricsData(ctx, apiContext); err != nil {
		errors = append(errors, fmt.Errorf("metrics: %w", err))
		report.Metrics = &MetricsData{Available: false, Error: err.Error()}
	} else {
		report.Metrics = metrics
	}

	if len(errors) > 0 {
		report.Warnings = errors
	}

	return report, nil
}

func (s *Service) GetQualityGate(ctx context.Context, apiContext APIContext) (*QualityGateInfo, error) {
	params := make(map[string]string)
	for k, v := range apiContext.BaseParams {
		params[k] = v
	}
	params["projectKey"] = apiContext.ProjectKey

	var qualityGate QualityGate
	if err := s.client.GetJSON(ctx, "qualitygates/project_status", params, apiContext, &qualityGate); err != nil {
		return nil, err
	}

	info := &QualityGateInfo{
		Status: qualityGate.ProjectStatus.Status,
		Passed: qualityGate.ProjectStatus.Status == "OK",
		Conditions: make([]QualityGateCondition, 0, len(qualityGate.ProjectStatus.Conditions)),
		FailedConditions: make([]QualityGateCondition, 0),
	}

	for _, condition := range qualityGate.ProjectStatus.Conditions {
		qgCondition := QualityGateCondition{
			MetricKey:   condition.MetricKey,
			MetricName:  s.getMetricDisplayName(condition.MetricKey),
			Comparator:  condition.Comparator,
			Threshold:   condition.ErrorThreshold,
			ActualValue: condition.ActualValue,
			Status:      condition.Status,
			Failed:      condition.Status == "ERROR",
			OnNewCode:   condition.PeriodIndex > 0,
		}

		info.Conditions = append(info.Conditions, qgCondition)
		if qgCondition.Failed {
			info.FailedConditions = append(info.FailedConditions, qgCondition)
		}
	}

	return info, nil
}

func (s *Service) GetCoverageData(ctx context.Context, apiContext APIContext, filters FilterOptions) (*CoverageData, error) {
	data := &CoverageData{
		Available: true,
		Files:     make([]CoverageFile, 0),
		UncoveredLines: make([]UncoveredLine, 0),
	}

	if err := s.getProjectCoverage(ctx, apiContext, data); err != nil {
		return nil, err
	}

	if err := s.getFileCoverage(ctx, apiContext, data, filters); err != nil {
		return nil, err
	}

	if err := s.getUncoveredLines(ctx, apiContext, data, filters); err != nil {
		if filters.Debug {
			fmt.Printf("DEBUG: Error getting uncovered lines: %v\n", err)
		}
	}
	
	if filters.Debug {
		fmt.Printf("DEBUG: Coverage details count: %d\n", len(data.CoverageDetails))
		fmt.Printf("DEBUG: Uncovered lines count: %d\n", len(data.UncoveredLines))
	}

	return data, nil
}

func (s *Service) getProjectCoverage(ctx context.Context, apiContext APIContext, data *CoverageData) error {
	params := make(map[string]string)
	for k, v := range apiContext.BaseParams {
		params[k] = v
	}

	metrics := []string{"coverage", "uncovered_lines"}
	if apiContext.IsPullRequest {
		metrics = append(metrics, "new_coverage", "new_uncovered_lines")
	}
	params["metricKeys"] = strings.Join(metrics, ",")

	var measure ComponentMeasure
	if err := s.client.GetJSON(ctx, "measures/component", params, apiContext, &measure); err != nil {
		return err
	}

	for _, metric := range measure.Component.Measures {
		switch metric.Metric {
		case "coverage":
			if coverage, err := strconv.ParseFloat(metric.Value, 64); err == nil {
				data.OverallCoverage = coverage
			}
		case "new_coverage":
			if len(metric.Periods) > 0 {
				if coverage, err := strconv.ParseFloat(metric.Periods[0].Value, 64); err == nil {
					data.NewCodeCoverage = coverage
				}
			}
		case "uncovered_lines":
			if lines, err := strconv.Atoi(metric.Value); err == nil {
				data.Summary.UncoveredLines = lines
			}
		case "new_uncovered_lines":
			if len(metric.Periods) > 0 {
				if lines, err := strconv.Atoi(metric.Periods[0].Value); err == nil {
					data.Summary.NewUncoveredLines = lines
				}
			}
		}
	}

	return nil
}

func (s *Service) getFileCoverage(ctx context.Context, apiContext APIContext, data *CoverageData, filters FilterOptions) error {
	params := make(map[string]string)
	for k, v := range apiContext.BaseParams {
		params[k] = v
	}

	params["qualifiers"] = "FIL"
	params["metricKeys"] = "coverage,uncovered_lines,new_coverage,new_uncovered_lines"

	if filters.ShowWorstFirst {
		params["s"] = "metric"
		params["metricSort"] = "coverage"
		params["asc"] = "true"
	}

	pageSize := filters.Limit
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}
	params["ps"] = strconv.Itoa(pageSize)

	var tree ComponentTree
	if err := s.client.GetJSON(ctx, "measures/component_tree", params, apiContext, &tree); err != nil {
		return err
	}

	for _, component := range tree.Components {
		file := CoverageFile{
			Path:         component.Path,
			Name:         component.Name,
			Language:     component.Language,
			ComponentKey: component.Key,
		}

		for _, measure := range component.Measures {
			switch measure.Metric {
			case "coverage":
				if coverage, err := strconv.ParseFloat(measure.Value, 64); err == nil {
					file.Coverage = coverage
				}
			case "uncovered_lines":
				if lines, err := strconv.Atoi(measure.Value); err == nil {
					file.UncoveredLines = lines
				}
			case "new_coverage":
				if len(measure.Periods) > 0 {
					if coverage, err := strconv.ParseFloat(measure.Periods[0].Value, 64); err == nil {
						file.NewCoverage = coverage
					}
				}
			case "new_uncovered_lines":
				if len(measure.Periods) > 0 {
					if lines, err := strconv.Atoi(measure.Periods[0].Value); err == nil {
						file.NewUncoveredLines = lines
					}
				}
			}
		}

		if filters.CoverageThreshold > 0 && file.Coverage >= filters.CoverageThreshold {
			continue
		}

		data.Files = append(data.Files, file)
	}

	return nil
}

func (s *Service) getUncoveredLines(ctx context.Context, apiContext APIContext, data *CoverageData, filters FilterOptions) error {
	if filters.NoLineDetails {
		return nil
	}

	eligibleFiles := s.filterEligibleFiles(data.Files, filters)
	if filters.Debug {
		fmt.Printf("DEBUG: Eligible files for line details: %d\n", len(eligibleFiles))
		for i, file := range eligibleFiles {
			fmt.Printf("DEBUG: File %d: %s (ComponentKey: %s, NewUncovered: %d)\n", 
				i+1, file.Path, file.ComponentKey, file.NewUncoveredLines)
		}
	}
	if len(eligibleFiles) == 0 {
		return nil
	}

	coverageDetails, err := s.getUncoveredLinesForFiles(ctx, eligibleFiles, apiContext, filters)
	if err != nil {
		return err
	}

	data.CoverageDetails = coverageDetails

	for _, details := range coverageDetails {
		data.UncoveredLines = append(data.UncoveredLines, details.UncoveredLines...)
	}

	return nil
}

func (s *Service) filterEligibleFiles(files []CoverageFile, filters FilterOptions) []CoverageFile {
	var eligible []CoverageFile
	
	for _, file := range files {
		if filters.NewLinesOnly && file.NewUncoveredLines == 0 {
			continue
		}
		
		if filters.MinUncoveredLines > 0 && file.UncoveredLines < filters.MinUncoveredLines {
			continue
		}
		
		if filters.MaxUncoveredLines > 0 && file.UncoveredLines > filters.MaxUncoveredLines {
			continue
		}
		
		if file.UncoveredLines > 500 {
			continue
		}
		
		if file.Coverage >= 100.0 {
			continue
		}
		
		if s.isGeneratedFile(file.Path) {
			continue
		}
		
		if filters.FilePattern != "" {
			matched, err := s.matchFilePattern(file.Path, filters.FilePattern)
			if err != nil || !matched {
				continue
			}
		}
		
		eligible = append(eligible, file)
	}
	
	return eligible
}

func (s *Service) isGeneratedFile(path string) bool {
	generatedPatterns := []string{
		"node_modules/", "__pycache__/", ".git/",
		"_pb2.py", ".pb.go", "generated/",
		"vendor/", "build/", "dist/",
		".min.js", ".min.css",
	}
	
	for _, pattern := range generatedPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	
	return false
}

func (s *Service) matchFilePattern(path, pattern string) (bool, error) {
	if pattern == "" {
		return true, nil
	}
	
	matched, err := filepath.Match(pattern, path)
	if err != nil {
		return false, err
	}
	if matched {
		return true, nil
	}
	
	matched, err = filepath.Match(pattern, filepath.Base(path))
	return matched, err
}

func (s *Service) getUncoveredLinesForFiles(ctx context.Context, files []CoverageFile, apiContext APIContext, filters FilterOptions) ([]CoverageDetails, error) {
	var results []CoverageDetails
	
	batchSize := 5
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batch := files[i:end]
		
		batchResults, err := s.processBatch(ctx, batch, apiContext, filters)
		if err != nil {
			continue
		}
		
		results = append(results, batchResults...)
		
		if i+batchSize < len(files) {
			time.Sleep(200 * time.Millisecond)
		}
	}
	
	return results, nil
}

func (s *Service) processBatch(ctx context.Context, files []CoverageFile, apiContext APIContext, filters FilterOptions) ([]CoverageDetails, error) {
	var results []CoverageDetails
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for _, file := range files {
		wg.Add(1)
		go func(f CoverageFile) {
			defer wg.Done()
			
			details, err := s.getUncoveredLinesForFile(ctx, f, apiContext, filters)
			if err != nil {
				return
			}
			
			mu.Lock()
			results = append(results, *details)
			mu.Unlock()
		}(file)
	}
	
	wg.Wait()
	return results, nil
}

func (s *Service) getUncoveredLinesForFile(ctx context.Context, file CoverageFile, apiContext APIContext, filters FilterOptions) (*CoverageDetails, error) {
	params := make(map[string]string)
	for k, v := range apiContext.BaseParams {
		params[k] = v
	}
	params["key"] = file.ComponentKey

	if filters.Debug {
		fmt.Printf("DEBUG: Getting lines for file: %s (key: %s)\n", file.Path, file.ComponentKey)
	}

	var sourceLines SourceLines
	if err := s.client.GetJSON(ctx, "sources/lines", params, apiContext, &sourceLines); err != nil {
		if filters.Debug {
			fmt.Printf("DEBUG: Error getting lines for %s: %v\n", file.Path, err)
		}
		return nil, err
	}

	if filters.Debug {
		fmt.Printf("DEBUG: Got %d source lines for %s\n", len(sourceLines.Sources), file.Path)
	}

	details := &CoverageDetails{
		FilePath:        file.Path,
		FileName:        file.Name,
		CoveragePercent: file.Coverage,
		TotalUncovered:  file.UncoveredLines,
		Language:        file.Language,
		UncoveredLines:  make([]UncoveredLine, 0),
	}

	for _, line := range sourceLines.Sources {
		if line.LineHits != nil && *line.LineHits == 0 {
			uncoveredLine := UncoveredLine{
				File:  file.Path,
				Line:  line.Line,
				Code:  s.processCodeLine(line.Code, filters.TruncateLines, file.Path),
				IsNew: line.IsNew,
			}

			if filters.NewLinesOnly && !line.IsNew {
				continue
			}

			details.UncoveredLines = append(details.UncoveredLines, uncoveredLine)
			
			if line.IsNew {
				details.NewUncovered++
			}
		}
	}

	if !filters.ShowAllLines {
		details.UncoveredLines = s.prioritizeUncoveredLines(details.UncoveredLines, filters.LinesPerFile)
	}

	return details, nil
}

func (s *Service) processCodeLine(code string, truncateLength int, filePath string) string {
	code = strings.TrimSpace(code)
	
	if !s.shouldPreserveHTMLTags(filePath) {
		code = s.cleanHTMLTags(code)
	}
	
	if truncateLength > 0 && len(code) > truncateLength {
		return code[:truncateLength-3] + "..."
	}
	
	return code
}

func (s *Service) shouldPreserveHTMLTags(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	webExtensions := []string{".html", ".htm", ".tsx", ".jsx", ".vue", ".svelte"}
	
	for _, webExt := range webExtensions {
		if ext == webExt {
			return true
		}
	}
	
	return false
}

func (s *Service) cleanHTMLTags(code string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	cleaned := re.ReplaceAllString(code, "")
	
	cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	cleaned = strings.TrimSpace(cleaned)
	
	return cleaned
}

func (s *Service) prioritizeUncoveredLines(lines []UncoveredLine, maxLines int) []UncoveredLine {
	if maxLines <= 0 || len(lines) <= maxLines {
		return lines
	}

	var newLines, oldLines []UncoveredLine
	
	for _, line := range lines {
		if line.IsNew {
			newLines = append(newLines, line)
		} else {
			oldLines = append(oldLines, line)
		}
	}

	var result []UncoveredLine
	result = append(result, newLines...)

	remaining := maxLines - len(newLines)
	if remaining > 0 && len(oldLines) > 0 {
		end := remaining
		if end > len(oldLines) {
			end = len(oldLines)
		}
		result = append(result, oldLines[:end]...)
	}

	return result
}

func (s *Service) GetIssuesData(ctx context.Context, apiContext APIContext, filters FilterOptions) (*IssuesData, error) {
	data := &IssuesData{
		Available: true,
		Issues:    make([]ProcessedIssue, 0),
		Summary: IssuesSummary{
			BySeverity: make(map[string]int),
			ByType:     make(map[string]int),
			ByLanguage: make(map[string]int),
		},
	}

	params := make(map[string]string)
	for k, v := range apiContext.BaseParams {
		params[k] = v
	}

	params["componentKeys"] = apiContext.ProjectKey
	params["types"] = "BUG,VULNERABILITY,CODE_SMELL"

	if len(filters.SeverityFilter) > 0 {
		params["severities"] = strings.Join(filters.SeverityFilter, ",")
	} else {
		params["severities"] = "BLOCKER,CRITICAL,MAJOR,MINOR,INFO"
	}

	params["s"] = "SEVERITY"
	params["asc"] = "false"

	pageSize := filters.Limit
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 100
	}
	params["ps"] = strconv.Itoa(pageSize)

	var issues IssuesSearch
	if err := s.client.GetJSON(ctx, "issues/search", params, apiContext, &issues); err != nil {
		return nil, err
	}

	data.TotalIssues = issues.Total

	ruleNames := make(map[string]string)
	for _, rule := range issues.Rules {
		ruleNames[rule.Key] = rule.Name
	}

	for _, issue := range issues.Issues {
		processedIssue := ProcessedIssue{
			Key:           issue.Key,
			Type:          issue.Type,
			Severity:      issue.Severity,
			Rule:          issue.Rule,
			RuleName:      ruleNames[issue.Rule],
			Component:     issue.Component,
			File:          s.extractFileFromComponent(issue.Component),
			Line:          issue.Line,
			Message:       issue.Message,
			Effort:        issue.Effort,
			TechnicalDebt: issue.Debt,
			CreatedAt:     issue.CreationDate,
		}

		data.Summary.ByType[issue.Type]++
		switch issue.Type {
		case "BUG":
			data.Bugs++
		case "VULNERABILITY":
			data.Vulnerabilities++
		case "CODE_SMELL":
			data.CodeSmells++
		case "SECURITY_HOTSPOT":
			data.SecurityHotspots++
		}

		data.Summary.BySeverity[issue.Severity]++

		data.Issues = append(data.Issues, processedIssue)
	}

	return data, nil
}

func (s *Service) GetMetricsData(ctx context.Context, apiContext APIContext) (*MetricsData, error) {
	params := make(map[string]string)
	for k, v := range apiContext.BaseParams {
		params[k] = v
	}

	params["metricKeys"] = "duplicated_lines_density,reliability_rating,security_rating,sqale_rating"

	var measure ComponentMeasure
	if err := s.client.GetJSON(ctx, "measures/component", params, apiContext, &measure); err != nil {
		return nil, err
	}

	data := &MetricsData{
		Available: true,
		Metrics:   make(map[string]string),
		Ratings:   make(map[string]string),
	}

	for _, metric := range measure.Component.Measures {
		switch metric.Metric {
		case "duplicated_lines_density":
			if dup, err := strconv.ParseFloat(metric.Value, 64); err == nil {
				data.Duplication = dup
			}
		case "reliability_rating", "security_rating", "sqale_rating":
			data.Ratings[metric.Metric] = metric.Value
		default:
			data.Metrics[metric.Metric] = metric.Value
		}
	}

	return data, nil
}


func (s *Service) getMetricDisplayName(metricKey string) string {
	displayNames := map[string]string{
		"new_coverage":                 "Coverage on New Code",
		"coverage":                     "Coverage",
		"new_bugs":                     "Bugs on New Code",
		"bugs":                         "Bugs",
		"new_vulnerabilities":          "Vulnerabilities on New Code",
		"vulnerabilities":              "Vulnerabilities",
		"new_code_smells":              "Code Smells on New Code",
		"code_smells":                  "Code Smells",
		"new_security_hotspots":        "Security Hotspots on New Code",
		"security_hotspots":            "Security Hotspots",
		"duplicated_lines_density":     "Duplicated Lines",
		"new_duplicated_lines_density": "Duplicated Lines on New Code",
		"sqale_rating":                 "Maintainability Rating",
		"reliability_rating":           "Reliability Rating",
		"security_rating":              "Security Rating",
	}

	if displayName, exists := displayNames[metricKey]; exists {
		return displayName
	}

	return metricKey
}

func (s *Service) extractFileFromComponent(component string) string {
	parts := strings.SplitN(component, ":", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return component
}

func (s *Service) GetProjectKeyStrategies() []string {
	return s.discovery.GetProjectKeyStrategies()
}

func (s *Service) TestConnection(ctx context.Context) error {
	params := map[string]string{}
	apiContext := APIContext{
		ProjectKey: "test",
		BaseParams: map[string]string{},
	}

	_, err := s.client.Request(ctx, "GET", "system/status", params, apiContext)
	return err
}
