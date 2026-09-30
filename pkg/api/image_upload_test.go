package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Minimal valid 1x1 PNG.
var tinyPNG = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
	0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

func writeFile(t *testing.T, name string, data []byte) string {
	p := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(p, data, 0o600))
	return p
}

var testSession = &auth.WebSession{SessionToken: "SECRET_SESSION", CSRFToken: "SECRET_CSRF"}

func TestImageUploader_Upload(t *testing.T) {
	t.Run("sends expected request", func(t *testing.T) {
		var gotPath, gotFilename, gotContentType string
		var gotBody []byte
		var gotHeader http.Header
		var gotCookies map[string]string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotHeader = r.Header
			gotCookies = map[string]string{}
			for _, c := range r.Cookies() {
				gotCookies[c.Name] = c.Value
			}
			f, fh, err := r.FormFile("file")
			require.NoError(t, err)
			gotFilename = fh.Filename
			gotContentType = fh.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(f)
			json.NewEncoder(w).Encode(map[string]string{"href": "https://bitbucket.org/img/1-a.png", "filename": fh.Filename})
		}))
		defer srv.Close()

		u := &ImageUploader{BaseURL: srv.URL, HTTPClient: srv.Client()}
		img, err := u.Upload(context.Background(), testSession, "truora", "web", writeFile(t, "shot.png", tinyPNG))
		require.NoError(t, err)

		assert.Equal(t, "https://bitbucket.org/img/1-a.png", img.Href)
		assert.Equal(t, "/xhr/truora/web/image-upload/", gotPath)
		assert.Equal(t, "shot.png", gotFilename)
		assert.Equal(t, "image/png", gotContentType)
		assert.Equal(t, tinyPNG, gotBody)
		assert.Equal(t, "SECRET_SESSION", gotCookies["cloud.session.token"])
		assert.Equal(t, "SECRET_CSRF", gotCookies["csrftoken"])
		assert.Equal(t, "SECRET_CSRF", gotHeader.Get("X-CSRFToken"))
		assert.Equal(t, "XMLHttpRequest", gotHeader.Get("X-Requested-With"))
		assert.Equal(t, "frontbucket", gotHeader.Get("X-Bitbucket-Frontend"))
		assert.Equal(t, srv.URL+"/truora/web/", gotHeader.Get("Referer"))
	})

	t.Run("escapes parentheses", func(t *testing.T) {
		var gotFilename string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, fh, err := r.FormFile("file")
			require.NoError(t, err)
			gotFilename = fh.Filename
			w.Write([]byte(`{"href":"h","filename":"f"}`))
		}))
		defer srv.Close()

		u := &ImageUploader{BaseURL: srv.URL, HTTPClient: srv.Client()}
		_, err := u.Upload(context.Background(), testSession, "w", "r", writeFile(t, "shot (1).png", tinyPNG))
		require.NoError(t, err)
		assert.Equal(t, "shot %281%29.png", gotFilename)
	})

	for _, status := range []int{http.StatusUnauthorized, http.StatusForbidden} {
		t.Run(http.StatusText(status)+" maps to session error", func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				w.Write([]byte("<html>CSRF verification failed</html>"))
			}))
			defer srv.Close()

			u := &ImageUploader{BaseURL: srv.URL, HTTPClient: srv.Client()}
			_, err := u.Upload(context.Background(), testSession, "w", "r", writeFile(t, "a.png", tinyPNG))
			require.Error(t, err)
			assert.True(t, errors.Is(err, ErrWebSessionInvalid))
			assert.Contains(t, err.Error(), "BITBUCKET_SESSION_TOKEN")
			assert.NotContains(t, err.Error(), "SECRET")
		})
	}

	t.Run("other status strips html and truncates", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			w.Write([]byte("<html><body><h1>Too   big</h1></body></html>"))
		}))
		defer srv.Close()

		u := &ImageUploader{BaseURL: srv.URL, HTTPClient: srv.Client()}
		_, err := u.Upload(context.Background(), testSession, "w", "r", writeFile(t, "a.png", tinyPNG))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "413")
		assert.Contains(t, err.Error(), "Too big")
		assert.NotContains(t, err.Error(), "<h1>")
	})

	t.Run("2xx without href", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		u := &ImageUploader{BaseURL: srv.URL, HTTPClient: srv.Client()}
		_, err := u.Upload(context.Background(), testSession, "w", "r", writeFile(t, "a.png", tinyPNG))
		require.Error(t, err)
	})

	t.Run("rejects non-image without request", func(t *testing.T) {
		called := false
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
		defer srv.Close()

		u := &ImageUploader{BaseURL: srv.URL, HTTPClient: srv.Client()}
		p := writeFile(t, "notes.txt", []byte("hello"))
		_, err := u.Upload(context.Background(), testSession, "w", "r", p)
		require.Error(t, err)
		assert.Contains(t, err.Error(), p)
		assert.False(t, called)
	})

	t.Run("missing file and directory", func(t *testing.T) {
		u := &ImageUploader{BaseURL: "http://unused", HTTPClient: http.DefaultClient}
		missing := filepath.Join(t.TempDir(), "nope.png")
		_, err := u.Upload(context.Background(), testSession, "w", "r", missing)
		require.Error(t, err)
		assert.Contains(t, err.Error(), missing)

		dir := t.TempDir()
		_, err = u.Upload(context.Background(), testSession, "w", "r", dir)
		require.Error(t, err)
		assert.Contains(t, err.Error(), dir)
	})
}
