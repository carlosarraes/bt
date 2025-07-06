# MILESTONE 2: Pipeline Debug MVP - Manual QA Checklist

## Overview
This document provides a systematic manual testing checklist for validating MILESTONE 2: Pipeline Debug MVP. These tests complement the automated tests and ensure the pipeline debugging functionality provides the promised 5x faster debugging experience compared to Bitbucket web UI.

**Status**: 🔄 IN PROGRESS  
**Validation Date**: TBD  
**Tested By**: TBD  
**Bitbucket API Version**: v2.0  
**Key Differentiator**: 5x faster pipeline debugging vs web UI

---

## Pre-Test Setup

### Environment Preparation
- [ ] **Authentication Complete**: Ensure `bt auth status` shows authenticated user
- [ ] **Test Repository Access**: Verify access to repository with recent pipeline runs
- [ ] **Build Latest Version**: `make build` to ensure testing latest code
- [ ] **Network Connectivity**: Verify stable connection to api.bitbucket.org
- [ ] **Test Data Available**: Repository with diverse pipeline states (successful, failed, running)

### Baseline Verification
- [ ] **Command Availability**: All run commands available in `./build/bt run --help`
- [ ] **Version Check**: `./build/bt version` shows correct version
- [ ] **Authentication Status**: `./build/bt auth status` shows authenticated user
- [ ] **Repository Context**: Test in directory with valid Bitbucket git repository

### Timing Setup for 5x Speed Comparison
- [ ] **Web UI Baseline**: Time common debugging tasks in Bitbucket web interface
- [ ] **Stopwatch Ready**: Prepare to time CLI operations for comparison
- [ ] **Test Scenarios Planned**: Identify specific debugging workflows to compare

---

## Core Pipeline Debugging Tests

### 🔍 Pipeline Discovery and Filtering (bt run list)

#### Test Case 1: Basic Pipeline Listing
**Objective**: Verify pipeline list displays recent runs clearly

**Steps**:
1. [ ] Run `./build/bt run list`
2. [ ] Verify output shows recent pipeline runs
3. [ ] Check that table format is readable and informative
4. [ ] Note pipeline IDs for subsequent tests

**Expected Results**:
- [ ] List displays within 2 seconds
- [ ] Shows build numbers, status, branch, and timing
- [ ] Color coding for different statuses (if terminal supports)
- [ ] Clear column headers and formatting

**Performance Target**: < 500ms response time

---

#### Test Case 2: Status Filtering
**Objective**: Verify filtering by pipeline status works correctly

**Steps**:
1. [ ] Run `./build/bt run list --status failed`
2. [ ] Run `./build/bt run list --status successful`
3. [ ] Run `./build/bt run list --status in_progress`
4. [ ] Verify only matching statuses are shown

**Expected Results**:
- [ ] Each filter shows only matching pipeline states
- [ ] Clear indication when no pipelines match filter
- [ ] Consistent performance across different filters

---

#### Test Case 3: Branch Filtering
**Objective**: Verify branch-specific pipeline filtering

**Steps**:
1. [ ] Run `./build/bt run list --branch main`
2. [ ] Run `./build/bt run list --branch develop` (if exists)
3. [ ] Verify filtering works correctly

**Expected Results**:
- [ ] Shows only pipelines from specified branch
- [ ] Branch names displayed clearly in output
- [ ] Performance remains fast with filtering

---

#### Test Case 4: Combined Filtering and Limits
**Objective**: Test advanced filtering combinations

**Steps**:
1. [ ] Run `./build/bt run list --status failed --branch main --limit 5`
2. [ ] Run `./build/bt run list --limit 20`
3. [ ] Verify combinations work as expected

**Expected Results**:
- [ ] Filters combine correctly (AND operation)
- [ ] Limit parameter respected
- [ ] Clear output even with multiple filters

---

### 📋 Detailed Pipeline Inspection (bt run view)

#### Test Case 5: Pipeline Overview
**Objective**: Verify comprehensive pipeline detail display

**Steps**:
1. [ ] Get pipeline ID from list: `./build/bt run list --limit 1`
2. [ ] Run `./build/bt run view <pipeline-id>`
3. [ ] Verify detailed information display

**Expected Results**:
- [ ] Complete pipeline metadata (branch, commit, trigger, duration)
- [ ] Step-by-step breakdown with status indicators
- [ ] Clear visual distinction between step states
- [ ] Human-readable timestamps and durations
- [ ] Repository and workspace information

**Performance Target**: < 500ms for pipeline details

---

#### Test Case 6: Pipeline Build Number Support
**Objective**: Verify build number ID resolution works

**Steps**:
1. [ ] Get build number from list output (e.g., #123)
2. [ ] Run `./build/bt run view 123` (without # prefix)
3. [ ] Run `./build/bt run view #123` (with # prefix)
4. [ ] Verify both formats work

**Expected Results**:
- [ ] Both build number formats resolve correctly
- [ ] Same detailed information displayed
- [ ] Clear indication of build number in output

---

#### Test Case 7: Real-time Pipeline Monitoring
**Objective**: Test live pipeline monitoring functionality

**Prerequisites**: Running or recent pipeline available

**Steps**:
1. [ ] Find running pipeline: `./build/bt run list --status in_progress`
2. [ ] Run `./build/bt run view <id> --watch`
3. [ ] Observe live updates every 5 seconds
4. [ ] Press Ctrl+C to exit gracefully

**Expected Results**:
- [ ] Live updates every 5 seconds
- [ ] Clear indication of status changes
- [ ] Graceful exit with Ctrl+C
- [ ] No hanging processes after exit

---

### 📜 Log Analysis and Error Detection (bt run view --log*)

#### Test Case 8: Complete Log Viewing
**Objective**: Verify comprehensive log streaming

**Steps**:
1. [ ] Run `./build/bt run view <pipeline-id> --log`
2. [ ] Verify logs from all steps are displayed
3. [ ] Check performance with large log files

**Expected Results**:
- [ ] All pipeline step logs displayed
- [ ] Clear step separation in output
- [ ] Reasonable performance even with large logs (< 5 seconds for 10MB)
- [ ] Proper character encoding and formatting

---

#### Test Case 9: Failed Step Log Analysis (KILLER FEATURE)
**Objective**: Test intelligent error detection and highlighting

**Steps**:
1. [ ] Find failed pipeline: `./build/bt run list --status failed --limit 1`
2. [ ] Run `./build/bt run view <failed-pipeline-id> --log-failed`
3. [ ] Verify error highlighting and analysis

**Expected Results**:
- [ ] Shows logs only from failed steps
- [ ] Error lines highlighted (red/bold in terminal)
- [ ] Last 100 lines by default (smart truncation)
- [ ] Clear indication of which steps failed
- [ ] Actionable error information

**5x Speed Test**: 
- [ ] Time: Finding error in CLI vs web UI
- [ ] CLI should be significantly faster (target: 5x improvement)

---

#### Test Case 10: Full Output for Deep Analysis
**Objective**: Test complete log retrieval for complex debugging

**Steps**:
1. [ ] Run `./build/bt run view <failed-pipeline-id> --log-failed --full-output`
2. [ ] Verify complete failed step logs displayed

**Expected Results**:
- [ ] Complete logs from failed steps (not truncated)
- [ ] Suitable for deep error analysis
- [ ] Performance acceptable for large logs
- [ ] Clear step boundaries in output

---

#### Test Case 11: Test Results Analysis
**Objective**: Test specialized test result viewing

**Steps**:
1. [ ] Find pipeline with tests: Look for pipelines with test steps
2. [ ] Run `./build/bt run view <pipeline-id> --tests`
3. [ ] Verify test-specific output

**Expected Results**:
- [ ] Test results clearly displayed
- [ ] Failed test details highlighted
- [ ] Test summary information
- [ ] Clear pass/fail indicators

---

#### Test Case 12: Step-Specific Log Filtering
**Objective**: Test targeted step log viewing

**Steps**:
1. [ ] Run `./build/bt run view <pipeline-id>` to see step names
2. [ ] Run `./build/bt run view <pipeline-id> --step <step-name> --log`
3. [ ] Verify only specified step logs shown

**Expected Results**:
- [ ] Only logs from specified step
- [ ] Step name matching works (fuzzy matching)
- [ ] Clear indication of selected step
- [ ] Fast response for targeted analysis

---

### ⏱️ Real-time Monitoring (bt run watch)

#### Test Case 13: Dedicated Watch Command
**Objective**: Test standalone pipeline monitoring

**Prerequisites**: Running pipeline or recent pipeline

**Steps**:
1. [ ] Run `./build/bt run watch <pipeline-id>`
2. [ ] Observe real-time status updates
3. [ ] Wait for state changes or completion
4. [ ] Test Ctrl+C interruption

**Expected Results**:
- [ ] Live status updates every 5 seconds
- [ ] Progress indicators for steps (✅ ❌ 🔄 ⏳)
- [ ] Step completion notifications
- [ ] Automatic exit on pipeline completion
- [ ] Clean interrupt handling

---

#### Test Case 14: Watch with JSON Output
**Objective**: Test automation-friendly watch output

**Steps**:
1. [ ] Run `./build/bt run watch <pipeline-id> --output json`
2. [ ] Verify JSON formatted status updates

**Expected Results**:
- [ ] Valid JSON output for each update
- [ ] Structured data suitable for automation
- [ ] Consistent JSON schema across updates

---

### ❌ Pipeline Management (bt run cancel)

#### Test Case 15: Pipeline Cancellation
**Objective**: Test pipeline cancellation functionality

**Prerequisites**: Running pipeline (optional - can test with completed pipeline for error handling)

**Steps**:
1. [ ] Find running pipeline: `./build/bt run list --status in_progress`
2. [ ] Run `./build/bt run cancel <pipeline-id>`
3. [ ] Respond to confirmation prompt
4. [ ] Verify cancellation result

**Expected Results**:
- [ ] Clear confirmation prompt before cancellation
- [ ] Success message after cancellation
- [ ] Proper error message if pipeline can't be cancelled
- [ ] Status verification in subsequent commands

---

#### Test Case 16: Force Cancellation
**Objective**: Test automated cancellation for scripts

**Steps**:
1. [ ] Run `./build/bt run cancel <pipeline-id> --force`
2. [ ] Verify no confirmation prompt

**Expected Results**:
- [ ] Immediate cancellation without prompts
- [ ] Suitable for automation scenarios
- [ ] Clear success/failure feedback

---

### 📊 Multi-format Output Testing

#### Test Case 17: JSON Output for Automation
**Objective**: Verify all commands support JSON output

**Steps**:
1. [ ] Run `./build/bt run list --output json`
2. [ ] Run `./build/bt run view <id> --output json`
3. [ ] Verify JSON structure and validity

**Expected Results**:
- [ ] Valid JSON output for all commands
- [ ] Consistent schema across commands
- [ ] All essential data included in JSON
- [ ] Suitable for AI/LLM processing

---

#### Test Case 18: YAML Output Support
**Objective**: Test alternative structured output format

**Steps**:
1. [ ] Run `./build/bt run list --output yaml`
2. [ ] Run `./build/bt run view <id> --output yaml`
3. [ ] Verify YAML formatting

**Expected Results**:
- [ ] Valid YAML output
- [ ] Human-readable structure
- [ ] Consistent with JSON data content

---

## Performance Validation (5x Speed Improvement)

### Test Case 19: Speed Comparison vs Web UI
**Objective**: Validate the 5x faster debugging claim

**Comparison Scenarios**:

#### Scenario A: Find Latest Failed Pipeline
- [ ] **Web UI**: Navigate to repository → Pipelines → Filter by failed
- [ ] **CLI**: `./build/bt run list --status failed --limit 1`
- [ ] **Timing**: CLI should be significantly faster

#### Scenario B: Analyze Failed Pipeline Logs
- [ ] **Web UI**: Open failed pipeline → Navigate to failed step → View logs
- [ ] **CLI**: `./build/bt run view <id> --log-failed`
- [ ] **Timing**: CLI should be much faster for error identification

#### Scenario C: Monitor Running Pipeline
- [ ] **Web UI**: Manual refresh of pipeline page
- [ ] **CLI**: `./build/bt run watch <id>`
- [ ] **Experience**: CLI provides automatic updates vs manual refresh

#### Scenario D: Find Specific Error in Logs
- [ ] **Web UI**: Scroll through logs in browser
- [ ] **CLI**: Use `--log-failed` with error highlighting
- [ ] **Efficiency**: CLI error highlighting should be much faster

**Performance Targets**:
- [ ] List operations: < 500ms
- [ ] Pipeline details: < 500ms
- [ ] Log retrieval: < 2s for 10MB logs
- [ ] Overall workflow: 5x faster than web UI

---

## Error Handling and Edge Cases

### Test Case 20: Invalid Pipeline IDs
**Objective**: Verify graceful error handling

**Steps**:
1. [ ] Run `./build/bt run view nonexistent-id`
2. [ ] Run `./build/bt run view 99999`
3. [ ] Verify clear error messages

**Expected Results**:
- [ ] Clear "pipeline not found" messages
- [ ] Suggestions for valid pipeline IDs
- [ ] No technical stack traces
- [ ] Appropriate exit codes

---

### Test Case 21: Network Issues
**Objective**: Test behavior with connectivity problems

**Steps**:
1. [ ] Simulate network issue (disconnect/slow connection)
2. [ ] Run various pipeline commands
3. [ ] Observe error handling

**Expected Results**:
- [ ] Clear network error messages
- [ ] Reasonable timeout behavior (< 30s)
- [ ] Suggestions for troubleshooting
- [ ] Graceful failure without hanging

---

### Test Case 22: Large Repository Performance
**Objective**: Test performance with repositories that have many pipelines

**Steps**:
1. [ ] Test commands on repository with 100+ pipelines
2. [ ] Measure response times
3. [ ] Verify pagination works correctly

**Expected Results**:
- [ ] Performance remains good with large datasets
- [ ] Pagination limits prevent overwhelming output
- [ ] Memory usage remains reasonable

---

## User Experience Validation

### Test Case 23: Command Discoverability
**Objective**: Verify commands are intuitive and discoverable

**Steps**:
1. [ ] Run `./build/bt run --help`
2. [ ] Check help text for each subcommand
3. [ ] Verify examples are clear

**Expected Results**:
- [ ] Clear command descriptions
- [ ] Helpful examples for common use cases
- [ ] Consistent help format across commands
- [ ] Easy to understand for new users

---

### Test Case 24: Error Message Quality
**Objective**: Ensure error messages are helpful and actionable

**Steps**:
1. [ ] Trigger various error conditions
2. [ ] Evaluate error message clarity
3. [ ] Check for actionable suggestions

**Expected Results**:
- [ ] Clear, non-technical error descriptions
- [ ] Suggestions for resolution
- [ ] Consistent error message format
- [ ] Appropriate error codes

---

## Automation and AI Integration

### Test Case 25: JSON Schema Consistency
**Objective**: Verify JSON output is suitable for automation

**Steps**:
1. [ ] Collect JSON output from all commands
2. [ ] Verify schema consistency
3. [ ] Test with real automation scripts

**Expected Results**:
- [ ] Consistent field names across commands
- [ ] Stable schema that won't break automation
- [ ] Complete data for decision making
- [ ] Suitable for LLM/AI processing

---

### Test Case 26: Error Data for AI Analysis
**Objective**: Test structured error data for AI consumption

**Steps**:
1. [ ] Find failed pipelines with errors
2. [ ] Extract error data using JSON output
3. [ ] Verify structure is suitable for AI analysis

**Expected Results**:
- [ ] Error context included (branch, commit, step)
- [ ] Structured error categorization
- [ ] Complete failure information
- [ ] Suitable for automated error analysis

---

## Validation Summary

### QA Checklist from TASKS.md

#### Human QA Validation
- [ ] `bt run list` shows pipeline runs with proper filtering
- [ ] `bt run view <id>` displays comprehensive pipeline details
- [ ] `bt run view <id> --log-failed` extracts errors accurately
- [ ] `bt run watch <id>` provides real-time updates
- [ ] Pipeline debugging workflow is faster than Bitbucket web UI
- [ ] Error highlighting is clear and actionable

#### AI QA Validation
- [ ] JSON output includes all necessary metadata for automation
- [ ] Error extraction produces structured data suitable for LLM analysis
- [ ] API responses are properly typed and documented
- [ ] Performance meets <500ms target for common operations

### Additional Validation Items

#### Performance & Efficiency
- [ ] 5x faster debugging workflow validated vs web UI
- [ ] All operations meet <500ms performance targets
- [ ] Memory usage remains reasonable during operations
- [ ] Handles large repositories and log files efficiently

#### User Experience
- [ ] Commands are intuitive and discoverable
- [ ] Error messages are clear and actionable
- [ ] Output formatting is readable and informative
- [ ] Workflow is efficient for daily developer use

#### Technical Integration
- [ ] JSON/YAML output suitable for automation
- [ ] Error data structured for AI/LLM analysis
- [ ] Cross-platform compatibility verified
- [ ] Integration with existing auth and config systems

---

## Performance Benchmark Results

### Speed Comparison: CLI vs Web UI

| Task | Web UI Time | CLI Time | Improvement |
|------|-------------|----------|-------------|
| Find latest failed pipeline | ___s | ___s | ___x |
| View pipeline details | ___s | ___s | ___x |
| Analyze failed step logs | ___s | ___s | ___x |
| Monitor running pipeline | Manual refresh | Auto updates | Qualitative |
| **Overall workflow** | ___s | ___s | **___x** |

### API Response Times

| Operation | Target | Actual | Status |
|-----------|--------|--------|--------|
| `bt run list` | <500ms | ___ms | [ ] PASS / [ ] FAIL |
| `bt run view` | <500ms | ___ms | [ ] PASS / [ ] FAIL |
| Log retrieval | <2s | ___s | [ ] PASS / [ ] FAIL |

---

## Sign-off

### Human QA Validation
- **Date**: ___________
- **Tester**: ___________
- **Platform**: ___________
- **Test Repository**: ___________
- **Status**: [ ] PASSED / [ ] FAILED / [ ] PARTIAL

### Performance Validation
- **5x Speed Improvement**: [ ] CONFIRMED / [ ] NOT CONFIRMED
- **All Performance Targets**: [ ] MET / [ ] NOT MET
- **Memory Usage**: [ ] ACCEPTABLE / [ ] EXCESSIVE

### Issues Found
(List any issues discovered during testing)

1. _________________________________
2. _________________________________
3. _________________________________

### Recommendations
(List any recommendations for improvement)

1. _________________________________
2. _________________________________
3. _________________________________

### Final Assessment
- [ ] **MILESTONE 2 READY**: Pipeline debugging system provides proven 5x speed improvement
- [ ] **MILESTONE 2 BLOCKED**: Critical issues prevent validation
- [ ] **MILESTONE 2 PARTIAL**: Core functionality working, issues documented

---

**Next Steps**: Upon successful completion of this checklist, update TASKS.md with MILESTONE 2 validation results and proceed to MILESTONE 3: User Experience Polish validation.