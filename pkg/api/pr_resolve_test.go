package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestPRClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	mockAuth := &MockAuthManager{}
	mockAuth.On("SetHTTPHeaders", mock.Anything).Return(nil)
	client, err := NewClient(mockAuth, &ClientConfig{BaseURL: server.URL, Timeout: 5 * time.Second, RetryAttempts: 1, UserAgent: "bt/test"})
	require.NoError(t, err)
	return client
}

func TestGetPullRequestFiles_PaginatedValues(t *testing.T) {
	var serverURL string
	client := newTestPRClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repositories/w/r/pullrequests/7/diffstat", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `{"values":[{"status":"removed","lines_added":0,"lines_removed":9,"old":{"path":"gone.go"},"new":null}]}`)
			return
		}
		fmt.Fprintf(w, `{"values":[{"status":"modified","lines_added":3,"lines_removed":1,"old":{"path":"a.go"},"new":{"path":"a.go"}},{"status":"added","lines_added":5,"lines_removed":0,"old":null,"new":{"path":"b.go"}}],"next":"%s/repositories/w/r/pullrequests/7/diffstat?page=2"}`, serverURL)
	})
	serverURL = client.BaseURL()

	for name, fetch := range map[string]func() (*PullRequestDiffStat, error){
		"GetPullRequestFiles": func() (*PullRequestDiffStat, error) {
			return client.PullRequests.GetPullRequestFiles(context.Background(), "w", "r", 7)
		},
		"GetDiffstat": func() (*PullRequestDiffStat, error) {
			return client.PullRequests.GetDiffstat(context.Background(), "w", "r", 7)
		},
	} {
		t.Run(name, func(t *testing.T) {
			ds, err := fetch()
			require.NoError(t, err)
			require.Len(t, ds.Files, 3)
			assert.Equal(t, "a.go", ds.Files[0].NewPath)
			assert.Equal(t, 3, ds.Files[0].LinesAdded)
			assert.Equal(t, "b.go", ds.Files[1].NewPath)
			assert.Equal(t, "", ds.Files[1].OldPath)
			assert.Equal(t, "gone.go", ds.Files[2].OldPath)
			assert.Equal(t, "removed", ds.Files[2].Status)
			assert.Equal(t, 3, ds.FilesChanged)
			assert.Equal(t, 8, ds.LinesAdded)
			assert.Equal(t, 10, ds.LinesRemoved)
		})
	}
}

func TestResolveAndUnresolveComment(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestPRClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"type":"comment_resolution","user":{"display_name":"Ana"},"created_on":"2026-10-07T10:00:00Z"}`)
	})

	res, err := client.PullRequests.ResolveComment(context.Background(), "w", "r", 7, 42)
	require.NoError(t, err)
	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/repositories/w/r/pullrequests/7/comments/42/resolve", gotPath)
	assert.Equal(t, "Ana", res.User.DisplayName)

	require.NoError(t, client.PullRequests.UnresolveComment(context.Background(), "w", "r", 7, 42))
	assert.Equal(t, http.MethodDelete, gotMethod)
	assert.Equal(t, "/repositories/w/r/pullrequests/7/comments/42/resolve", gotPath)
}

func TestResolveComment_APIError(t *testing.T) {
	client := newTestPRClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		fmt.Fprint(w, `{"type":"error","error":{"message":"already resolved"}}`)
	})
	_, err := client.PullRequests.ResolveComment(context.Background(), "w", "r", 7, 42)
	require.Error(t, err)
}

func TestCommentDecodesResolution(t *testing.T) {
	var c PullRequestComment
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"resolution":{"type":"comment_resolution","user":{"display_name":"Rafael Sakamoto"},"created_on":"2026-10-07T10:00:00Z"}}`), &c))
	require.NotNil(t, c.Resolution)
	assert.Equal(t, "Rafael Sakamoto", c.Resolution.User.DisplayName)
	out, _ := json.Marshal(c)
	assert.Contains(t, string(out), `"resolution"`)
}
