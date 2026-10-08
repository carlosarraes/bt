package pr

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	rafael = &api.User{DisplayName: "Rafael Sakamoto", Nickname: "rsakamoto", AccountID: "557058:abc", Username: "rafael-s"}
	ana    = &api.User{DisplayName: "Ana Lima", AccountID: "557058:def"}
)

func TestUserMatches(t *testing.T) {
	cases := []struct {
		selector string
		want     bool
	}{
		{"Rafael", true},
		{"rafael", true},
		{"sakamoto", true},
		{"Rafael Sakamoto", true},
		{"RSAKA", true},
		{"557058:abc", true},
		{"557058", false},
		{"rafael-s", true},
		{"lima", false},
		{"", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, userMatches(rafael, c.selector), "selector %q", c.selector)
	}
}

func threadFixture() []api.PullRequestComment {
	return []api.PullRequestComment{
		{ID: 1, User: rafael, Resolution: &api.CommentResolution{User: ana}},
		{ID: 2, User: ana, Parent: &api.PullRequestComment{ID: 1}},
		{ID: 3, User: ana, Parent: &api.PullRequestComment{ID: 2}},
		{ID: 4, User: rafael},
		{ID: 5, User: ana, Parent: &api.PullRequestComment{ID: 4}},
		{ID: 6, User: ana},
	}
}

func ids(comments []api.PullRequestComment) []int {
	out := make([]int, 0, len(comments))
	for _, c := range comments {
		out = append(out, c.ID)
	}
	return out
}

func TestFilterThreadsByResolution(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3}, ids(filterThreadsByResolution(threadFixture(), true)))
	assert.Equal(t, []int{4, 5, 6}, ids(filterThreadsByResolution(threadFixture(), false)))
}

func TestResolutionLabel(t *testing.T) {
	assert.Equal(t, "✓ resolved by Ana Lima", resolutionLabel(threadFixture()[0]))
	assert.Equal(t, "", resolutionLabel(threadFixture()[3]))
	assert.Equal(t, "✓ resolved", resolutionLabel(api.PullRequestComment{Resolution: &api.CommentResolution{}}))
}

func TestUnresolvedThreadsFrom(t *testing.T) {
	assert.Equal(t, []int{4}, ids(unresolvedThreadsFrom(threadFixture(), "rafael")))
	assert.Equal(t, []int{6}, ids(unresolvedThreadsFrom(threadFixture(), "ana")))
	assert.Empty(t, unresolvedThreadsFrom(threadFixture(), "nobody"))
}

type fakeResolver struct {
	resolved, unresolved []int
	failOn               int
}

func (f *fakeResolver) ResolveComment(_ context.Context, _, _ string, _, cid int) (*api.CommentResolution, error) {
	if cid == f.failOn {
		return nil, errors.New("conflict")
	}
	f.resolved = append(f.resolved, cid)
	return &api.CommentResolution{}, nil
}

func (f *fakeResolver) UnresolveComment(_ context.Context, _, _ string, _, cid int) error {
	if cid == f.failOn {
		return errors.New("conflict")
	}
	f.unresolved = append(f.unresolved, cid)
	return nil
}

func TestApplyResolution(t *testing.T) {
	var out bytes.Buffer
	f := &fakeResolver{failOn: 8}
	err := applyResolution(context.Background(), f, "w", "r", 2060, []int{7, 8, 9}, false, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "1 of 3")
	assert.Equal(t, []int{7, 9}, f.resolved)
	assert.Contains(t, out.String(), "#7")
	assert.Contains(t, out.String(), "#8")
	assert.Contains(t, out.String(), "conflict")

	out.Reset()
	f = &fakeResolver{}
	require.NoError(t, applyResolution(context.Background(), f, "w", "r", 2060, []int{7}, true, &out))
	assert.Equal(t, []int{7}, f.unresolved)
	assert.Contains(t, out.String(), "Reopened")
}

func TestParseCommentIDs(t *testing.T) {
	got, err := parseCommentIDs([]string{"12", "#34"})
	require.NoError(t, err)
	assert.Equal(t, []int{12, 34}, got)
	_, err = parseCommentIDs([]string{"x"})
	assert.Error(t, err)
	_, err = parseCommentIDs([]string{"0"})
	assert.Error(t, err)
}
