package shared

import (
	"fmt"

	"github.com/carlosarraes/bt/pkg/api"
)

type APIDomain string

const (
	DomainPullRequest APIDomain = "pull_request"
	DomainPipeline    APIDomain = "pipeline"
)

func HandleAPIError(err error, domain APIDomain) error {
	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		switch bitbucketErr.Type {
		case api.ErrorTypeNotFound:
			return notFoundError(domain)
		case api.ErrorTypeAuthentication:
			return fmt.Errorf("authentication failed. Please run 'bt auth login' to authenticate")
		case api.ErrorTypePermission:
			return fmt.Errorf("permission denied. You may not have access to this repository")
		case api.ErrorTypeRateLimit:
			return fmt.Errorf("rate limit exceeded. Please wait before making more requests")
		default:
			return fmt.Errorf("API error: %s", bitbucketErr.Message)
		}
	}

	return fallbackError(err, domain)
}

func notFoundError(domain APIDomain) error {
	switch domain {
	case DomainPullRequest:
		return fmt.Errorf("repository not found or no pull requests exist. Verify the repository exists and you have access")
	case DomainPipeline:
		return fmt.Errorf("repository not found or pipelines not enabled. Verify the repository exists and has Bitbucket Pipelines enabled")
	default:
		return fmt.Errorf("resource not found")
	}
}

func fallbackError(err error, domain APIDomain) error {
	switch domain {
	case DomainPullRequest:
		return fmt.Errorf("API request failed: %w", err)
	case DomainPipeline:
		return fmt.Errorf("failed to list pipelines: %w", err)
	default:
		return fmt.Errorf("API request failed: %w", err)
	}
}
