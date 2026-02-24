---
name: bt
description: Debug Bitbucket pipeline failures, analyze test coverage, and manage CI/CD using the `bt` CLI. Use this skill whenever the user mentions pipelines, builds, CI/CD, test failures, test coverage, uncovered lines, SonarCloud, or quality gates. Trigger on natural phrases like "check pipeline", "check my PR", "why did the build fail", "build is broken", "CI failed", "tests are red", "what tests failed", "check coverage", "what lines need tests", "improve coverage", "missing tests", "is the pipeline done", "rerun the pipeline", "retry the build", "run report", "sonar", or "quality gate". Also trigger when referencing `bt run`, `bt pr report`, pipeline IDs, or PR numbers in a debugging context. This skill is essential for any Bitbucket pipeline or PR quality workflow — when in doubt about whether to use it, use it.
---

# bt Pipeline Debugging

`bt` is a Bitbucket CLI (like `gh` for GitHub). This skill covers the debugging workflow: finding failures, reading logs, analyzing test results, and checking coverage gaps.

## Prerequisites

The user must be inside a git repo that has a Bitbucket remote. Authentication must be configured (`bt auth status` to check). For SonarCloud reports, `SONARCLOUD_TOKEN` must be set.

## Debugging Workflow

Follow this sequence. Skip steps that aren't needed based on context.

### 1. Find the Failed Pipeline

```bash
# Recent failures
bt run list --status failed

# Failures on a specific branch
bt run list --status failed --branch feature-xyz

# JSON for structured analysis
bt run list --status failed --output json
```

If the user gives a PR number instead of a pipeline ID, use `bt pr report` directly (step 4).

### 2. Diagnose the Failure

Start narrow, expand if needed:

```bash
# Quick error summary (last 100 lines of failed steps) — start here
bt run view <ID> --log-failed

# Full failure logs if truncated output isn't enough
bt run view <ID> --log-failed --full-output

# Test failures specifically (assertion errors, counts)
bt run view <ID> --tests

# Logs for a specific step by name
bt run view <ID> --step "Run Tests"
```

For JSON output (useful for parsing structured error data):
```bash
bt run view <ID> --log-failed --output json
```

The JSON output includes a `steps` array where each failed step has `name`, `state`, and `logs` fields.

### 3. Pipeline Overview

When you need the full picture (not just errors):

```bash
# Summary: status, branch, commit, duration, all steps
bt run view <ID>

# Watch a running pipeline in real-time
bt run watch <ID>
```

### 4. Coverage & Quality Analysis

Two entry points — by pipeline ID or by PR number:

```bash
# By pipeline
bt run report <PIPELINE_ID> --coverage

# By pull request (more common for devs)
bt pr report <PR_ID> --coverage
```

**Finding uncovered lines (the core use case):**

```bash
# Files below 80% coverage
bt pr report <PR_ID> --coverage --coverage-threshold 80

# Only NEW uncovered lines from this PR
bt pr report <PR_ID> --coverage --new-lines-only

# Quick wins: files needing few lines to reach threshold
bt pr report <PR_ID> --coverage --max-uncovered-lines 5

# Show surrounding code context (PR report only)
bt pr report <PR_ID> --coverage --context 3

# All uncovered lines (not just top 5 per file)
bt pr report <PR_ID> --coverage --show-all-lines

# Filter to specific files
bt pr report <PR_ID> --coverage --file "pkg/api/*.go"

# JSON for programmatic analysis
bt pr report <PR_ID> --coverage --output json
```

**Code quality issues:**

```bash
# Issues only
bt pr report <PR_ID> --issues

# Critical/blocker issues only
bt pr report <PR_ID> --issues --severity CRITICAL --severity BLOCKER

# Combined: coverage + issues, new code only
bt pr report <PR_ID> --new-code-only

# Open SonarCloud dashboard
bt pr report <PR_ID> --web
```

### 5. Act on Results

After diagnosing, common next steps:

```bash
# Rerun the pipeline (or just failed steps)
bt run rerun <ID>
bt run rerun <ID> --failed

# Cancel a stuck pipeline
bt run cancel <ID>
```

## Flag Reference

For the full list of flags on each command, read `references/flags.md` in this skill.

## Key Patterns

- **Always use `--output json`** when you need to parse or analyze results programmatically
- **`bt run view --log-failed`** is the fastest path to error diagnosis — start there
- **`bt pr report --coverage --new-lines-only`** is the most useful coverage command for PR reviews — it shows exactly what the developer needs to cover
- **`--context N`** on `bt pr report` shows surrounding source code, making it easy to understand what's uncovered without opening an editor
- **`--coverage-threshold`** combined with `--max-uncovered-lines` finds "quick win" files that are close to the target
