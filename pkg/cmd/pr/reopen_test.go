package pr

import (
	"testing"

	"github.com/carlosarraes/bt/pkg/api"
)

func TestReopenCmd_ParsePRID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{
			name:  "valid PR ID",
			input: "123",
			want:  123,
		},
		{
			name:  "PR ID with hash prefix",
			input: "#456",
			want:  456,
		},
		{
			name:    "empty PR ID",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid PR ID",
			input:   "abc",
			wantErr: true,
		},
		{
			name:    "zero PR ID",
			input:   "0",
			wantErr: true,
		},
		{
			name:    "negative PR ID",
			input:   "-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReopenCmd{PRID: tt.input}
			got, err := cmd.parsePRID()

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParsePRID() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParsePRID() unexpected error: %v", err)
				return
			}

			if got != tt.want {
				t.Errorf("ParsePRID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReopenCmd_validatePRState(t *testing.T) {
	tests := []struct {
		name    string
		state   string
		wantErr bool
	}{
		{
			name:  "declined PR",
			state: "DECLINED",
		},
		{
			name:    "already open PR",
			state:   "OPEN",
			wantErr: true,
		},
		{
			name:    "merged PR",
			state:   "MERGED",
			wantErr: true,
		},
		{
			name:    "superseded PR",
			state:   "SUPERSEDED",
			wantErr: true,
		},
		{
			name:    "unknown state",
			state:   "UNKNOWN",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ReopenCmd{}
			pr := &api.PullRequest{
				ID:    123,
				State: tt.state,
			}

			err := cmd.validatePRState(pr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePRState() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("validatePRState() unexpected error: %v", err)
			}
		})
	}
}
