# MILESTONE 1: Authentication MVP - Manual QA Checklist

## Overview
This document provides a systematic manual testing checklist for validating MILESTONE 1: Authentication MVP. These tests complement the automated tests and ensure the authentication system works correctly from an end-user perspective.

**Status**: 🔄 IN PROGRESS  
**Validation Date**: TBD  
**Tested By**: TBD  
**Bitbucket API Version**: v2.0  

---

## Pre-Test Setup

### Environment Preparation
- [ ] **Clean Test Environment**: Remove existing auth config (`rm -rf ~/.config/bt/`)
- [ ] **Build Latest Version**: `make build` to ensure testing latest code
- [ ] **Network Connectivity**: Verify internet connection to api.bitbucket.org
- [ ] **Test Credentials Available**: 
  - [ ] Bitbucket App Password (username + password)
  - [ ] Bitbucket Access Token 
  - [ ] OAuth Client ID/Secret (if testing OAuth)

### Baseline Verification
- [ ] **Command Availability**: `./build/bt --help` shows all commands
- [ ] **Version Check**: `./build/bt version` shows correct version (0.0.1)
- [ ] **Initial State**: `./build/bt auth status` indicates not authenticated

---

## Manual Test Cases

### 🔐 Authentication Method Testing

#### Test Case 1: App Password Authentication
**Objective**: Verify app password authentication works end-to-end

**Steps**:
1. [ ] Run `./build/bt auth login`
2. [ ] Select "App Password" authentication method
3. [ ] Enter valid Bitbucket username
4. [ ] Enter valid app password (input should be hidden)
5. [ ] Verify success message with username display

**Expected Results**:
- [ ] Login completes successfully
- [ ] Success message shows authenticated username
- [ ] No password visible in terminal output
- [ ] Process completes in <10 seconds

**Validation**:
- [ ] `./build/bt auth status` shows authenticated user
- [ ] Username matches entered credentials
- [ ] Auth method shows as "app_password"

---

#### Test Case 2: Access Token Authentication  
**Objective**: Verify access token authentication works correctly

**Steps**:
1. [ ] Run `./build/bt auth logout` (if previously authenticated)
2. [ ] Run `./build/bt auth login --with-token`
3. [ ] Enter valid access token (input should be hidden)
4. [ ] Verify success message

**Expected Results**:
- [ ] Login completes successfully with token
- [ ] Success message confirms authentication
- [ ] Token input is hidden from display

**Validation**:
- [ ] `./build/bt auth status` shows authenticated user
- [ ] Auth method shows as "access_token"
- [ ] Token is not displayed in status output

---

#### Test Case 3: OAuth Authentication (if available)
**Objective**: Verify OAuth browser flow works correctly

**Steps**:
1. [ ] Run `./build/bt auth logout` (if previously authenticated)  
2. [ ] Run `./build/bt auth login`
3. [ ] Select "OAuth" authentication method
4. [ ] Verify browser opens to Bitbucket OAuth page
5. [ ] Complete OAuth authorization in browser
6. [ ] Return to CLI and verify success

**Expected Results**:
- [ ] Browser opens automatically to Bitbucket
- [ ] OAuth flow completes successfully
- [ ] CLI receives and stores OAuth tokens

**Validation**:
- [ ] `./build/bt auth status` shows authenticated user
- [ ] Auth method shows as "oauth"

---

### 🔄 Session Management Testing

#### Test Case 4: Credential Persistence
**Objective**: Verify credentials survive application restarts

**Prerequisites**: Complete any authentication method above

**Steps**:
1. [ ] Verify currently authenticated: `./build/bt auth status`
2. [ ] Note the authenticated username
3. [ ] Simulate application restart (no action needed)
4. [ ] Run `./build/bt auth status` again

**Expected Results**:
- [ ] Authentication status is maintained
- [ ] Same username is displayed
- [ ] No re-authentication required

**File System Validation**:
- [ ] Auth config exists: `ls ~/.config/bt/auth.yml`
- [ ] File contains encrypted/encoded data (not plaintext credentials)

---

#### Test Case 5: Multiple Session Support
**Objective**: Verify multiple concurrent CLI sessions work

**Prerequisites**: Authenticated session

**Steps**:
1. [ ] Open first terminal, run `./build/bt auth status`
2. [ ] Open second terminal, run `./build/bt auth status`  
3. [ ] Verify both show same authentication

**Expected Results**:
- [ ] Both terminals show identical auth status
- [ ] No interference between sessions
- [ ] Credentials shared correctly

---

### 🚪 Logout and Cleanup Testing

#### Test Case 6: Standard Logout
**Objective**: Verify logout clears credentials properly

**Prerequisites**: Authenticated session

**Steps**:
1. [ ] Run `./build/bt auth logout`
2. [ ] Confirm logout when prompted (if applicable)
3. [ ] Verify success message

**Expected Results**:
- [ ] Logout completes successfully
- [ ] Confirmation message displayed
- [ ] Process completes quickly (<2 seconds)

**Validation**:
- [ ] `./build/bt auth status` shows "not authenticated"
- [ ] Auth config file removed: `ls ~/.config/bt/auth.yml` (should not exist)
- [ ] No sensitive data remains in filesystem

---

#### Test Case 7: Force Logout
**Objective**: Verify force logout works without confirmation

**Prerequisites**: Authenticated session

**Steps**:
1. [ ] Run `./build/bt auth logout --force` (if flag exists)
2. [ ] Verify immediate logout without prompts

**Expected Results**:
- [ ] Logout completes without confirmation prompts
- [ ] Same cleanup as standard logout

---

### ⚠️ Error Handling Testing

#### Test Case 8: Invalid Credentials
**Objective**: Verify clear error messages for invalid credentials

**Steps**:
1. [ ] Run `./build/bt auth login`
2. [ ] Enter invalid username/password or token
3. [ ] Observe error message

**Expected Results**:
- [ ] Clear error message about invalid credentials
- [ ] No confusing technical errors
- [ ] Suggests next steps (e.g., "Check credentials")
- [ ] Does not store invalid credentials

**Validation**:
- [ ] `./build/bt auth status` still shows "not authenticated"
- [ ] Error message is user-friendly

---

#### Test Case 9: Network Connectivity Issues
**Objective**: Verify graceful handling of network issues

**Steps**:
1. [ ] Disconnect from internet or block api.bitbucket.org
2. [ ] Run `./build/bt auth login` with valid credentials
3. [ ] Observe error handling

**Expected Results**:
- [ ] Clear error message about network connectivity
- [ ] Suggests checking internet connection
- [ ] Does not hang indefinitely
- [ ] Fails gracefully within reasonable time (<30 seconds)

---

#### Test Case 10: Revoked/Expired Credentials
**Objective**: Verify handling of revoked credentials

**Prerequisites**: Valid credentials that can be revoked

**Steps**:
1. [ ] Authenticate successfully
2. [ ] Revoke credentials in Bitbucket web interface
3. [ ] Run `./build/bt auth status`
4. [ ] Try to use a command that requires authentication

**Expected Results**:
- [ ] Clear error about invalid/expired credentials
- [ ] Suggests re-authentication
- [ ] Offers to clear invalid credentials

---

### 🌍 Environment Variable Testing

#### Test Case 11: Environment Variable Authentication
**Objective**: Verify environment variable authentication works

**Steps**:
1. [ ] Ensure no stored authentication: `./build/bt auth logout`
2. [ ] Set environment variable: `export BITBUCKET_TOKEN=<valid_token>`
3. [ ] Run `./build/bt auth status`

**Expected Results**:
- [ ] Authentication detected from environment variable
- [ ] Status shows authenticated user
- [ ] Indicates auth source as environment variable

**Cleanup**:
- [ ] `unset BITBUCKET_TOKEN`

---

#### Test Case 12: Environment Variable Precedence
**Objective**: Verify environment variables override stored credentials

**Prerequisites**: Stored authentication

**Steps**:
1. [ ] Note current authenticated user
2. [ ] Set `BITBUCKET_TOKEN` to different valid token
3. [ ] Run `./build/bt auth status`
4. [ ] Verify different user is shown

**Expected Results**:
- [ ] Environment variable takes precedence
- [ ] Different user shown in status
- [ ] Clear indication of auth source

---

## Performance Testing

### Test Case 13: Authentication Performance
**Objective**: Verify authentication performance meets targets

**Steps**:
1. [ ] Time authentication process: `time ./build/bt auth login`
2. [ ] Time status check: `time ./build/bt auth status`
3. [ ] Measure multiple consecutive status checks

**Expected Results**:
- [ ] Initial authentication: <10 seconds
- [ ] Status check: <2 seconds  
- [ ] Consecutive status checks: <1 second each

---

## Cross-Platform Testing (if applicable)

### Test Case 14: Cross-Platform Compatibility
**Objective**: Verify auth works across different platforms

**Platforms to Test**:
- [ ] Linux (primary development platform)
- [ ] macOS (if available)
- [ ] Windows (if available)

**Validation for Each Platform**:
- [ ] Authentication works correctly
- [ ] Credential storage works
- [ ] File paths are correct
- [ ] No platform-specific errors

---

## Security Validation

### Test Case 15: Credential Security
**Objective**: Verify credentials are stored securely

**Steps**:
1. [ ] Authenticate with any method
2. [ ] Examine auth config file: `cat ~/.config/bt/auth.yml`
3. [ ] Check file permissions: `ls -la ~/.config/bt/auth.yml`

**Expected Results**:
- [ ] No plaintext passwords/tokens in config file
- [ ] Data appears encrypted or encoded
- [ ] File permissions restrict access (600 or similar)
- [ ] Directory permissions appropriate

**Security Checklist**:
- [ ] Passwords never displayed in terminal
- [ ] Tokens not logged to files
- [ ] Auth file not world-readable
- [ ] No credentials in shell history

---

## Validation Summary

### QA Checklist from TASKS.md

- [ ] `bt auth login` completes successfully with each auth method
- [ ] `bt auth status` shows correct user information  
- [ ] `bt auth logout` clears credentials properly
- [ ] Credentials persist across CLI sessions
- [ ] Error messages are clear and actionable

### Additional Validation Items

- [ ] Performance meets targets (<10s auth, <2s status)
- [ ] Security best practices followed
- [ ] Cross-platform compatibility (where applicable)
- [ ] Environment variable support works correctly
- [ ] All error scenarios handled gracefully

---

## Sign-off

### Human QA Validation
- **Date**: ___________
- **Tester**: ___________
- **Platform**: ___________
- **Status**: [ ] PASSED / [ ] FAILED / [ ] PARTIAL

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
- [ ] **MILESTONE 1 READY**: Authentication system is production-ready
- [ ] **MILESTONE 1 BLOCKED**: Critical issues prevent advancement
- [ ] **MILESTONE 1 PARTIAL**: Some functionality working, issues documented

---

**Next Steps**: Upon successful completion of this checklist, proceed to MILESTONE 2: Pipeline Debug MVP validation.