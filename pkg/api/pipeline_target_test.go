package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlexibleID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
		want FlexibleID
	}{
		{"number", `{"id": 312}`, 312},
		{"quoted string", `{"id": "312"}`, 312},
		{"padded string", `{"id": " 312 "}`, 312},
		{"null", `{"id": null}`, 0},
		{"absent", `{}`, 0},
		{"empty string", `{"id": ""}`, 0},
		{"non-numeric tolerated", `{"id": "abc"}`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var pr PipelineTargetPullRequest
			// A malformed ID must never fail the decode of the surrounding pipeline.
			require.NoError(t, json.Unmarshal([]byte(tt.json), &pr))
			assert.Equal(t, tt.want, pr.ID)
		})
	}
}

func TestPipelineTarget_PullRequestPayload(t *testing.T) {
	payload := `{
		"type": "pipeline_pullrequest_target",
		"source": "feat/auth",
		"destination": "main",
		"destination_commit": {"hash": "aaa111"},
		"commit": {"hash": "bbb222"},
		"pullrequest": {"type": "pullrequest", "id": "312"}
	}`

	var target PipelineTarget
	require.NoError(t, json.Unmarshal([]byte(payload), &target))

	assert.True(t, target.IsPullRequest())

	n, ok := target.PRNumber()
	require.True(t, ok)
	assert.Equal(t, 312, n)

	assert.Equal(t, "feat/auth", target.BranchName())
	assert.Equal(t, "main", target.DestinationBranch())
	assert.Equal(t, "bbb222", target.CommitHash())
	assert.Equal(t, "PR #312 feat/auth→main", target.DisplayRef())
}

func TestPipelineTarget_BranchPayload(t *testing.T) {
	payload := `{
		"type": "pipeline_ref_target",
		"ref_type": "branch",
		"ref_name": "main",
		"commit": {"hash": "ccc333"}
	}`

	var target PipelineTarget
	require.NoError(t, json.Unmarshal([]byte(payload), &target))

	assert.False(t, target.IsPullRequest())
	_, ok := target.PRNumber()
	assert.False(t, ok)
	assert.Equal(t, "main", target.BranchName())
	assert.Empty(t, target.DestinationBranch())
	assert.Equal(t, "main", target.DisplayRef())
}

func TestPipelineTarget_Accessors(t *testing.T) {
	legacyID := 99

	tests := []struct {
		name       string
		target     *PipelineTarget
		wantPR     bool
		wantNumber int
		wantBranch string
		wantRef    string
	}{
		{
			name:    "nil target",
			target:  nil,
			wantRef: "-",
		},
		{
			name:    "empty target",
			target:  &PipelineTarget{},
			wantRef: "-",
		},
		{
			name:    "branch target without ref_name",
			target:  &PipelineTarget{Type: TargetTypeBranch},
			wantRef: "branch",
		},
		{
			name: "PR target missing id still identifies as PR",
			target: &PipelineTarget{
				Type:        TargetTypePullRequest,
				Source:      "feat/x",
				Destination: "main",
			},
			wantPR:     true,
			wantBranch: "feat/x",
			wantRef:    "PR feat/x→main",
		},
		{
			name: "PR target without destination",
			target: &PipelineTarget{
				Type:        TargetTypePullRequest,
				Source:      "feat/x",
				PullRequest: &PipelineTargetPullRequest{ID: 7},
			},
			wantPR:     true,
			wantNumber: 7,
			wantBranch: "feat/x",
			wantRef:    "PR #7 feat/x",
		},
		{
			name:       "legacy pull_request_id still honoured",
			target:     &PipelineTarget{PullRequestId: &legacyID},
			wantPR:     true,
			wantNumber: 99,
			wantRef:    "PR #99",
		},
		{
			name:       "ref_name wins over source",
			target:     &PipelineTarget{RefName: "main", Source: "feat/x"},
			wantBranch: "main",
			wantRef:    "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantPR, tt.target.IsPullRequest())
			assert.Equal(t, tt.wantBranch, tt.target.BranchName())
			assert.Equal(t, tt.wantRef, tt.target.DisplayRef())

			n, ok := tt.target.PRNumber()
			assert.Equal(t, tt.wantNumber != 0, ok)
			assert.Equal(t, tt.wantNumber, n)
		})
	}
}
