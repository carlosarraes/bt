package pr

import (
	"os"
	"strings"
	"testing"
)

func TestCommentCmd_parsePRID(t *testing.T) {
	tests := []struct {
		name    string
		prid    string
		want    int
		wantErr bool
	}{
		{
			name: "valid ID",
			prid: "123",
			want: 123,
		},
		{
			name: "valid ID with hash prefix",
			prid: "#123",
			want: 123,
		},
		{
			name:    "empty ID",
			prid:    "",
			wantErr: true,
		},
		{
			name:    "invalid ID",
			prid:    "abc",
			wantErr: true,
		},
		{
			name:    "negative ID",
			prid:    "-1",
			wantErr: true,
		},
		{
			name:    "zero ID",
			prid:    "0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CommentCmd{PRID: tt.prid}
			got, err := cmd.parsePRID()
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePRID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parsePRID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentCmd_getCommentBody(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		bodyFile   string
		fileContent string
		want       string
		wantErr    bool
	}{
		{
			name: "body from flag",
			body: "Test comment",
			want: "Test comment",
		},
		{
			name:        "body from file",
			bodyFile:    "test_comment.txt",
			fileContent: "Comment from file",
			want:        "Comment from file",
		},
		{
			name:     "both body and file specified",
			body:     "Test comment",
			bodyFile: "test_comment.txt",
			wantErr:  true,
		},
		{
			name:     "file does not exist",
			bodyFile: "nonexistent.txt",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CommentCmd{
				Body:     tt.body,
				BodyFile: tt.bodyFile,
			}

			if tt.bodyFile != "" && tt.fileContent != "" {
				tmpFile, err := os.CreateTemp("", "test_comment_*.txt")
				if err != nil {
					t.Fatalf("Failed to create temp file: %v", err)
				}
				defer os.Remove(tmpFile.Name())

				if _, err := tmpFile.WriteString(tt.fileContent); err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
				tmpFile.Close()

				cmd.BodyFile = tmpFile.Name()
			}

			if tt.body == "" && tt.bodyFile == "" {
				t.Skip("Interactive prompt test skipped")
			}

			got, err := cmd.getCommentBody()
			if (err != nil) != tt.wantErr {
				t.Errorf("getCommentBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCommentBody() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommentCmd_Validation(t *testing.T) {
	tests := []struct {
		name        string
		prid        string
		body        string
		bodyFile    string
		replyTo     string
		wantPRIDErr bool
		wantBodyErr bool
	}{
		{
			name: "valid comment",
			prid: "123",
			body: "Test comment",
		},
		{
			name:        "missing PR ID",
			prid:        "",
			body:        "Test comment",
			wantPRIDErr: true,
		},
		{
			name:        "invalid PR ID",
			prid:        "abc",
			body:        "Test comment",
			wantPRIDErr: true,
		},
		{
			name:     "valid with reply-to",
			prid:     "123",
			body:     "Test comment",
			replyTo:  "456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &CommentCmd{
				PRID:     tt.prid,
				Body:     tt.body,
				BodyFile: tt.bodyFile,
				ReplyTo:  tt.replyTo,
			}

			_, err := cmd.parsePRID()
			if (err != nil) != tt.wantPRIDErr {
				t.Errorf("parsePRID() error = %v, wantPRIDErr %v", err, tt.wantPRIDErr)
			}

			if tt.body != "" {
				body, err := cmd.getCommentBody()
				if (err != nil) != tt.wantBodyErr {
					t.Errorf("getCommentBody() error = %v, wantBodyErr %v", err, tt.wantBodyErr)
				}
				if err == nil && strings.TrimSpace(body) == "" {
					t.Error("getCommentBody() returned empty body without error")
				}
			}
		})
	}
}
