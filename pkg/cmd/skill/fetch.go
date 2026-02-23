package skill

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const repoBaseURL = "https://raw.githubusercontent.com/carlosarraes/bt/main/"

var skillFiles = []string{
	"skills/bt/SKILL.md",
	"skills/bt/references/flags.md",
}

func skillDir() (string, error) {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configDir, "bt", "skills"), nil
}

func fetchRemoteVersion(timeout time.Duration) (string, error) {
	client := &http.Client{Timeout: timeout}
	resp, err := client.Get(repoBaseURL + "skills/version")
	if err != nil {
		return "", fmt.Errorf("failed to fetch version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d fetching version", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read version: %w", err)
	}
	return strings.TrimSpace(string(body)), nil
}

func fetchAndStore() (string, error) {
	dir, err := skillDir()
	if err != nil {
		return "", err
	}

	version, err := fetchRemoteVersion(10 * time.Second)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for _, f := range skillFiles {
		body, err := fetchFile(client, f)
		if err != nil {
			return "", err
		}

		relPath := strings.TrimPrefix(f, "skills/")
		destPath := filepath.Join(dir, relPath)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}
		if err := os.WriteFile(destPath, body, 0644); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", destPath, err)
		}
	}

	versionPath := filepath.Join(dir, ".version")
	if err := os.WriteFile(versionPath, []byte(version+"\n"), 0644); err != nil {
		return "", fmt.Errorf("failed to write version: %w", err)
	}

	return version, nil
}

func fetchFile(client *http.Client, path string) ([]byte, error) {
	resp, err := client.Get(repoBaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching %s", resp.StatusCode, path)
	}
	return io.ReadAll(resp.Body)
}

func installedVersion() (string, error) {
	dir, err := skillDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, ".version"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func isInstalled() bool {
	_, err := installedVersion()
	return err == nil
}
