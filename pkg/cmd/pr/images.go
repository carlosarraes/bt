package pr

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/auth"
)

type imageUploader interface {
	Upload(ctx context.Context, sess *auth.WebSession, workspace, repo, path string) (*api.UploadedImage, error)
}

var altTextEscaper = strings.NewReplacer(`[`, `\[`, `]`, `\]`)

func uploadImagesForPR(ctx context.Context, workspace, repo string, paths []string) ([]string, error) {
	for _, p := range paths {
		if err := api.ValidateImage(p); err != nil {
			return nil, err
		}
	}
	sess, err := auth.LoadWebSession(os.Stderr)
	if err != nil {
		return nil, err
	}
	return uploadImages(ctx, api.NewImageUploader(), sess, workspace, repo, paths)
}

func uploadImages(ctx context.Context, up imageUploader, sess *auth.WebSession, workspace, repo string, paths []string) ([]string, error) {
	lines := make([]string, 0, len(paths))
	for _, p := range paths {
		fmt.Fprintf(os.Stderr, "📎 Uploading %s...\n", p)
		img, err := up.Upload(ctx, sess, workspace, repo, p)
		if err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("![%s](%s)", altTextEscaper.Replace(filepath.Base(p)), img.Href))
	}
	return lines, nil
}

func appendImages(body string, lines []string) string {
	if len(lines) == 0 {
		return body
	}
	images := strings.Join(lines, "\n")
	body = strings.TrimRight(body, " \t\r\n")
	if body == "" {
		return images
	}
	return body + "\n\n" + images
}
