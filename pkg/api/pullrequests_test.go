package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Test data for pull request responses
const samplePullRequestJSON = `{
	"type": "pullrequest",
	"id": 123,
	"title": "Fix login bug",
	"description": "This PR fixes the login issue reported in #456",
	"state": "OPEN",
	"author": {
		"type": "user",
		"username": "developer",
		"display_name": "Developer User"
	},
	"source": {
		"branch": {
			"name": "feature/fix-login"
		},
		"commit": {
			"hash": "abc123def456"
		},
		"repository": {
			"name": "my-repo",
			"full_name": "workspace/my-repo"
		}
	},
	"destination": {
		"branch": {
			"name": "main"
		},
		"commit": {
			"hash": "def456abc123"
		},
		"repository": {
			"name": "my-repo",
			"full_name": "workspace/my-repo"
		}
	},
	"comment_count": 5,
	"task_count": 2,
	"close_source_branch": true,
	"created_on": "2025-01-01T12:00:00.000Z",
	"updated_on": "2025-01-02T12:00:00.000Z",
	"reviewers": [
		{
			"type": "participant",
			"user": {
				"username": "reviewer1"
			},
			"role": "REVIEWER",
			"approved": true,
			"state": "approved"
		}
	]
}`

const samplePullRequestListJSON = `{
	"size": 2,
	"page": 1,
	"pagelen": 10,
	"next": null,
	"previous": null,
	"values": [
		` + samplePullRequestJSON + `,
		{
			"type": "pullrequest",
			"id": 124,
			"title": "Update documentation",
			"state": "MERGED",
			"author": {
				"username": "doc-writer"
			}
		}
	]
}`

const sampleDiffJSON = `diff --git a/src/auth.go b/src/auth.go
index 1234567..abcdefg 100644
--- a/src/auth.go
+++ b/src/auth.go
@@ -10,7 +10,7 @@ func Login(username, password string) error {
 	if username == "" {
 		return errors.New("username required")
 	}
-	if password == "" {
+	if password == "" || len(password) < 8 {
 		return errors.New("password required")
 	}
 	return authenticateUser(username, password)
`

const sampleDiffStatJSON = `{
	"type": "diffstat",
	"status": "modified",
	"lines_added": 15,
	"lines_removed": 3,
	"files_changed": 2,
	"files": [
		{
			"type": "file",
			"status": "modified",
			"old_path": "src/auth.go",
			"new_path": "src/auth.go",
			"lines_added": 10,
			"lines_removed": 2,
			"binary": false
		},
		{
			"type": "file", 
			"status": "modified",
			"old_path": "src/login.go",
			"new_path": "src/login.go",
			"lines_added": 5,
			"lines_removed": 1,
			"binary": false
		}
	]
}`

// MockAuthManager is already defined in client_test.go

func TestPullRequestService_ListPullRequestsValidation(t *testing.T) {
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       "https://api.bitbucket.org/2.0",
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test validation errors
	_, err = client.PullRequests.ListPullRequests(ctx, "", "repo", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace")

	_, err = client.PullRequests.ListPullRequests(ctx, "workspace", "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository slug")
}

func TestPullRequestService_ListPullRequests(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests", r.URL.Path)
		
		// Check query parameters
		query := r.URL.Query()
		if state := query.Get("state"); state != "" {
			assert.Equal(t, "OPEN", state)
		}
		if author := query.Get("author.username"); author != "" {
			assert.Equal(t, "test-author", author)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(samplePullRequestListJSON))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	// Set up mock expectations
	mockAuth.On("SetHTTPHeaders", mock.Anything).Return(nil)
	
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test listing with options
	options := &PullRequestListOptions{
		State:  "OPEN",
		Author: "test-author",
		Sort:   "-updated_on",
	}

	result, err := client.PullRequests.ListPullRequests(ctx, "test-workspace", "test-repo", options)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.Size)

	// Verify the pull requests in the response
	var pullRequests []PullRequest
	for _, value := range result.Values {
		prData, err := json.Marshal(value)
		require.NoError(t, err)
		
		var pr PullRequest
		err = json.Unmarshal(prData, &pr)
		require.NoError(t, err)
		pullRequests = append(pullRequests, pr)
	}

	assert.Len(t, pullRequests, 2)
	assert.Equal(t, 123, pullRequests[0].ID)
	assert.Equal(t, "Fix login bug", pullRequests[0].Title)
	assert.Equal(t, "OPEN", pullRequests[0].State)
}

func TestPullRequestService_GetPullRequestValidation(t *testing.T) {
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       "https://api.bitbucket.org/2.0",
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test validation errors
	_, err = client.PullRequests.GetPullRequest(ctx, "", "repo", 123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "workspace")

	_, err = client.PullRequests.GetPullRequest(ctx, "workspace", "", 123)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository slug")

	_, err = client.PullRequests.GetPullRequest(ctx, "workspace", "repo", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pull request ID must be positive")

	_, err = client.PullRequests.GetPullRequest(ctx, "workspace", "repo", -1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pull request ID must be positive")
}

func TestPullRequestService_GetPullRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(samplePullRequestJSON))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting a pull request
	pr, err := client.PullRequests.GetPullRequest(ctx, "test-workspace", "test-repo", 123)
	require.NoError(t, err)
	assert.NotNil(t, pr)
	assert.Equal(t, 123, pr.ID)
	assert.Equal(t, "Fix login bug", pr.Title)
	assert.Equal(t, "OPEN", pr.State)
	assert.Equal(t, "developer", pr.Author.Username)
	assert.Equal(t, "feature/fix-login", pr.Source.Branch.Name)
	assert.Equal(t, "main", pr.Destination.Branch.Name)
	assert.Equal(t, 5, pr.CommentCount)
	assert.True(t, pr.CloseSourceBranch)
}

func TestPullRequestService_GetPullRequestDiff(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123/diff", r.URL.Path)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleDiffJSON))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting diff
	diff, err := client.PullRequests.GetPullRequestDiff(ctx, "test-workspace", "test-repo", 123)
	require.NoError(t, err)
	assert.NotEmpty(t, diff)
	assert.Contains(t, diff, "diff --git a/src/auth.go b/src/auth.go")
	assert.Contains(t, diff, "+	if password == \"\" || len(password) < 8 {")
}

func TestPullRequestService_GetPullRequestFiles(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123/diffstat", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleDiffStatJSON))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test getting files
	diffStat, err := client.PullRequests.GetPullRequestFiles(ctx, "test-workspace", "test-repo", 123)
	require.NoError(t, err)
	assert.NotNil(t, diffStat)
	assert.Equal(t, 15, diffStat.LinesAdded)
	assert.Equal(t, 3, diffStat.LinesRemoved)
	assert.Equal(t, 2, diffStat.FilesChanged)
	assert.Len(t, diffStat.Files, 2)
	assert.Equal(t, "src/auth.go", diffStat.Files[0].OldPath)
	assert.Equal(t, "modified", diffStat.Files[0].Status)
}

func TestPullRequestService_ApprovePullRequest(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123/approve", r.URL.Path)

		approvalResponse := `{
			"type": "pullrequest_approval",
			"user": {
				"username": "reviewer"
			},
			"date": "2025-01-02T15:30:00.000Z"
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(approvalResponse))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test approval
	approval, err := client.PullRequests.ApprovePullRequest(ctx, "test-workspace", "test-repo", 123)
	require.NoError(t, err)
	assert.NotNil(t, approval)
	assert.Equal(t, "pullrequest_approval", approval.Type)
	assert.Equal(t, "reviewer", approval.User.Username)
}

func TestPullRequestService_RequestChanges(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123/request-changes", r.URL.Path)

		// Verify request body
		var requestBody RequestChangesRequest
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)
		assert.Equal(t, "pullrequest_comment", requestBody.Type)
		assert.Equal(t, "Please fix the validation logic", requestBody.Content.Raw)

		commentResponse := `{
			"type": "pullrequest_comment",
			"id": 456,
			"content": {
				"raw": "Please fix the validation logic"
			},
			"user": {
				"username": "reviewer"
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(commentResponse))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test requesting changes
	comment, err := client.PullRequests.RequestChanges(ctx, "test-workspace", "test-repo", 123, "Please fix the validation logic")
	require.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, 456, comment.ID)
	assert.Equal(t, "Please fix the validation logic", comment.Content.Raw)
}

func TestPullRequestService_AddComment(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123/comments", r.URL.Path)

		// Verify request body
		var requestBody AddCommentRequest
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)
		assert.Equal(t, "pullrequest_comment", requestBody.Type)
		assert.Equal(t, "This looks good to me!", requestBody.Content.Raw)

		commentResponse := `{
			"type": "pullrequest_comment",
			"id": 789,
			"content": {
				"raw": "This looks good to me!"
			},
			"user": {
				"username": "commenter"
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(commentResponse))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test adding comment
	comment, err := client.PullRequests.AddComment(ctx, "test-workspace", "test-repo", 123, "This looks good to me!", nil)
	require.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, 789, comment.ID)
	assert.Equal(t, "This looks good to me!", comment.Content.Raw)
}

func TestPullRequestService_AddInlineComment(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/repositories/test-workspace/test-repo/pullrequests/123/comments", r.URL.Path)

		// Verify request body includes inline information
		var requestBody AddCommentRequest
		err := json.NewDecoder(r.Body).Decode(&requestBody)
		require.NoError(t, err)
		assert.Equal(t, "src/auth.go", requestBody.Inline.Path)
		assert.Equal(t, 15, requestBody.Inline.To)

		commentResponse := `{
			"type": "pullrequest_comment",
			"id": 999,
			"content": {
				"raw": "Consider using a constant for minimum password length"
			},
			"inline": {
				"path": "src/auth.go",
				"to": 15
			},
			"user": {
				"username": "reviewer"
			}
		}`

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(commentResponse))
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test adding inline comment
	comment, err := client.PullRequests.AddInlineComment(ctx, "test-workspace", "test-repo", 123, 
		"Consider using a constant for minimum password length", "src/auth.go", 15)
	require.NoError(t, err)
	assert.NotNil(t, comment)
	assert.Equal(t, 999, comment.ID)
	assert.NotNil(t, comment.Inline)
	assert.Equal(t, "src/auth.go", comment.Inline.Path)
	assert.Equal(t, 15, comment.Inline.To)
}

func TestPullRequestService_ConvenienceMethods(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		
		query := r.URL.Query()
		
		// Check different convenience method calls
		if path := r.URL.Path; strings.Contains(path, "pullrequests") {
			if query.Get("state") == "OPEN" && query.Get("author.username") == "test-author" {
				// ListPullRequestsByAuthor call
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(samplePullRequestListJSON))
				return
			}
			if query.Get("state") == "OPEN" && query.Get("reviewers.username") == "test-reviewer" {
				// ListPullRequestsForReviewer call
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(samplePullRequestListJSON))
				return
			}
			if query.Get("state") == "OPEN" && query.Get("sort") == "-updated_on" {
				// ListOpenPullRequests call
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(samplePullRequestListJSON))
				return
			}
		}
		
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test convenience methods
	t.Run("ListOpenPullRequests", func(t *testing.T) {
		result, err := client.PullRequests.ListOpenPullRequests(ctx, "test-workspace", "test-repo")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Size)
	})

	t.Run("ListPullRequestsByAuthor", func(t *testing.T) {
		result, err := client.PullRequests.ListPullRequestsByAuthor(ctx, "test-workspace", "test-repo", "test-author")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Size)
	})

	t.Run("ListPullRequestsForReviewer", func(t *testing.T) {
		result, err := client.PullRequests.ListPullRequestsForReviewer(ctx, "test-workspace", "test-repo", "test-reviewer")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Size)
	})
}

func TestPullRequestService_ErrorHandling(t *testing.T) {
	// Create a test server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "404") {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"type": "error", "error": {"message": "Not found"}}`))
			return
		}
		if strings.Contains(r.URL.Path, "403") {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"type": "error", "error": {"message": "Forbidden"}}`))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create client with test server
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/test",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test 404 error
	_, err = client.PullRequests.GetPullRequest(ctx, "test-workspace", "test-repo", 404)
	assert.Error(t, err)
	
	// Test 403 error
	_, err = client.PullRequests.GetPullRequest(ctx, "test-workspace", "test-repo", 403)
	assert.Error(t, err)

	// Test GetPullRequestByID convenience method with 404
	_, err = client.PullRequests.GetPullRequestByID(ctx, "test-workspace", "test-repo", 404)
	assert.Error(t, err)
	// Should be converted to BitbucketError with NotFound type
	var bbErr *BitbucketError
	if assert.ErrorAs(t, err, &bbErr) {
		assert.Equal(t, ErrorTypeNotFound, bbErr.Type)
	}
}

// Benchmark tests
func BenchmarkPullRequestService_ListPullRequests(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(samplePullRequestListJSON))
	}))
	defer server.Close()

	// Create client
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/benchmark",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(b, err)

	ctx := context.Background()

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.PullRequests.ListPullRequests(ctx, "test-workspace", "test-repo", nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPullRequestService_GetPullRequest(b *testing.B) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(samplePullRequestJSON))
	}))
	defer server.Close()

	// Create client
	mockAuth := &MockAuthManager{}
	config := &ClientConfig{
		BaseURL:       server.URL,
		Timeout:       5 * time.Second,
		RetryAttempts: 1,
		UserAgent:     "bt/benchmark",
	}

	client, err := NewClient(mockAuth, config)
	require.NoError(b, err)

	ctx := context.Background()

	// Run benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.PullRequests.GetPullRequest(ctx, "test-workspace", "test-repo", 123)
		if err != nil {
			b.Fatal(err)
		}
	}
}