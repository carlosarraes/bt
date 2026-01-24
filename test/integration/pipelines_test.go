package integration

import (
	"context"
	"testing"
	"time"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PipelineIntegrationSuite provides comprehensive pipeline API integration tests
type PipelineIntegrationSuite struct {
	suite.Suite
	client    *api.Client
	testCtx   *utils.TestContext
	workspace string
	repo      string
}

// SetupSuite initializes the test suite
func (suite *PipelineIntegrationSuite) SetupSuite() {
	utils.RequireIntegrationEnv(suite.T())

	suite.testCtx = utils.NewTestContext(suite.T())
	suite.workspace = utils.GetTestWorkspace()
	suite.repo = utils.GetTestRepo()

	// Create an authenticated API client
	authManager, err := utils.CreateTestAuthManager()
	require.NoError(suite.T(), err, "Failed to create auth manager")

	clientConfig := api.DefaultClientConfig()
	clientConfig.EnableLogging = true
	clientConfig.Logger = utils.GetTestLogger()

	suite.client, err = api.NewClient(authManager, clientConfig)
	require.NoError(suite.T(), err, "Failed to create API client")

	// Verify API connectivity
	suite.verifyAPIConnectivity()
}

// TearDownSuite cleans up after the test suite
func (suite *PipelineIntegrationSuite) TearDownSuite() {
	if suite.testCtx != nil {
		suite.testCtx.Close()
	}
}

// verifyAPIConnectivity ensures we can connect to the Bitbucket API
func (suite *PipelineIntegrationSuite) verifyAPIConnectivity() {
	ctx := context.Background()
	resp, err := suite.client.Get(ctx, "user")
	require.NoError(suite.T(), err, "Failed to connect to Bitbucket API")
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		suite.T().Fatal("API authentication failed. Check BT_TEST_* environment variables.")
	}

	require.True(suite.T(), resp.StatusCode < 400,
		"API connectivity check failed with status %d", resp.StatusCode)
}

// TestListPipelines tests the ListPipelines API method
func (suite *PipelineIntegrationSuite) TestListPipelines() {
	ctx := context.Background()

	// Test basic pipeline listing
	result, err := suite.client.Pipelines.ListPipelines(ctx, suite.workspace, suite.repo, nil)

	if err != nil {
		// Check if it's a 404 (pipelines not enabled)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Pipelines not enabled for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Failed to list pipelines")
	}

	require.NotNil(suite.T(), result, "Pipeline list result should not be nil")

	// Verify pagination structure
	assert.GreaterOrEqual(suite.T(), result.Size, 0, "Size should be non-negative")
	assert.GreaterOrEqual(suite.T(), result.Page, 1, "Page should be at least 1")
	assert.GreaterOrEqual(suite.T(), result.PageLen, 0, "PageLen should be non-negative")

	suite.T().Logf("Found %d pipelines", result.Size)
}

// TestListPipelinesWithOptions tests pipeline listing with various filter options
func (suite *PipelineIntegrationSuite) TestListPipelinesWithOptions() {
	ctx := context.Background()

	// Test with pagination options
	options := &api.PipelineListOptions{
		PageLen: 5,
		Page:    1,
		Sort:    "-created_on", // Most recent first
	}

	result, err := suite.client.Pipelines.ListPipelines(ctx, suite.workspace, suite.repo, options)

	if err != nil {
		// Check if it's a 404 (pipelines not enabled)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Pipelines not enabled for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Failed to list pipelines with options")
	}

	require.NotNil(suite.T(), result, "Pipeline list result should not be nil")

	// Verify pagination was applied
	assert.LessOrEqual(suite.T(), result.PageLen, 5, "PageLen should be at most 5")

	suite.T().Logf("Found %d pipelines with options", result.Size)
}

// TestGetPipeline tests retrieving a specific pipeline
func (suite *PipelineIntegrationSuite) TestGetPipeline() {
	ctx := context.Background()

	// First, get a list of pipelines to find one to test with
	result, err := suite.client.Pipelines.ListPipelines(ctx, suite.workspace, suite.repo, &api.PipelineListOptions{
		PageLen: 1,
		Page:    1,
	})

	if err != nil {
		// Check if it's a 404 (pipelines not enabled)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Pipelines not enabled for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Failed to list pipelines")
	}

	// Parse the first pipeline from the result
	var pipelines []*api.Pipeline
	if result.Size > 0 {
		// We need to unmarshal the Values field
		// For now, we'll skip this test if no pipelines are found
		if result.Size == 0 {
			suite.T().Skipf("No pipelines found to test GetPipeline")
			return
		}

		// We would need to parse the JSON properly here
		suite.T().Logf("Would test GetPipeline with pipeline, but need to implement JSON parsing")
		suite.T().Skip("Skipping GetPipeline test until JSON parsing is implemented")
	}

	suite.T().Logf("Pipeline count: %d", len(pipelines))
}

// TestPipelineService methods validation
func (suite *PipelineIntegrationSuite) TestPipelineServiceValidation() {
	ctx := context.Background()

	// Test validation for ListPipelines
	_, err := suite.client.Pipelines.ListPipelines(ctx, "", suite.repo, nil)
	assert.Error(suite.T(), err, "Should error with empty workspace")

	_, err = suite.client.Pipelines.ListPipelines(ctx, suite.workspace, "", nil)
	assert.Error(suite.T(), err, "Should error with empty repo")

	// Test validation for GetPipeline
	_, err = suite.client.Pipelines.GetPipeline(ctx, "", suite.repo, "test-uuid")
	assert.Error(suite.T(), err, "Should error with empty workspace")

	_, err = suite.client.Pipelines.GetPipeline(ctx, suite.workspace, "", "test-uuid")
	assert.Error(suite.T(), err, "Should error with empty repo")

	_, err = suite.client.Pipelines.GetPipeline(ctx, suite.workspace, suite.repo, "")
	assert.Error(suite.T(), err, "Should error with empty pipeline UUID")

	// Test validation for GetPipelineSteps
	_, err = suite.client.Pipelines.GetPipelineSteps(ctx, "", suite.repo, "test-uuid")
	assert.Error(suite.T(), err, "Should error with empty workspace")

	// Test validation for GetStepLogs
	_, err = suite.client.Pipelines.GetStepLogs(ctx, "", suite.repo, "test-uuid", "step-uuid")
	assert.Error(suite.T(), err, "Should error with empty workspace")

	_, err = suite.client.Pipelines.GetStepLogs(ctx, suite.workspace, "", "test-uuid", "step-uuid")
	assert.Error(suite.T(), err, "Should error with empty repo")

	_, err = suite.client.Pipelines.GetStepLogs(ctx, suite.workspace, suite.repo, "", "step-uuid")
	assert.Error(suite.T(), err, "Should error with empty pipeline UUID")

	_, err = suite.client.Pipelines.GetStepLogs(ctx, suite.workspace, suite.repo, "test-uuid", "")
	assert.Error(suite.T(), err, "Should error with empty step UUID")

	// Test validation for CancelPipeline
	err = suite.client.Pipelines.CancelPipeline(ctx, "", suite.repo, "test-uuid")
	assert.Error(suite.T(), err, "Should error with empty workspace")

	err = suite.client.Pipelines.CancelPipeline(ctx, suite.workspace, "", "test-uuid")
	assert.Error(suite.T(), err, "Should error with empty repo")

	err = suite.client.Pipelines.CancelPipeline(ctx, suite.workspace, suite.repo, "")
	assert.Error(suite.T(), err, "Should error with empty pipeline UUID")
}

// TestPipelineErrorHandling tests error handling for various scenarios
func (suite *PipelineIntegrationSuite) TestPipelineErrorHandling() {
	ctx := context.Background()

	// Test with non-existent repository
	_, err := suite.client.Pipelines.ListPipelines(ctx, "nonexistent", "nonexistent", nil)
	assert.Error(suite.T(), err, "Should error with non-existent repository")

	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		assert.Equal(suite.T(), 404, bitbucketErr.StatusCode, "Should return 404 for non-existent repository")
		assert.Equal(suite.T(), api.ErrorTypeNotFound, bitbucketErr.Type, "Should be NotFound error type")
	}

	// Test getting non-existent pipeline
	_, err = suite.client.Pipelines.GetPipeline(ctx, suite.workspace, suite.repo, "nonexistent-uuid")
	assert.Error(suite.T(), err, "Should error with non-existent pipeline")

	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		assert.Equal(suite.T(), 404, bitbucketErr.StatusCode, "Should return 404 for non-existent pipeline")
	}
}

// TestTriggerPipelineValidation tests pipeline triggering validation
func (suite *PipelineIntegrationSuite) TestTriggerPipelineValidation() {
	ctx := context.Background()

	// Test validation for TriggerPipeline
	_, err := suite.client.Pipelines.TriggerPipeline(ctx, "", suite.repo, nil)
	assert.Error(suite.T(), err, "Should error with empty workspace")

	_, err = suite.client.Pipelines.TriggerPipeline(ctx, suite.workspace, "", nil)
	assert.Error(suite.T(), err, "Should error with empty repo")

	_, err = suite.client.Pipelines.TriggerPipeline(ctx, suite.workspace, suite.repo, nil)
	assert.Error(suite.T(), err, "Should error with nil request")

	// Test with invalid request (no target)
	invalidRequest := &api.TriggerPipelineRequest{}
	_, err = suite.client.Pipelines.TriggerPipeline(ctx, suite.workspace, suite.repo, invalidRequest)
	assert.Error(suite.T(), err, "Should error with request missing target")
}

// TestPipelinePerformance tests pipeline API performance
func (suite *PipelineIntegrationSuite) TestPipelinePerformance() {
	ctx := context.Background()

	// Test that pipeline listing completes within reasonable time
	start := time.Now()

	_, err := suite.client.Pipelines.ListPipelines(ctx, suite.workspace, suite.repo, &api.PipelineListOptions{
		PageLen: 10,
	})

	duration := time.Since(start)

	// Allow for pipelines not being enabled (404 error)
	if err != nil {
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Pipelines not enabled for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Performance test should not fail due to API errors")
	}

	// Performance target: <500ms for pipeline listing
	assert.Less(suite.T(), duration, 500*time.Millisecond,
		"Pipeline listing should complete within 500ms, took %v", duration)

	suite.T().Logf("Pipeline listing took %v", duration)
}

// TestArtifactAPI tests artifact-related API endpoints
func (suite *PipelineIntegrationSuite) TestArtifactAPI() {
	ctx := context.Background()

	// Test listing artifacts
	artifacts, err := suite.client.Pipelines.ListArtifacts(ctx, suite.workspace, suite.repo)

	if err != nil {
		// Check if it's a 404 (artifacts not available)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Artifacts not available for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Failed to list artifacts")
	}

	suite.T().Logf("Found %d artifacts", len(artifacts))

	// If we have artifacts, test downloading one
	if len(artifacts) > 0 {
		artifact := artifacts[0]
		assert.NotEmpty(suite.T(), artifact.UUID, "Artifact should have UUID")
		assert.NotEmpty(suite.T(), artifact.Name, "Artifact should have name")

		// Test downloading artifact (we won't actually download the content)
		reader, err := suite.client.Pipelines.DownloadArtifact(ctx, suite.workspace, suite.repo, artifact.UUID)
		if err == nil {
			reader.Close() // Close immediately to avoid downloading large files
			suite.T().Logf("Successfully initiated download for artifact: %s", artifact.Name)
		} else {
			suite.T().Logf("Download failed for artifact %s: %v", artifact.Name, err)
		}
	}
}

// TestConvenienceMethods tests the convenience methods
func (suite *PipelineIntegrationSuite) TestConvenienceMethods() {
	ctx := context.Background()

	// Test GetPipelinesByBranch
	pipelines, err := suite.client.Pipelines.GetPipelinesByBranch(ctx, suite.workspace, suite.repo, "main", 5)

	if err != nil {
		// Check if it's a 404 (pipelines not enabled)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Pipelines not enabled for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Failed to get pipelines by branch")
	}

	suite.T().Logf("Found %d pipelines for main branch", len(pipelines))

	// Test GetFailedPipelines
	failedPipelines, err := suite.client.Pipelines.GetFailedPipelines(ctx, suite.workspace, suite.repo, 5)

	if err != nil {
		// Check if it's a 404 (pipelines not enabled)
		if bitbucketErr, ok := err.(*api.BitbucketError); ok && bitbucketErr.StatusCode == 404 {
			suite.T().Skipf("Pipelines not enabled for repository '%s/%s'", suite.workspace, suite.repo)
			return
		}
		require.NoError(suite.T(), err, "Failed to get failed pipelines")
	}

	suite.T().Logf("Found %d failed pipelines", len(failedPipelines))
}

// TestLogStreaming tests the log streaming functionality
func (suite *PipelineIntegrationSuite) TestLogStreaming() {
	ctx := context.Background()

	// We can't easily test log streaming without a real pipeline
	// So we'll test the error handling for invalid pipeline/step UUIDs

	logReader, err := suite.client.Pipelines.GetStepLogs(ctx, suite.workspace, suite.repo, "invalid-pipeline-uuid", "invalid-step-uuid")

	// This should fail with 404
	assert.Error(suite.T(), err, "Should error with invalid pipeline UUID")
	assert.Nil(suite.T(), logReader, "Log reader should be nil on error")

	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		assert.Equal(suite.T(), 404, bitbucketErr.StatusCode, "Should return 404 for invalid pipeline")
	}

	// Test streaming with range
	logReader, err = suite.client.Pipelines.GetStepLogsWithRange(ctx, suite.workspace, suite.repo, "invalid-pipeline-uuid", "invalid-step-uuid", 0, 1024)

	assert.Error(suite.T(), err, "Should error with invalid pipeline UUID for range request")
	assert.Nil(suite.T(), logReader, "Log reader should be nil on error")
}

// Run the test suite
func TestPipelineIntegration(t *testing.T) {
	suite.Run(t, new(PipelineIntegrationSuite))
}

// Standalone pipeline integration tests

// TestPipelineBasicConnectivity tests basic pipeline API connectivity
func TestPipelineBasicConnectivity(t *testing.T) {
	utils.SkipIfNoIntegration(t)

	workspace := utils.GetTestWorkspace()
	repo := utils.GetTestRepo()

	authManager, err := utils.CreateTestAuthManager()
	require.NoError(t, err, "Failed to create auth manager")

	client, err := api.NewClient(authManager, nil)
	require.NoError(t, err, "Failed to create API client")

	ctx := context.Background()

	// Test basic pipeline listing
	_, err = client.Pipelines.ListPipelines(ctx, workspace, repo, nil)

	// Allow 404 for repositories without pipelines enabled
	if err != nil {
		if bitbucketErr, ok := err.(*api.BitbucketError); ok {
			if bitbucketErr.StatusCode == 404 {
				t.Logf("Pipelines not enabled for repository '%s/%s' - this is expected", workspace, repo)
				return
			}

			// Other client errors might indicate auth issues
			if bitbucketErr.StatusCode == 401 || bitbucketErr.StatusCode == 403 {
				t.Fatalf("Pipeline API authentication failed: %v", err)
			}
		}

		// Network or other errors
		t.Fatalf("Pipeline API connectivity failed: %v", err)
	}

	t.Logf("Pipeline API connectivity test passed")
}

// TestPipelineStructValidation tests that our pipeline structs are correctly defined
func TestPipelineStructValidation(t *testing.T) {
	// Test that we can create and populate pipeline structs
	pipeline := &api.Pipeline{
		UUID:        "test-uuid",
		BuildNumber: 123,
		State: &api.PipelineState{
			Name: "SUCCESSFUL",
		},
		Target: &api.PipelineTarget{
			RefType: "branch",
			RefName: "main",
		},
	}

	assert.Equal(t, "test-uuid", pipeline.UUID)
	assert.Equal(t, 123, pipeline.BuildNumber)
	assert.Equal(t, "SUCCESSFUL", pipeline.State.Name)
	assert.Equal(t, "branch", pipeline.Target.RefType)
	assert.Equal(t, "main", pipeline.Target.RefName)

	// Test pipeline state constants
	assert.Equal(t, "PENDING", api.PipelineStatePending.String())
	assert.Equal(t, "IN_PROGRESS", api.PipelineStateInProgress.String())
	assert.Equal(t, "SUCCESSFUL", api.PipelineStateSuccessful.String())
	assert.Equal(t, "FAILED", api.PipelineStateFailed.String())
	assert.Equal(t, "ERROR", api.PipelineStateError.String())
	assert.Equal(t, "STOPPED", api.PipelineStateStopped.String())

	t.Logf("Pipeline struct validation passed")
}
