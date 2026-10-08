package shared

import (
	"context"
	"testing"

	"github.com/carlosarraes/bt/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepoFlag(t *testing.T) {
	valid := map[string][2]string{
		"truora/api":                        {"truora", "api"},
		" truora/api ":                      {"truora", "api"},
		"truora/api.git":                    {"truora", "api"},
		"https://bitbucket.org/truora/web":  {"truora", "web"},
		"https://bitbucket.org/truora/web/": {"truora", "web"},
		"https://bitbucket.org/truora/web/pull-requests/2060": {"truora", "web"},
		"bitbucket.org/truora/web":                            {"truora", "web"},
		"git@bitbucket.org:truora/web.git":                    {"truora", "web"},
	}
	for in, want := range valid {
		ws, repo, err := ParseRepoFlag(in)
		require.NoError(t, err, in)
		assert.Equal(t, want[0], ws, in)
		assert.Equal(t, want[1], repo, in)
	}
	for _, in := range []string{"", "truora", "truora/", "/api", "a/b/c", "https://github.com/a/b"} {
		_, _, err := ParseRepoFlag(in)
		assert.Error(t, err, in)
	}
}

func TestResolveWorkspaceRepo_OverrideSkipsGit(t *testing.T) {
	t.Chdir(t.TempDir())
	cfg := &config.Config{}

	ws, repo, err := resolveWorkspaceRepo(WithRepoOverride(context.Background(), "truora", "api"), cfg, false)
	require.NoError(t, err)
	assert.Equal(t, "truora", ws)
	assert.Equal(t, "api", repo)

	_, _, err = resolveWorkspaceRepo(context.Background(), cfg, false)
	assert.Error(t, err)
}
