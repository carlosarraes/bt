package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func detectShellProfile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	shell := os.Getenv("SHELL")
	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zshrc"), nil
	case strings.Contains(shell, "bash"):
		if runtime.GOOS == "darwin" {
			bp := filepath.Join(home, ".bash_profile")
			if _, err := os.Stat(bp); err == nil {
				return bp, nil
			}
		}
		return filepath.Join(home, ".bashrc"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q — please set environment variables manually:\n  export BITBUCKET_EMAIL=\"your-email\"\n  export BITBUCKET_API_TOKEN=\"your-token\"", shell)
	}
}

func profilePerm(path string) os.FileMode {
	if info, err := os.Stat(path); err == nil {
		return info.Mode().Perm()
	}
	return 0600
}

func writeEnvsToProfile(profile string, vars [][2]string) error {
	perm := profilePerm(profile)

	content, err := os.ReadFile(profile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", profile, err)
	}

	text := string(content)
	for _, kv := range vars {
		key, value := kv[0], kv[1]
		exportLine := fmt.Sprintf("export %s=%q", key, value)
		pattern := regexp.MustCompile(`(?m)^(?:export\s+)?` + regexp.QuoteMeta(key) + `=.*$`)

		if pattern.MatchString(text) {
			text = pattern.ReplaceAllString(text, exportLine)
		} else {
			if len(text) > 0 && !strings.HasSuffix(text, "\n") {
				text += "\n"
			}
			text += exportLine + "\n"
		}
	}

	return os.WriteFile(profile, []byte(text), perm)
}

func writeEnvToProfile(profile, key, value string) error {
	return writeEnvsToProfile(profile, [][2]string{{key, value}})
}

func removeEnvsFromProfile(profile string, keys []string) error {
	content, err := os.ReadFile(profile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read %s: %w", profile, err)
	}

	perm := profilePerm(profile)
	text := string(content)
	for _, key := range keys {
		pattern := regexp.MustCompile(`(?m)^(?:export\s+)?` + regexp.QuoteMeta(key) + `=.*\n?`)
		text = pattern.ReplaceAllString(text, "")
	}

	return os.WriteFile(profile, []byte(text), perm)
}
