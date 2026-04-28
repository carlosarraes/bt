package sonarcloud

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, srvURL string) *Client {
	t.Helper()
	c, err := NewClient(&ClientConfig{
		BaseURL:       srvURL + "/api",
		Token:         "test-token",
		Timeout:       5 * time.Second,
		RetryAttempts: 0,
		EnableCache:   false,
		UserAgent:     "bt-test",
	})
	require.NoError(t, err)
	return c
}

const actionableJSON = `{
  "total": 1,
  "issues": [{
    "key": "K1",
    "type": "CODE_SMELL",
    "severity": "MAJOR",
    "status": "OPEN",
    "message": "Define a constant instead of duplicating this literal.",
    "rule": "python:S1192",
    "component": "proj:pagamentos/models.py",
    "line": 570
  }],
  "rules": [{"key":"python:S1192","name":"String literals should not be duplicated","lang":"py"}]
}`

const acceptedJSON = `{
  "total": 2,
  "issues": [
    {"key":"K2","status":"RESOLVED","resolution":"FALSE_POSITIVE","type":"CODE_SMELL","severity":"MINOR","message":"m2","rule":"python:S1192","component":"proj:foo.py","line":20},
    {"key":"K3","status":"RESOLVED","resolution":"WONT_FIX","type":"BUG","severity":"MAJOR","message":"m3","rule":"python:S2","component":"proj:bar.py","line":30,"assignee":"alice"}
  ],
  "rules": [
    {"key":"python:S1192","name":"n","lang":"py"},
    {"key":"python:S2","name":"n2","lang":"py"}
  ]
}`

func TestGetPRIssueBuckets(t *testing.T) {
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(r.URL.RawQuery, "resolved=false"):
			_, _ = w.Write([]byte(actionableJSON))
		case strings.Contains(r.URL.RawQuery, "resolved=true"):
			_, _ = w.Write([]byte(acceptedJSON))
		default:
			t.Fatalf("unexpected request without resolved filter: %s", r.URL.RawQuery)
		}
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{
		ProjectKey:    "proj",
		BaseParams:    map[string]string{"component": "proj", "pullRequest": "42"},
		IsPullRequest: true,
		PullRequestID: 42,
	}

	buckets, err := svc.GetPRIssueBuckets(context.Background(), apiCtx, FilterOptions{Limit: 10})
	require.NoError(t, err)
	require.NotNil(t, buckets)

	require.Len(t, calls, 2, "expected exactly two issues/search calls")
	resolvedFalseSeen, resolvedTrueSeen := false, false
	for _, q := range calls {
		if strings.Contains(q, "resolved=false") {
			resolvedFalseSeen = true
			assert.Contains(t, q, "issueStatuses=OPEN%2CCONFIRMED%2CREOPENED")
		}
		if strings.Contains(q, "resolved=true") {
			resolvedTrueSeen = true
		}
	}
	assert.True(t, resolvedFalseSeen, "actionable query missing")
	assert.True(t, resolvedTrueSeen, "accepted query missing")

	require.Len(t, buckets.Actionable.Issues, 1)
	act := buckets.Actionable.Issues[0]
	assert.Equal(t, "MAJOR", act.Severity)
	assert.Equal(t, "OPEN", act.Status)
	assert.Empty(t, act.Resolution)
	assert.True(t, act.IsNew, "PR-scoped issues must be marked IsNew")
	assert.Equal(t, "pagamentos/models.py", act.File)
	assert.Equal(t, "py", act.RuleLang)

	require.Len(t, buckets.Accepted.Issues, 2)
	for _, iss := range buckets.Accepted.Issues {
		assert.NotEmpty(t, iss.Resolution, "accepted bucket issues must have a resolution")
	}
	assert.Equal(t, "FALSE_POSITIVE", buckets.Accepted.Issues[0].Resolution)
	assert.Equal(t, "WONT_FIX", buckets.Accepted.Issues[1].Resolution)
	assert.Equal(t, "alice", buckets.Accepted.Issues[1].Assignee)
}

func TestGetPRIssueBuckets_PropagatesErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"errors":[{"msg":"oops"}]}`))
	}))
	defer srv.Close()

	svc := &Service{client: newTestClient(t, srv.URL)}
	apiCtx := APIContext{ProjectKey: "p", BaseParams: map[string]string{"component": "p"}, IsPullRequest: true, PullRequestID: 1}
	_, err := svc.GetPRIssueBuckets(context.Background(), apiCtx, FilterOptions{})
	require.Error(t, err)
}
