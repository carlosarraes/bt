# pkg/api

Bitbucket API v2 client implementation.

## Overview

This package provides a comprehensive client for the Bitbucket Cloud REST API v2. It handles authentication, rate limiting, error handling, and pagination.

## Files (To Be Implemented)

- `client.go` - Core HTTP client with authentication
- `errors.go` - Bitbucket API error handling
- `pagination.go` - Cursor-based pagination support
- `pipelines.go` - Pipeline API methods (critical priority)
- `repositories.go` - Repository API methods
- `pullrequests.go` - Pull request API methods
- `types.go` - API response struct definitions

## Key Features

- **Authentication Integration**: Automatic auth header injection
- **Rate Limiting**: Exponential backoff on HTTP 429 responses  
- **Error Handling**: Consistent Bitbucket error format parsing
- **Pagination**: Transparent cursor-based pagination
- **Performance**: <500ms target response time
- **Real Data**: No mock responses, designed for real API integration

## API Base URL

All requests are made to: `https://api.bitbucket.org/2.0`

## Critical Endpoints (Pipeline Debugging)

The pipeline API methods are critical priority for the 5x faster debugging feature:

- `GET /repositories/{workspace}/{repo}/pipelines` - List runs
- `GET /repositories/{workspace}/{repo}/pipelines/{uuid}` - Get run details
- `GET /repositories/{workspace}/{repo}/pipelines/{uuid}/steps/{step}/log` - Stream logs
- `POST /repositories/{workspace}/{repo}/pipelines/{uuid}/stopPipeline` - Cancel runs