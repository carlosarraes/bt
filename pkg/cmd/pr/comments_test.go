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

func TestBuildCommentThreads(t *testing.T) {
	comment := func(id int, parentID int) api.PullRequestComment {
		c := api.PullRequestComment{ID: id}
		if parentID != 0 {
			c.Parent = &api.PullRequestComment{ID: parentID}
		}
		return c
	}

	t.Run("nests replies under their parent", func(t *testing.T) {
		roots := buildCommentThreads([]api.PullRequestComment{
			comment(1, 0),
			comment(2, 1),
			comment(3, 0),
			comment(4, 2),
		})

		if len(roots) != 2 {
			t.Fatalf("expected 2 roots, got %d", len(roots))
		}
		if roots[0].comment.ID != 1 || roots[1].comment.ID != 3 {
			t.Fatalf("unexpected root order: %d, %d", roots[0].comment.ID, roots[1].comment.ID)
		}
		if len(roots[0].children) != 1 || roots[0].children[0].comment.ID != 2 {
			t.Fatalf("comment 2 should be a child of 1")
		}
		if len(roots[0].children[0].children) != 1 || roots[0].children[0].children[0].comment.ID != 4 {
			t.Fatalf("comment 4 should be a grandchild of 1")
		}
	})

	t.Run("orphaned reply is promoted to a root", func(t *testing.T) {
		// Parent 99 was deleted or filtered out by --author; the reply must still show.
		roots := buildCommentThreads([]api.PullRequestComment{comment(5, 99)})

		if len(roots) != 1 || roots[0].comment.ID != 5 {
			t.Fatalf("orphaned reply should be a root, got %+v", roots)
		}
	})

	t.Run("self parent does not nest", func(t *testing.T) {
		roots := buildCommentThreads([]api.PullRequestComment{comment(7, 7)})

		if len(roots) != 1 || len(roots[0].children) != 0 {
			t.Fatalf("self-parented comment should render flat, got %+v", roots)
		}
	})

	t.Run("parent cycle falls back to a flat list", func(t *testing.T) {
		roots := buildCommentThreads([]api.PullRequestComment{comment(8, 9), comment(9, 8)})

		if len(roots) != 2 {
			t.Fatalf("cycle must not hide comments, got %d roots", len(roots))
		}
		for _, r := range roots {
			if len(r.children) != 0 {
				t.Fatalf("flat fallback should have no children")
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		if roots := buildCommentThreads(nil); len(roots) != 0 {
			t.Fatalf("expected no roots, got %d", len(roots))
		}
	})
}

func TestInlineAnchor(t *testing.T) {
	tests := []struct {
		name   string
		inline *api.PullRequestCommentInline
		want   string
	}{
		{"new side", &api.PullRequestCommentInline{Path: "a.go", To: 12}, "a.go:12"},
		{"old side", &api.PullRequestCommentInline{Path: "a.go", From: 7}, "a.go:7"},
		{"file level", &api.PullRequestCommentInline{Path: "a.go"}, "a.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inlineAnchor(tt.inline); got != tt.want {
				t.Errorf("inlineAnchor() = %q, want %q", got, tt.want)
			}
		})
	}
}
