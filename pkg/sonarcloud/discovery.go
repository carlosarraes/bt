package sonarcloud

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
)

type ProjectKeyDiscovery struct {
	bitbucketClient *api.Client
}

func NewProjectKeyDiscovery(bitbucketClient *api.Client) *ProjectKeyDiscovery {
	return &ProjectKeyDiscovery{
		bitbucketClient: bitbucketClient,
	}
}

type DiscoveryResult struct {
	ProjectKey string
	Strategy   string
	Source     string
}

func (d *ProjectKeyDiscovery) DiscoverProjectKey(ctx context.Context, workspace, repo, commitHash string) (*DiscoveryResult, error) {
	strategies := []struct {
		name string
		fn   func(ctx context.Context, workspace, repo, commitHash string) (string, string, error)
	}{
		{"Bitbucket Reports API", d.extractFromBitbucketReports},
		{"Environment Variable", d.loadFromEnvironment},
		{"Configuration File", d.loadFromConfigFile},
		{"Git Repository", d.detectFromGitRepo},
		{"Heuristic Naming", d.generateHeuristicName},
	}

	for _, strategy := range strategies {
		projectKey, source, err := strategy.fn(ctx, workspace, repo, commitHash)
		if err == nil && projectKey != "" {
			return &DiscoveryResult{
				ProjectKey: projectKey,
				Strategy:   strategy.name,
				Source:     source,
			}, nil
		}
	}

	return nil, fmt.Errorf("unable to discover SonarCloud project key for %s/%s", workspace, repo)
}

func (d *ProjectKeyDiscovery) extractFromBitbucketReports(ctx context.Context, workspace, repo, commitHash string) (string, string, error) {
	if d.bitbucketClient == nil {
		return "", "", fmt.Errorf("Bitbucket client not available")
	}

	endpoint := fmt.Sprintf("repositories/%s/%s/commit/%s/reports", workspace, repo, commitHash)

	var reports struct {
		Values []struct {
			UUID       string `json:"uuid"`
			Title      string `json:"title"`
			Details    string `json:"details"`
			ExternalID string `json:"external_id"`
			Reporter   string `json:"reporter"`
			Link       string `json:"link"`
			ReportType string `json:"report_type"`
			Result     string `json:"result"`
			Data       []struct {
				Title string `json:"title"`
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"data"`
		} `json:"values"`
	}

	if err := d.bitbucketClient.GetJSON(ctx, endpoint, &reports); err != nil {
		return "", "", fmt.Errorf("failed to get commit reports: %w", err)
	}

	for _, report := range reports.Values {
		if strings.Contains(strings.ToLower(report.Reporter), "sonar") ||
			strings.Contains(strings.ToLower(report.Title), "sonar") {

			if report.Link != "" {
				projectKey := d.extractProjectKeyFromURL(report.Link)
				if projectKey != "" {
					return projectKey, report.Link, nil
				}
			}

			if report.ExternalID != "" {
				if d.isValidProjectKey(report.ExternalID) {
					return report.ExternalID, "external_id", nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("no SonarCloud report found in commit reports")
}

func (d *ProjectKeyDiscovery) extractProjectKeyFromURL(url string) string {

	patterns := []string{
		`[?&]id=([^&]+)`,
		`/project/([^/?]+)`,
		`/dashboard/([^/?]+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(url)
		if len(matches) > 1 {
			projectKey := matches[1]
			if d.isValidProjectKey(projectKey) {
				return projectKey
			}
		}
	}

	return ""
}

func (d *ProjectKeyDiscovery) loadFromEnvironment(ctx context.Context, workspace, repo, commitHash string) (string, string, error) {
	if key := os.Getenv("SONARCLOUD_PROJECT_KEY"); key != "" {
		return key, "SONARCLOUD_PROJECT_KEY", nil
	}

	envVar := fmt.Sprintf("SONARCLOUD_PROJECT_KEY_%s_%s",
		strings.ToUpper(strings.ReplaceAll(workspace, "-", "_")),
		strings.ToUpper(strings.ReplaceAll(repo, "-", "_")))

	if key := os.Getenv(envVar); key != "" {
		return key, envVar, nil
	}

	return "", "", fmt.Errorf("no project key found in environment variables")
}

func (d *ProjectKeyDiscovery) loadFromConfigFile(ctx context.Context, workspace, repo, commitHash string) (string, string, error) {
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".config", "bt", "sonarcloud.json")
		if projectKey, err := d.loadProjectKeyFromFile(configPath, workspace, repo); err == nil {
			return projectKey, configPath, nil
		}
	}

	if projectKey, err := d.loadProjectKeyFromFile("sonarcloud.json", workspace, repo); err == nil {
		return projectKey, "sonarcloud.json", nil
	}

	return "", "", fmt.Errorf("no project key found in configuration files")
}

func (d *ProjectKeyDiscovery) loadProjectKeyFromFile(filePath, workspace, repo string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var config struct {
		Projects map[string]string `json:"projects"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return "", err
	}

	repoKey := fmt.Sprintf("%s/%s", workspace, repo)
	if projectKey, exists := config.Projects[repoKey]; exists {
		return projectKey, nil
	}

	variations := []string{
		strings.ToLower(repoKey),
		strings.ReplaceAll(repoKey, "-", "_"),
		strings.ReplaceAll(strings.ToLower(repoKey), "-", "_"),
		repo,
		strings.ToLower(repo),
		strings.ReplaceAll(repo, "-", "_"),
	}

	for _, variation := range variations {
		if projectKey, exists := config.Projects[variation]; exists {
			return projectKey, nil
		}
	}

	return "", fmt.Errorf("project not found in configuration")
}

func (d *ProjectKeyDiscovery) detectFromGitRepo(ctx context.Context, workspace, repo, commitHash string) (string, string, error) {
	propertyFiles := []string{
		"sonar-project.properties",
		".sonarcloud.properties",
		"sonar.properties",
	}

	for _, filename := range propertyFiles {
		if projectKey, err := d.extractProjectKeyFromProperties(filename); err == nil {
			return projectKey, filename, nil
		}
	}

	if projectKey, err := d.extractProjectKeyFromPackageJSON(); err == nil {
		return projectKey, "package.json", nil
	}

	return "", "", fmt.Errorf("no project key found in git repository")
}

func (d *ProjectKeyDiscovery) extractProjectKeyFromProperties(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "sonar.projectKey=") {
			projectKey := strings.TrimPrefix(line, "sonar.projectKey=")
			projectKey = strings.TrimSpace(projectKey)
			if projectKey != "" {
				return projectKey, nil
			}
		}
	}

	return "", fmt.Errorf("sonar.projectKey not found in %s", filename)
}

func (d *ProjectKeyDiscovery) extractProjectKeyFromPackageJSON() (string, error) {
	data, err := os.ReadFile("package.json")
	if err != nil {
		return "", err
	}

	var pkg struct {
		SonarJS struct {
			ProjectKey string `json:"projectKey"`
		} `json:"sonarjs"`
		Sonar struct {
			ProjectKey string `json:"projectKey"`
		} `json:"sonar"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return "", err
	}

	if pkg.SonarJS.ProjectKey != "" {
		return pkg.SonarJS.ProjectKey, nil
	}

	if pkg.Sonar.ProjectKey != "" {
		return pkg.Sonar.ProjectKey, nil
	}

	return "", fmt.Errorf("no SonarCloud project key found in package.json")
}

func (d *ProjectKeyDiscovery) generateHeuristicName(ctx context.Context, workspace, repo, commitHash string) (string, string, error) {
	projectKey := fmt.Sprintf("%s_%s", workspace, repo)

	projectKey = d.sanitizeProjectKey(projectKey)

	return projectKey, "heuristic", nil
}

func (d *ProjectKeyDiscovery) sanitizeProjectKey(key string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\-_.]`)
	sanitized := re.ReplaceAllString(key, "_")

	re = regexp.MustCompile(`_+`)
	sanitized = re.ReplaceAllString(sanitized, "_")

	sanitized = strings.Trim(sanitized, "_")

	return sanitized
}

func (d *ProjectKeyDiscovery) isValidProjectKey(key string) bool {
	if key == "" {
		return false
	}

	if len(key) < 1 || len(key) > 400 {
		return false
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9\-_.]+$`)
	return re.MatchString(key)
}

func (d *ProjectKeyDiscovery) GetProjectKeyStrategies() []string {
	return []string{
		"Bitbucket Reports API: Extract from SonarCloud report link in commit reports",
		"Environment Variable: SONARCLOUD_PROJECT_KEY or repo-specific variables",
		"Configuration File: ~/.config/bt/sonarcloud.json or ./sonarcloud.json",
		"Git Repository: sonar-project.properties, .sonarcloud.properties, package.json",
		"Heuristic Naming: {workspace}_{repository} with sanitization",
	}
}
