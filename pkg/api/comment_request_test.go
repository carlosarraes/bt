package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAddCommentRequest(t *testing.T) {
	parentID := 12344
	zero := 0
	negative := -1

	tests := []struct {
		name       string
		body       string
		inline     *PullRequestCommentInline
		parentID   *int
		wantParent int // 0 means no parent expected
		wantInline bool
	}{
		{
			name:     "top level comment",
			body:     "looks good",
			parentID: nil,
		},
		{
			// The original bug: --reply-to was accepted, validated, then dropped, so
			// the reply silently posted as a top-level comment.
			name:       "reply carries its parent",
			body:       "done",
			parentID:   &parentID,
			wantParent: 12344,
		},
		{
			name:       "inline comment",
			body:       "extract this",
			inline:     &PullRequestCommentInline{Path: "src/auth.go", To: 15},
			wantInline: true,
		},
		{
			name:     "zero parent id is not a parent",
			body:     "hi",
			parentID: &zero,
		},
		{
			name:     "negative parent id is not a parent",
			body:     "hi",
			parentID: &negative,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := BuildAddCommentRequest(tt.body, tt.inline, tt.parentID)

			require.NotNil(t, request.Content)
			assert.Equal(t, tt.body, request.Content.Raw)
			assert.Equal(t, "pullrequest_comment", request.Type)

			if tt.wantParent == 0 {
				assert.Nil(t, request.Parent)
			} else {
				require.NotNil(t, request.Parent, "reply must carry a parent")
				assert.Equal(t, tt.wantParent, request.Parent.ID)
			}

			if tt.wantInline {
				require.NotNil(t, request.Inline)
			} else {
				assert.Nil(t, request.Inline)
			}
		})
	}
}

func TestBuildAddCommentRequest_WireShape(t *testing.T) {
	parentID := 999

	raw, err := json.Marshal(BuildAddCommentRequest("done", nil, &parentID))
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))

	// Bitbucket wants parent as {"id": N}; sending a whole nested comment back
	// would be rejected.
	assert.Equal(t, map[string]any{"id": float64(999)}, decoded["parent"])
	assert.NotContains(t, decoded, "inline", "inline must be omitted when unset")

	topLevel, err := json.Marshal(BuildAddCommentRequest("hi", nil, nil))
	require.NoError(t, err)
	assert.NotContains(t, string(topLevel), "parent")
}
