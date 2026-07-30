package cmd

import (
	"reflect"
	"strings"
	"testing"
)

// subcommandNames walks a Kong command-group struct and returns the CLI names of
// its subcommands, honouring explicit `cmd:"name"` overrides.
func subcommandNames(group any) []string {
	t := reflect.TypeOf(group)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var names []string
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		tag, ok := field.Tag.Lookup("cmd")
		if !ok {
			continue
		}

		name := tag
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		names = append(names, name)
	}
	return names
}

// The report that prompted this test came from an agent reading `bt pr --llm`:
// `pr comments` and `pr review-history` existed in the command tree but were absent
// from the LLM guide, so the agent concluded bt could not read comments at all.
// Undocumented commands are invisible commands - fail the build instead.
func TestLLMHelpCoversAllCommands(t *testing.T) {
	tests := []struct {
		group string
		cmd   any
		help  string
	}{
		{"pr", &PRCmd{}, prLLMHelpText()},
		{"run", &RunCmd{}, runLLMHelpText()},
	}

	for _, tt := range tests {
		t.Run(tt.group, func(t *testing.T) {
			names := subcommandNames(tt.cmd)
			if len(names) == 0 {
				t.Fatalf("no subcommands discovered for %q - reflection walk is broken", tt.group)
			}

			for _, name := range names {
				invocation := "bt " + tt.group + " " + name
				if !strings.Contains(tt.help, invocation) {
					t.Errorf("%q is missing from `bt %s --llm`; add it to %sLLMHelpText so agents can discover it",
						invocation, tt.group, tt.group)
				}
			}
		})
	}
}

// Flags that unlock a capability are as easy to miss as whole commands.
func TestLLMHelpDocumentsKeyFlags(t *testing.T) {
	tests := []struct {
		name string
		help string
		want []string
	}{
		{
			name: "pr",
			help: prLLMHelpText(),
			want: []string{"--reply-to", "--comments", "--author", "--file", "--line"},
		},
		{
			name: "run",
			help: runLLMHelpText(),
			want: []string{"--branch", "--commit", "--event"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, flag := range tt.want {
				if !strings.Contains(tt.help, flag) {
					t.Errorf("%s is not documented in `bt %s --llm`", flag, tt.name)
				}
			}
		})
	}
}

func TestSubcommandNames_HonoursCmdTagOverride(t *testing.T) {
	names := subcommandNames(&PRCmd{})

	joined := strings.Join(names, ",")
	for _, want := range []string{"list-all", "review-history", "update-branch"} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected explicit cmd tag %q in %v", want, names)
		}
	}
	if strings.Contains(joined, "reviewhistory") {
		t.Errorf("field name leaked instead of cmd tag: %v", names)
	}
}
