package pr

import (
	"context"
	"errors"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeUploader struct {
	calls  []string
	failOn string
}

func (f *fakeUploader) Upload(_ context.Context, _ *auth.WebSession, _, _, path string) (*api.UploadedImage, error) {
	f.calls = append(f.calls, path)
	if path == f.failOn {
		return nil, errors.New("boom " + path)
	}
	return &api.UploadedImage{Href: "https://bb/img/" + path}, nil
}

func TestUploadImages(t *testing.T) {
	up := &fakeUploader{}
	lines, err := uploadImages(context.Background(), up, &auth.WebSession{}, "w", "r", []string{"dir/a.png", "b.png"})
	require.NoError(t, err)
	assert.Equal(t, []string{"![a.png](https://bb/img/dir/a.png)", "![b.png](https://bb/img/b.png)"}, lines)
}

func TestUploadImages_StopsOnFirstError(t *testing.T) {
	up := &fakeUploader{failOn: "a.png"}
	_, err := uploadImages(context.Background(), up, &auth.WebSession{}, "w", "r", []string{"a.png", "b.png"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "a.png")
	assert.Equal(t, []string{"a.png"}, up.calls)
}

func TestImageMarkdown_EscapesAltText(t *testing.T) {
	up := &fakeUploader{}
	lines, err := uploadImages(context.Background(), up, &auth.WebSession{}, "w", "r", []string{"shot [v2].png"})
	require.NoError(t, err)
	assert.Equal(t, `![shot \[v2\].png](https://bb/img/shot [v2].png)`, lines[0])
}

func TestAppendImages(t *testing.T) {
	lines := []string{"![a](h1)", "![b](h2)"}
	assert.Equal(t, "![a](h1)\n![b](h2)", appendImages("", lines))
	assert.Equal(t, "![a](h1)\n![b](h2)", appendImages("  \n", lines))
	assert.Equal(t, "desc\n\n![a](h1)\n![b](h2)", appendImages("desc\n\n\n", lines))
	assert.Equal(t, "desc", appendImages("desc", nil))
}
