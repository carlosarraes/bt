package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/auth"
	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageUpload(t *testing.T) {
	utils.RequireIntegrationEnv(t)
	if os.Getenv("BITBUCKET_SESSION_TOKEN") == "" {
		t.Skip("BITBUCKET_SESSION_TOKEN not set")
	}
	ws, repo := os.Getenv("BT_TEST_WORKSPACE"), os.Getenv("BT_TEST_REPOSITORY")
	if ws == "" || repo == "" {
		t.Skip("BT_TEST_WORKSPACE/BT_TEST_REPOSITORY not set")
	}

	sess, err := auth.LoadWebSession(os.Stderr)
	require.NoError(t, err)

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0d, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
		0x42, 0x60, 0x82,
	}
	p := filepath.Join(t.TempDir(), "bt-integration.png")
	require.NoError(t, os.WriteFile(p, png, 0o600))

	img, err := api.NewImageUploader().Upload(context.Background(), sess, ws, repo, p)
	require.NoError(t, err)
	assert.Contains(t, img.Href, "/images/")
}
