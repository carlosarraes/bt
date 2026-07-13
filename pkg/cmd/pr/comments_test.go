package pr

import (
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentsCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name    string
		prid    string
		want    int
		wantErr bool
	}{
		{
			name: "valid number",
			prid: "123",
			want: 123,
		},
		{
			name: "valid number with hash prefix",
			prid: "#456",
			want: 456,
		},
		{
			name:    "empty string",
			prid:    "",
			wantErr: true,
		},
		{
			name:    "invalid number",
			prid:    "abc",
			wantErr: true,
		},
		{
			name:    "negative number",
			prid:    "-123",
			wantErr: true,
		},
		{
			name:    "zero",
			prid:    "0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CommentsCmd{PRID: tt.prid}
			got, err := cmd.ParsePRID()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFilterDeletedComments(t *testing.T) {
	tests := []struct {
		name     string
		comments []api.PullRequestComment
		wantIDs  []int
	}{
		{
			name: "removes deleted comments",
			comments: []api.PullRequestComment{
				{ID: 1},
				{ID: 2, Deleted: true},
				{ID: 3},
			},
			wantIDs: []int{1, 3},
		},
		{
			name: "all deleted",
			comments: []api.PullRequestComment{
				{ID: 1, Deleted: true},
			},
			wantIDs: []int{},
		},
		{
			name:     "nil input returns empty slice",
			comments: nil,
			wantIDs:  []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterDeletedComments(tt.comments)

			require.NotNil(t, got)
			gotIDs := make([]int, 0, len(got))
			for _, c := range got {
				gotIDs = append(gotIDs, c.ID)
			}
			assert.Equal(t, tt.wantIDs, gotIDs)
		})
	}
}
