package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/carlosarraes/bt/pkg/auth"
)

// ErrWebSessionInvalid is returned when bitbucket.org rejects the web session (401/403).
var ErrWebSessionInvalid = errors.New("Bitbucket web session expired or invalid — re-export BITBUCKET_SESSION_TOKEN and BITBUCKET_CSRF_TOKEN")

type UploadedImage struct {
	Href     string `json:"href"`
	Filename string `json:"filename"`
}

// ImageUploader uses bitbucket.org's undocumented web endpoint, which only accepts a
// browser session; it is deliberately separate from the API-token REST Client.
type ImageUploader struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewImageUploader() *ImageUploader {
	return &ImageUploader{
		BaseURL: "https://bitbucket.org",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
			// Redirects mean the session was rejected (login page); following them
			// would forward X-CSRFToken to another host.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
	}
}

var (
	htmlTagRe    = regexp.MustCompile(`<[^>]*>`)
	whitespaceRe = regexp.MustCompile(`\s+`)
	filenameRepl = strings.NewReplacer("(", "%28", ")", "%29")
)

func (u *ImageUploader) Upload(ctx context.Context, sess *auth.WebSession, workspace, repo, path string) (*UploadedImage, error) {
	data, contentType, err := readImage(path)
	if err != nil {
		return nil, err
	}

	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filenameRepl.Replace(filepath.Base(path))))
	partHeader.Set("Content-Type", contentType)
	part, err := mw.CreatePart(partHeader)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(data); err != nil {
		return nil, err
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	repoURL := fmt.Sprintf("%s/%s/%s/", u.BaseURL, workspace, repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/xhr/%s/%s/image-upload/", u.BaseURL, workspace, repo), &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Cookie", "cloud.session.token="+sess.SessionToken+"; csrftoken="+sess.CSRFToken)
	req.Header.Set("X-CSRFToken", sess.CSRFToken)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("X-Bitbucket-Frontend", "frontbucket")
	req.Header.Set("Referer", repoURL)

	resp, err := u.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload %s: %w", path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden,
		resp.StatusCode >= 300 && resp.StatusCode < 400:
		return nil, fmt.Errorf("upload %s: %w", path, ErrWebSessionInvalid)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, fmt.Errorf("upload %s: HTTP %d: %s", path, resp.StatusCode, summarizeBody(respBody))
	}

	var img UploadedImage
	if err := json.Unmarshal(respBody, &img); err != nil || img.Href == "" {
		return nil, fmt.Errorf("upload %s: unexpected response: %s", path, summarizeBody(respBody))
	}
	return &img, nil
}

// ValidateImage checks that path is a readable image file without uploading it.
func ValidateImage(path string) error {
	_, _, err := readImage(path)
	return err
}

func readImage(path string) ([]byte, string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, "", fmt.Errorf("image %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, "", fmt.Errorf("image %s: not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("image %s: %w", path, err)
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, "", fmt.Errorf("image %s: not an image (detected %s)", path, contentType)
	}
	return data, contentType, nil
}

func summarizeBody(b []byte) string {
	s := strings.TrimSpace(whitespaceRe.ReplaceAllString(htmlTagRe.ReplaceAllString(string(b), " "), " "))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
