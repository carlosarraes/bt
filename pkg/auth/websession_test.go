package auth

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fakeJWT(exp time.Time) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"exp":%d}`, exp.Unix())))
	return "eyJhbGciOiJIUzI1NiJ9." + payload + ".sig"
}

func envMap(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadWebSession(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	valid := fakeJWT(now.Add(20 * 24 * time.Hour))
	missingFile := filepath.Join(t.TempDir(), "none")

	t.Run("env vars", func(t *testing.T) {
		var warn bytes.Buffer
		s, err := loadWebSession(envMap(map[string]string{
			"BITBUCKET_SESSION_TOKEN": valid, "BITBUCKET_CSRF_TOKEN": "csrf1",
		}), missingFile, now, &warn)
		require.NoError(t, err)
		assert.Equal(t, valid, s.SessionToken)
		assert.Equal(t, "csrf1", s.CSRFToken)
		assert.Equal(t, now.Add(20*24*time.Hour).Unix(), s.ExpiresAt.Unix())
		assert.Empty(t, warn.String())
	})

	t.Run("env wins over file", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "bb-session")
		require.NoError(t, os.WriteFile(f, []byte("Cookie: cloud.session.token=fromfile; csrftoken=filecsrf\n"), 0o600))
		s, err := loadWebSession(envMap(map[string]string{
			"BITBUCKET_SESSION_TOKEN": valid, "BITBUCKET_CSRF_TOKEN": "csrf1",
		}), f, now, &bytes.Buffer{})
		require.NoError(t, err)
		assert.Equal(t, valid, s.SessionToken)
	})

	t.Run("partial env", func(t *testing.T) {
		_, err := loadWebSession(envMap(map[string]string{"BITBUCKET_SESSION_TOKEN": valid}), missingFile, now, &bytes.Buffer{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "BITBUCKET_CSRF_TOKEN")
		assert.NotContains(t, err.Error(), valid)
	})

	t.Run("file fallback", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "bb-session")
		content := "Cookie: bb_session=x; cloud.session.token=" + valid + "; csrftoken=cookiecsrf\nX-CSRFToken: headercsrf\n"
		require.NoError(t, os.WriteFile(f, []byte(content), 0o600))
		s, err := loadWebSession(envMap(nil), f, now, &bytes.Buffer{})
		require.NoError(t, err)
		assert.Equal(t, valid, s.SessionToken)
		assert.Equal(t, "headercsrf", s.CSRFToken)
	})

	t.Run("file without header line uses csrftoken cookie", func(t *testing.T) {
		f := filepath.Join(t.TempDir(), "bb-session")
		require.NoError(t, os.WriteFile(f, []byte("Cookie: cloud.session.token="+valid+"; csrftoken=cookiecsrf\n"), 0o600))
		s, err := loadWebSession(envMap(nil), f, now, &bytes.Buffer{})
		require.NoError(t, err)
		assert.Equal(t, "cookiecsrf", s.CSRFToken)
	})

	t.Run("nothing configured", func(t *testing.T) {
		_, err := loadWebSession(envMap(nil), missingFile, now, &bytes.Buffer{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "BITBUCKET_SESSION_TOKEN")
		assert.Contains(t, err.Error(), "BITBUCKET_CSRF_TOKEN")
	})

	t.Run("expired", func(t *testing.T) {
		expired := fakeJWT(now.Add(-time.Hour))
		_, err := loadWebSession(envMap(map[string]string{
			"BITBUCKET_SESSION_TOKEN": expired, "BITBUCKET_CSRF_TOKEN": "c",
		}), missingFile, now, &bytes.Buffer{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
		assert.NotContains(t, err.Error(), expired)
	})

	t.Run("near expiry warns", func(t *testing.T) {
		var warn bytes.Buffer
		_, err := loadWebSession(envMap(map[string]string{
			"BITBUCKET_SESSION_TOKEN": fakeJWT(now.Add(48 * time.Hour)), "BITBUCKET_CSRF_TOKEN": "c",
		}), missingFile, now, &warn)
		require.NoError(t, err)
		assert.True(t, strings.Contains(warn.String(), "expires"))
	})

	t.Run("non-JWT token accepted without warning", func(t *testing.T) {
		var warn bytes.Buffer
		s, err := loadWebSession(envMap(map[string]string{
			"BITBUCKET_SESSION_TOKEN": "opaque", "BITBUCKET_CSRF_TOKEN": "c",
		}), missingFile, now, &warn)
		require.NoError(t, err)
		assert.True(t, s.ExpiresAt.IsZero())
		assert.Empty(t, warn.String())
	})
}
