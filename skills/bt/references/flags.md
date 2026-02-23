# bt Command Flag Reference

## bt run view <PIPELINE_ID>

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--log` | bool | false | Full logs for all steps |
| `--log-failed` | bool | false | Logs for failed steps only (last 100 lines) |
| `--full-output` | bool | false | Complete logs (use with --log-failed) |
| `-t, --tests` | bool | false | Show test results and failures |
| `--step <name>` | string | | Specific step logs (case-insensitive partial match) |
| `-w, --watch` | bool | false | Live updates for running pipelines |
| `--web` | bool | false | Open in browser |
| `--url` | bool | false | Print URL instead of opening |
| `-o, --output` | string | table | Output format: table, json, yaml |
| `--no-color` | bool | false | Disable colored output |
| `-R, --repo` | string | | Override repository (HOST/OWNER/REPO) |

## bt run report <PIPELINE_ID>

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--coverage` | bool | false | Coverage info only |
| `--issues` | bool | false | Code quality issues only |
| `--coverage-threshold <N>` | int | 0 | Files below N% coverage |
| `--limit <N>` | int | 10 | Max files/issues shown |
| `--new-code-only` | bool | false | New code analysis only |
| `--severity <S>` | []string | | Filter: BLOCKER, CRITICAL, MAJOR, MINOR, INFO (repeatable) |
| `--show-all-lines` | bool | false | All uncovered lines (not top 5) |
| `--lines-per-file <N>` | int | 5 | Max uncovered lines per file |
| `--new-lines-only` | bool | false | Only NEW uncovered lines from PR |
| `--min-uncovered-lines <N>` | int | 0 | Files with N+ uncovered lines |
| `--max-uncovered-lines <N>` | int | 0 | Files with ≤N uncovered lines (quick wins) |
| `--file <glob>` | string | | Filter files by glob pattern |
| `--no-line-details` | bool | false | Skip line-by-line breakdown |
| `--truncate-lines <N>` | int | 80 | Truncate code at N chars |
| `--web` | bool | false | Open SonarCloud in browser |
| `--url` | bool | false | Print SonarCloud URL |
| `--debug` | bool | false | Debug output |
| `-o, --output` | string | table | Output format: table, json, yaml |

## bt pr report <PR_ID>

Same flags as `bt run report`, plus:

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--context <N>` | int | 0 | Lines of source context around uncovered lines |

PR IDs accept: number (`123`), hash-prefixed (`#123`), URL, or branch name.

## bt run list

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--status <S>` | string | | Filter: SUCCESSFUL, FAILED, ERROR, STOPPED, PENDING, IN_PROGRESS |
| `--branch <B>` | string | | Filter by branch name |
| `--creator <C>` | string | | Filter by creator display name (partial match) |
| `--limit <N>` | int | 10 | Max results |
| `-o, --output` | string | table | Output format: table, json, yaml |

## bt run rerun <PIPELINE_ID>

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--failed` | bool | false | Rerun only failed steps |
