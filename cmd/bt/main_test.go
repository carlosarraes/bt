package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func buildBT(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "bt")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

func runBT(t *testing.T, bin string, args ...string) (string, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = t.TempDir()
	cmd.Stdin = nil
	cmd.Env = []string{"HOME=" + t.TempDir(), "PATH=" + os.Getenv("PATH")}
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("bt %v hung (likely prompting): %s", args, out)
	}
	return string(out), err
}

func TestSubcommandHelpNeverRunsCommand(t *testing.T) {
	bin := buildBT(t)
	cases := [][]string{
		{"pr", "edit", "1", "--help"},
		{"pr", "edit", "--help"},
		{"pr", "create", "--help"},
		{"pr", "comment", "5", "-h"},
		{"run", "rerun", "1", "--help"},
		{"pr", "edit", "1", "--image", "x.png", "-h"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			out, err := runBT(t, bin, args...)
			if err != nil {
				t.Fatalf("exit error %v\n%s", err, out)
			}
			if !strings.Contains(out, "Usage:") {
				t.Fatalf("expected usage output, got:\n%s", out)
			}
			if strings.Contains(out, "Error") || strings.Contains(out, "(Y/n)") {
				t.Fatalf("command ran instead of help:\n%s", out)
			}
		})
	}
}

func TestSubcommandHelpListsFlags(t *testing.T) {
	out, err := runBT(t, buildBT(t), "pr", "edit", "--help")
	if err != nil || !strings.Contains(out, "--image") {
		t.Fatalf("expected --image in help (err=%v):\n%s", err, out)
	}
}

func TestGroupHelpStillCustom(t *testing.T) {
	bin := buildBT(t)
	for _, args := range [][]string{{"pr", "--help"}, {"--help"}, {}} {
		out, err := runBT(t, bin, args...)
		if err != nil || out == "" {
			t.Fatalf("bt %v: err=%v out=%s", args, err, out)
		}
	}
}

func TestRepoFlag(t *testing.T) {
	bin := buildBT(t)

	out, err := runBT(t, bin, "pr", "list", "-R", "a/b/c")
	if err == nil || !strings.Contains(out, "invalid --repo") {
		t.Fatalf("expected invalid --repo error, err=%v:\n%s", err, out)
	}

	for _, args := range [][]string{
		{"pr", "list", "-R", "truora/api", "-h"},
		{"-R", "truora/api", "run", "list", "--help"},
		{"pr", "comments", "1", "--repo", "truora/api", "-h"},
	} {
		if out, err := runBT(t, bin, args...); err != nil || !strings.Contains(out, "Usage:") {
			t.Fatalf("bt %v: err=%v\n%s", args, err, out)
		}
	}

	// Outside any git repo and with no config, -R must still pick the target
	// instead of failing on remote detection; auth is what fails next.
	out, _ = runBT(t, bin, "pr", "list", "-R", "truora/api")
	if strings.Contains(out, "not in a git repository") || strings.Contains(out, "unable to detect") {
		t.Fatalf("-R did not bypass git detection:\n%s", out)
	}
}
