# Pipeline API Usage

This document demonstrates how to use the newly implemented Pipeline API methods in the Bitbucket CLI.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/carlosarraes/bt/pkg/api"
    "github.com/carlosarraes/bt/pkg/auth"
)

func main() {
    // Create auth manager
    authManager := auth.NewAppPasswordAuth("username", "app_password")
    
    // Create API client
    client, err := api.NewClient(authManager, nil)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    workspace := "my-workspace"
    repo := "my-repo"
    
    // List recent pipelines
    pipelines, err := client.Pipelines.ListPipelines(ctx, workspace, repo, &api.PipelineListOptions{
        PageLen: 10,
        Sort:    "-created_on",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("Found %d pipelines\n", pipelines.Size)
}
```

## Available Methods

### 1. ListPipelines
List pipelines with optional filtering and pagination.

```go
options := &api.PipelineListOptions{
    Status:  "FAILED",      // Filter by status
    Branch:  "main",        // Filter by branch
    Sort:    "-created_on", // Sort by creation date (newest first)
    PageLen: 20,            // Items per page
}

result, err := client.Pipelines.ListPipelines(ctx, workspace, repo, options)
```

### 2. GetPipeline
Get detailed information about a specific pipeline.

```go
pipeline, err := client.Pipelines.GetPipeline(ctx, workspace, repo, pipelineUUID)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Pipeline %d: %s\n", pipeline.BuildNumber, pipeline.State.Name)
```

### 3. GetPipelineSteps
Get all steps for a specific pipeline.

```go
steps, err := client.Pipelines.GetPipelineSteps(ctx, workspace, repo, pipelineUUID)
if err != nil {
    log.Fatal(err)
}

for _, step := range steps {
    fmt.Printf("Step: %s - %s\n", step.Name, step.State.Name)
}
```

### 4. GetStepLogs (Streaming)
Get logs for a specific pipeline step with streaming support.

```go
logReader, err := client.Pipelines.GetStepLogs(ctx, workspace, repo, pipelineUUID, stepUUID)
if err != nil {
    log.Fatal(err)
}
defer logReader.Close()

// Read logs line by line
scanner := bufio.NewScanner(logReader)
for scanner.Scan() {
    fmt.Println(scanner.Text())
}
```

### 5. StreamStepLogs (Channel-based)
Stream logs using Go channels for easier processing.

```go
logChan, errChan := client.Pipelines.StreamStepLogs(ctx, workspace, repo, pipelineUUID, stepUUID)

for {
    select {
    case line, ok := <-logChan:
        if !ok {
            return // Done
        }
        fmt.Println(line)
    case err := <-errChan:
        log.Printf("Error: %v", err)
        return
    case <-ctx.Done():
        return
    }
}
```

### 6. GetStepLogsWithRange
Get logs with HTTP Range support for large files.

```go
// Get first 1KB of logs
logReader, err := client.Pipelines.GetStepLogsWithRange(ctx, workspace, repo, pipelineUUID, stepUUID, 0, 1024)
if err != nil {
    log.Fatal(err)
}
defer logReader.Close()
```

### 7. CancelPipeline
Cancel a running pipeline.

```go
err := client.Pipelines.CancelPipeline(ctx, workspace, repo, pipelineUUID)
if err != nil {
    log.Fatal(err)
}
fmt.Println("Pipeline cancelled successfully")
```

### 8. TriggerPipeline
Trigger a new pipeline execution.

```go
request := &api.TriggerPipelineRequest{
    Target: &api.PipelineTarget{
        RefType: "branch",
        RefName: "main",
    },
    Variables: []*api.PipelineVariable{
        {
            Key:   "MY_VAR",
            Value: "my_value",
        },
    },
}

pipeline, err := client.Pipelines.TriggerPipeline(ctx, workspace, repo, request)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Started pipeline %d\n", pipeline.BuildNumber)
```

### 9. ListArtifacts & DownloadArtifact
Work with pipeline artifacts.

```go
// List artifacts
artifacts, err := client.Pipelines.ListArtifacts(ctx, workspace, repo)
if err != nil {
    log.Fatal(err)
}

// Download the first artifact
if len(artifacts) > 0 {
    artifact := artifacts[0]
    reader, err := client.Pipelines.DownloadArtifact(ctx, workspace, repo, artifact.UUID)
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()
    
    // Save to file or process the content
    fmt.Printf("Downloading artifact: %s\n", artifact.Name)
}
```

## Convenience Methods

### GetPipelinesByBranch
Get pipelines for a specific branch.

```go
pipelines, err := client.Pipelines.GetPipelinesByBranch(ctx, workspace, repo, "main", 10)
```

### GetFailedPipelines
Get recently failed pipelines.

```go
failedPipelines, err := client.Pipelines.GetFailedPipelines(ctx, workspace, repo, 5)
```

### WaitForPipelineCompletion
Poll a pipeline until it completes.

```go
// Poll every 5 seconds
finalPipeline, err := client.Pipelines.WaitForPipelineCompletion(ctx, workspace, repo, pipelineUUID, 5)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Pipeline completed with status: %s\n", finalPipeline.State.Name)
```

## Log Retrieval Improvements

The `GetStepLogs` method has been enhanced with better error reporting and multiple endpoint attempts:

1. **Multiple Endpoint Support**: Tries both singular (`/log`) and plural (`/logs`) endpoints
2. **Better Error Reporting**: Shows original API errors instead of masking them with generic messages
3. **Fallback Strategy**: Uses step details to retrieve dynamic log URLs when direct endpoints fail

```go
// Enhanced log retrieval with better error messages
logReader, err := client.Pipelines.GetStepLogs(ctx, workspace, repo, pipelineUUID, stepUUID)
if err != nil {
    // Error messages now include original API response details
    fmt.Printf("Log retrieval failed: %v\n", err)
    return
}
defer logReader.Close()

// Process logs
scanner := bufio.NewScanner(logReader)
for scanner.Scan() {
    fmt.Printf("LOG: %s\n", scanner.Text())
}
```

### Common Log Retrieval Issues

- **404 Not Found**: Step UUID doesn't exist or pipeline hasn't run yet
- **403 Permission Denied**: Missing `pipelines:read` OAuth scope or repository access
- **No logs available**: Step hasn't started, failed before logging, or logs were cleaned up
- **Endpoint variations**: Some repositories use different log endpoint formats

## Error Handling

All methods return structured Bitbucket API errors:

```go
pipelines, err := client.Pipelines.ListPipelines(ctx, workspace, repo, nil)
if err != nil {
    if bitbucketErr, ok := err.(*api.BitbucketError); ok {
        switch bitbucketErr.Type {
        case api.ErrorTypeNotFound:
            fmt.Println("Repository not found or pipelines not enabled")
        case api.ErrorTypeAuthentication:
            fmt.Println("Authentication failed")
        case api.ErrorTypeRateLimit:
            fmt.Println("Rate limit exceeded")
        default:
            fmt.Printf("API error: %s\n", bitbucketErr.Message)
        }
    } else {
        fmt.Printf("Network error: %v\n", err)
    }
}
```

## Performance Considerations

- **Streaming Logs**: Use `GetStepLogs()` or `StreamStepLogs()` for large log files
- **Range Requests**: Use `GetStepLogsWithRange()` for partial log downloads
- **Pagination**: Use appropriate `PageLen` values (default: 50, max: 100)
- **Timeouts**: Configure client timeout for long-running operations

## Integration Testing

Integration tests are available in `test/integration/pipelines_test.go`. To run them:

```bash
# Set environment variables
export BT_INTEGRATION_TESTS=1
export BT_TEST_USERNAME=your_username
export BT_TEST_APP_PASSWORD=your_app_password
export BT_TEST_WORKSPACE=your_workspace
export BT_TEST_REPO=your_repo_with_pipelines

# Run tests
go test ./test/integration/pipelines_test.go -v
```

## API Endpoints Implemented

- `GET /repositories/{workspace}/{repo}/pipelines` - List pipelines
- `GET /repositories/{workspace}/{repo}/pipelines/{uuid}` - Get pipeline details
- `GET /repositories/{workspace}/{repo}/pipelines/{uuid}/steps` - Get pipeline steps
- `GET /repositories/{workspace}/{repo}/pipelines/{uuid}/steps/{step_uuid}/log` - Get step logs
- `POST /repositories/{workspace}/{repo}/pipelines/{uuid}/stopPipeline` - Cancel pipeline
- `POST /repositories/{workspace}/{repo}/pipelines` - Trigger pipeline
- `GET /repositories/{workspace}/{repo}/downloads` - List artifacts
- `GET /repositories/{workspace}/{repo}/downloads/{uuid}` - Download artifact