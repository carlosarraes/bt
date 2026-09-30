package auth

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	EnvSessionToken = "BITBUCKET_SESSION_TOKEN"
	EnvCSRFToken    = "BITBUCKET_CSRF_TOKEN"

	sessionExpiryWarning = 3 * 24 * time.Hour
)

type WebSession struct {
	SessionToken string
	CSRFToken    string
	ExpiresAt    time.Time
}

func LoadWebSession(warn io.Writer) (*WebSession, error) {
	home, _ := os.UserHomeDir()
	return loadWebSession(os.Getenv, filepath.Join(home, ".config", "bt", "bb-session"), time.Now(), warn)
}

func loadWebSession(getenv func(string) string, sessionFile string, now time.Time, warn io.Writer) (*WebSession, error) {
	token, csrf := getenv(EnvSessionToken), getenv(EnvCSRFToken)

	switch {
	case token != "" && csrf != "":
	case token != "":
		return nil, fmt.Errorf("%s is set but %s is missing", EnvSessionToken, EnvCSRFToken)
	case csrf != "":
		return nil, fmt.Errorf("%s is set but %s is missing", EnvCSRFToken, EnvSessionToken)
	default:
		var err error
		token, csrf, err = readSessionFile(sessionFile)
		if err != nil {
			return nil, fmt.Errorf("no Bitbucket web session: set %s and %s (cookies cloud.session.token and csrftoken from bitbucket.org) or create %s: %w",
				EnvSessionToken, EnvCSRFToken, sessionFile, err)
		}
	}

	s := &WebSession{SessionToken: token, CSRFToken: csrf, ExpiresAt: jwtExpiry(token)}
	if !s.ExpiresAt.IsZero() {
		if !now.Before(s.ExpiresAt) {
			return nil, fmt.Errorf("Bitbucket web session expired on %s — re-export %s and %s",
				s.ExpiresAt.Format(time.RFC3339), EnvSessionToken, EnvCSRFToken)
		}
		if s.ExpiresAt.Sub(now) < sessionExpiryWarning {
			fmt.Fprintf(warn, "⚠️  Bitbucket web session expires on %s — re-export it soon\n", s.ExpiresAt.Format(time.RFC3339))
		}
	}
	return s, nil
}

// readSessionFile parses a "Copy as cURL"-style export: a "Cookie:" line and an
// optional "X-CSRFToken:" line, which wins over the csrftoken cookie.
func readSessionFile(path string) (token, csrf string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", err
	}
	defer f.Close()

	var cookieCSRF string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		name, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "cookie":
			for _, pair := range strings.Split(value, ";") {
				k, v, _ := strings.Cut(strings.TrimSpace(pair), "=")
				switch k {
				case "cloud.session.token":
					token = v
				case "csrftoken":
					cookieCSRF = v
				}
			}
		case "x-csrftoken":
			csrf = value
		}
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}
	if csrf == "" {
		csrf = cookieCSRF
	}
	if token == "" || csrf == "" {
		return "", "", errors.New("session file lacks cloud.session.token or csrftoken")
	}
	return token, csrf, nil
}

func jwtExpiry(token string) time.Time {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}
	}
	var claims struct {
		Exp int64 `json:"exp"`
	}
	if json.Unmarshal(payload, &claims) != nil || claims.Exp == 0 {
		return time.Time{}
	}
	return time.Unix(claims.Exp, 0)
}
