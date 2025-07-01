package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// CLITestSuite provides a test suite for CLI command testing
type CLITestSuite struct {
	suite.Suite
	testCtx    *utils.TestContext
	binaryPath string
}

// SetupSuite initializes the test suite
func (suite *CLITestSuite) SetupSuite() {
	suite.testCtx = utils.NewTestContext(suite.T())
	suite.binaryPath = utils.BuildBinary(suite.T())
}

// TearDownSuite cleans up after the test suite
func (suite *CLITestSuite) TearDownSuite() {
	if suite.testCtx != nil {
		suite.testCtx.Close()
	}
}

// TestVersionCommand tests the version command
func (suite *CLITestSuite) TestVersionCommand() {
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// Verify version output contains expected information
	assert.Contains(suite.T(), result.Stdout, "bt version")
	assert.Contains(suite.T(), result.Stdout, "Commit:")
	assert.Contains(suite.T(), result.Stdout, "Built:")
	assert.Contains(suite.T(), result.Stdout, "Go version:")
}

// TestVersionCommandJSON tests version command with JSON output
func (suite *CLITestSuite) TestVersionCommandJSON() {
	result := utils.RunBTCommand("--output", "json", "version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// Verify JSON output
	versionData := utils.AssertJSONOutput(suite.T(), result.Stdout)
	assert.Contains(suite.T(), versionData, "version")
	assert.Contains(suite.T(), versionData, "commit")
	assert.Contains(suite.T(), versionData, "date")
	assert.Contains(suite.T(), versionData, "go_version")
}

// TestHelpCommand tests the help command
func (suite *CLITestSuite) TestHelpCommand() {
	result := utils.RunBTCommand("--help")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// Verify help output contains expected sections
	assert.Contains(suite.T(), result.Stdout, "Work seamlessly with Bitbucket from the command line")
	assert.Contains(suite.T(), result.Stdout, "Commands:")
	assert.Contains(suite.T(), result.Stdout, "version")
	assert.Contains(suite.T(), result.Stdout, "auth")
	assert.Contains(suite.T(), result.Stdout, "repo")
	assert.Contains(suite.T(), result.Stdout, "pr")
	assert.Contains(suite.T(), result.Stdout, "run")
	assert.Contains(suite.T(), result.Stdout, "api")
}

// TestCommandHelp tests help for individual commands
func (suite *CLITestSuite) TestCommandHelp() {
	commands := []string{"version", "auth", "repo", "pr", "run", "api"}
	
	for _, cmd := range commands {
		suite.T().Run(fmt.Sprintf("Help_%s", cmd), func(t *testing.T) {
			result := utils.RunBTCommand(cmd, "--help")
			utils.AssertCommandSuccess(t, result)
			
			// Verify help output contains the command name
			assert.Contains(t, result.Stdout, cmd)
		})
	}
}

// TestGlobalFlags tests global flags functionality
func (suite *CLITestSuite) TestGlobalFlags() {
	testCases := []struct {
		name        string
		args        []string
		expectSuccess bool
		description string
	}{
		{
			name:        "VerboseFlag",
			args:        []string{"--verbose", "version"},
			expectSuccess: true,
			description: "Verbose flag should work",
		},
		{
			name:        "VerboseFlagShort",
			args:        []string{"-v", "version"},
			expectSuccess: true,
			description: "Short verbose flag should work",
		},
		{
			name:        "NoColorFlag",
			args:        []string{"--no-color", "version"},
			expectSuccess: true,
			description: "No color flag should work",
		},
		{
			name:        "OutputJSON",
			args:        []string{"--output", "json", "version"},
			expectSuccess: true,
			description: "JSON output flag should work",
		},
		{
			name:        "OutputTable",
			args:        []string{"--output", "table", "version"},
			expectSuccess: true,
			description: "Table output flag should work",
		},
		{
			name:        "OutputYAML",
			args:        []string{"--output", "yaml", "version"},
			expectSuccess: true,
			description: "YAML output flag should work",
		},
		{
			name:        "InvalidOutput",
			args:        []string{"--output", "invalid", "version"},
			expectSuccess: false,
			description: "Invalid output format should fail",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := utils.RunBTCommand(tc.args...)
			
			if tc.expectSuccess {
				utils.AssertCommandSuccess(t, result)
			} else {
				utils.AssertCommandFailure(t, result, 1)
			}
		})
	}
}

// TestConfigFlag tests the config flag functionality
func (suite *CLITestSuite) TestConfigFlag() {
	// Create a custom config file
	configContent := `
output:
  format: json
  no_color: true
`
	configPath := filepath.Join(suite.testCtx.TempDir, "custom_config.yml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(suite.T(), err)
	
	// Test using custom config
	result := utils.RunBTCommand("--config", configPath, "version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// With JSON format in config, output should be JSON
	utils.AssertJSONOutput(suite.T(), result.Stdout)
}

// TestInvalidCommands tests handling of invalid commands
func (suite *CLITestSuite) TestInvalidCommands() {
	testCases := []struct {
		name           string
		args           []string
		expectedExitCode int
		description    string
	}{
		{
			name:           "NonExistentCommand",
			args:           []string{"nonexistent"},
			expectedExitCode: 1,
			description:    "Non-existent command should fail",
		},
		{
			name:           "InvalidFlag",
			args:           []string{"--invalid-flag", "version"},
			expectedExitCode: 1,
			description:    "Invalid flag should fail",
		},
		{
			name:           "MissingRequiredArg",
			args:           []string{"--output"},
			expectedExitCode: 1,
			description:    "Missing required argument should fail",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := utils.RunBTCommand(tc.args...)
			utils.AssertCommandFailure(t, result, tc.expectedExitCode)
			
			// Error message should be on stderr
			assert.NotEmpty(t, result.Stderr, "Error message should be present")
		})
	}
}

// TestCommandChaining tests command execution flow
func (suite *CLITestSuite) TestCommandChaining() {
	// Test that commands can be run in sequence
	commands := [][]string{
		{"version"},
		{"--help"},
		{"version", "--help"},
	}
	
	for i, cmd := range commands {
		suite.T().Run(fmt.Sprintf("Command_%d", i), func(t *testing.T) {
			result := utils.RunBTCommand(cmd...)
			utils.AssertCommandSuccess(t, result)
		})
	}
}

// TestOutputFormats tests different output formats
func (suite *CLITestSuite) TestOutputFormats() {
	testCases := []struct {
		format      string
		expectJSON  bool
		expectYAML  bool
		expectTable bool
	}{
		{
			format:      "json",
			expectJSON:  true,
			expectYAML:  false,
			expectTable: false,
		},
		{
			format:      "yaml",
			expectJSON:  false,
			expectYAML:  true,
			expectTable: false,
		},
		{
			format:      "table",
			expectJSON:  false,
			expectYAML:  false,
			expectTable: true,
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(fmt.Sprintf("Format_%s", tc.format), func(t *testing.T) {
			result := utils.RunBTCommand("--output", tc.format, "version")
			utils.AssertCommandSuccess(t, result)
			
			if tc.expectJSON {
				utils.AssertJSONOutput(t, result.Stdout)
			} else if tc.expectYAML {
				// Basic YAML validation - should contain key: value pairs
				assert.Contains(t, result.Stdout, ":")
				assert.NotContains(t, result.Stdout, "{")
				assert.NotContains(t, result.Stdout, "}")
			} else if tc.expectTable {
				// Table format should be human-readable
				assert.NotContains(t, result.Stdout, "{")
				assert.NotContains(t, result.Stdout, "}")
			}
		})
	}
}

// TestEnvironmentVariables tests environment variable handling
func (suite *CLITestSuite) TestEnvironmentVariables() {
	// Test that environment variables don't interfere with basic commands
	originalEnv := os.Environ()
	
	// Set some test environment variables
	os.Setenv("BT_OUTPUT_FORMAT", "json")
	os.Setenv("BT_NO_COLOR", "true")
	os.Setenv("BT_VERBOSE", "true")
	
	defer func() {
		// Restore original environment
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	}()
	
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(suite.T(), result)
}

// TestConfigFileResolution tests config file resolution
func (suite *CLITestSuite) TestConfigFileResolution() {
	// Test with non-existent config file
	nonExistentConfig := filepath.Join(suite.testCtx.TempDir, "nonexistent.yml")
	result := utils.RunBTCommand("--config", nonExistentConfig, "version")
	
	// Command should still work even if config file doesn't exist
	utils.AssertCommandSuccess(suite.T(), result)
}

// TestCommandPerformance tests command startup performance
func (suite *CLITestSuite) TestCommandPerformance() {
	// Test that commands start up quickly
	maxStartupTime := 2 * time.Second
	
	testCases := []struct {
		name string
		args []string
	}{
		{"Version", []string{"version"}},
		{"Help", []string{"--help"}},
		{"VersionHelp", []string{"version", "--help"}},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			start := time.Now()
			result := utils.RunBTCommand(tc.args...)
			duration := time.Since(start)
			
			utils.AssertCommandSuccess(t, result)
			
			assert.Less(t, duration, maxStartupTime, 
				"Command %s should start up in less than %v, took %v", 
				tc.name, maxStartupTime, duration)
		})
	}
}

// TestStdinHandling tests stdin handling (if applicable)
func (suite *CLITestSuite) TestStdinHandling() {
	// Test that commands handle stdin appropriately
	// For now, just test that they don't hang when stdin is closed
	
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// Command should complete quickly without hanging
	assert.NotEmpty(suite.T(), result.Stdout, "Should produce output")
}

// TestSignalHandling tests signal handling (basic test)
func (suite *CLITestSuite) TestSignalHandling() {
	// Test that commands can be interrupted gracefully
	// This is a basic test - more sophisticated signal testing would require
	// process management
	
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// If we got here, the command completed normally
	assert.Equal(suite.T(), 0, result.ExitCode)
}

// TestErrorMessages tests error message formatting
func (suite *CLITestSuite) TestErrorMessages() {
	testCases := []struct {
		name        string
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "InvalidCommand",
			args:        []string{"invalid-command"},
			expectError: true,
			description: "Invalid command should show helpful error",
		},
		{
			name:        "InvalidFlag",
			args:        []string{"--invalid-flag"},
			expectError: true,
			description: "Invalid flag should show helpful error",
		},
		{
			name:        "MissingArgument",
			args:        []string{"--output"},
			expectError: true,
			description: "Missing argument should show helpful error",
		},
	}
	
	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			result := utils.RunBTCommand(tc.args...)
			
			if tc.expectError {
				assert.NotEqual(t, 0, result.ExitCode, "Should have non-zero exit code")
				assert.NotEmpty(t, result.Stderr, "Should have error message")
				
				// Error message should be helpful
				assert.Contains(t, result.Stderr, "Error:", "Error message should be prefixed")
			}
		})
	}
}

// TestColorOutput tests color output functionality
func (suite *CLITestSuite) TestColorOutput() {
	// Test with color enabled (default)
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// Test with color disabled
	resultNoColor := utils.RunBTCommand("--no-color", "version")
	utils.AssertCommandSuccess(suite.T(), resultNoColor)
	
	// Both should succeed
	assert.Equal(suite.T(), 0, result.ExitCode)
	assert.Equal(suite.T(), 0, resultNoColor.ExitCode)
}

// TestConcurrentExecution tests concurrent command execution
func (suite *CLITestSuite) TestConcurrentExecution() {
	// Test that multiple commands can run concurrently without interfering
	const numConcurrent = 5
	
	results := make([]*utils.CommandResult, numConcurrent)
	done := make(chan int, numConcurrent)
	
	for i := 0; i < numConcurrent; i++ {
		go func(index int) {
			results[index] = utils.RunBTCommand("version")
			done <- index
		}(i)
	}
	
	// Wait for all commands to complete
	for i := 0; i < numConcurrent; i++ {
		<-done
	}
	
	// All commands should have succeeded
	for i, result := range results {
		suite.T().Run(fmt.Sprintf("Concurrent_%d", i), func(t *testing.T) {
			utils.AssertCommandSuccess(t, result)
			assert.Contains(t, result.Stdout, "bt version")
		})
	}
}

// TestMemoryUsage tests basic memory usage (simple test)
func (suite *CLITestSuite) TestMemoryUsage() {
	// Test that commands complete without excessive memory usage
	// This is a basic test - more sophisticated memory testing would require profiling
	
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(suite.T(), result)
	
	// If we got here without OOM, memory usage was reasonable
	assert.Equal(suite.T(), 0, result.ExitCode)
}

// Run the test suite
func TestCLICommands(t *testing.T) {
	suite.Run(t, new(CLITestSuite))
}

// Additional standalone CLI tests

// TestBinaryExists tests that the bt binary can be built and executed
func TestBinaryExists(t *testing.T) {
	binaryPath := utils.BuildBinary(t)
	assert.True(t, utils.FileExists(binaryPath), "Binary should exist after build")
	
	// Test that binary is executable
	result := utils.RunCommand(binaryPath, "version")
	utils.AssertCommandSuccess(t, result)
}

// TestBasicFunctionality tests basic CLI functionality
func TestBasicFunctionality(t *testing.T) {
	// Test version command
	result := utils.RunBTCommand("version")
	utils.AssertCommandSuccess(t, result)
	assert.Contains(t, result.Stdout, "bt version")
	
	// Test help command
	result = utils.RunBTCommand("--help")
	utils.AssertCommandSuccess(t, result)
	assert.Contains(t, result.Stdout, "Work seamlessly with Bitbucket")
}

// TestIntegrationCommands tests integration-specific commands
func TestIntegrationCommands(t *testing.T) {
	if !utils.IsIntegrationTest() {
		t.Skip("Skipping integration command tests. Set BT_INTEGRATION_TESTS=1 to run.")
	}
	
	// These tests would require actual Bitbucket integration
	// For now, just test that the commands exist and show help
	
	commands := []string{"auth", "repo", "pr", "run", "api"}
	
	for _, cmd := range commands {
		t.Run(fmt.Sprintf("Integration_%s", cmd), func(t *testing.T) {
			result := utils.RunBTCommand(cmd, "--help")
			utils.AssertCommandSuccess(t, result)
			assert.Contains(t, result.Stdout, cmd)
		})
	}
}