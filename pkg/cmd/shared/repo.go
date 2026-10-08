package shared

import (
	"context"
	"fmt"
	"strings"
)

type repoOverrideKey struct{}

type repoOverride struct{ workspace, repository string }

// WithRepoOverride stores the global -R/--repo target so every command uses it
// instead of inferring the repository from the current directory's git remote.
func WithRepoOverride(ctx context.Context, workspace, repository string) context.Context {
	return context.WithValue(ctx, repoOverrideKey{}, repoOverride{workspace, repository})
}

func getRepoOverride(ctx context.Context) (repoOverride, bool) {
	o, ok := ctx.Value(repoOverrideKey{}).(repoOverride)
	return o, ok
}

// ParseRepoFlag accepts "workspace/repo", optionally with a trailing ".git", or a
// pasted bitbucket.org HTTPS/SSH URL (anything after the repo segment is ignored).
func ParseRepoFlag(value string) (string, string, error) {
	v := strings.TrimSpace(value)
	isURL := false
	for _, prefix := range []string{"https://bitbucket.org/", "http://bitbucket.org/", "bitbucket.org/", "git@bitbucket.org:"} {
		if strings.HasPrefix(v, prefix) {
			v = strings.TrimPrefix(v, prefix)
			isURL = true
			break
		}
	}

	parts := strings.Split(strings.Trim(v, "/"), "/")
	if isURL && len(parts) > 2 {
		parts = parts[:2]
	}
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.Contains(v, "://") {
		return "", "", fmt.Errorf("invalid --repo %q: expected workspace/repo (e.g. truora/api)", value)
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}
