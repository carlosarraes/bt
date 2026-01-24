package pr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/api"
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type PRContext = shared.CommandContext

// PullRequestStateColor returns the appropriate color for a pull request state
func PullRequestStateColor(state string) string {
	switch state {
	case "OPEN":
		return "green"
	case "MERGED":
		return "blue"
	case "DECLINED":
		return "red"
	case "SUPERSEDED":
		return "yellow"
	default:
		return "white"
	}
}

func handlePullRequestAPIError(err error) error {
	if bitbucketErr, ok := err.(*api.BitbucketError); ok {
		switch bitbucketErr.Type {
		case api.ErrorTypeNotFound:
			return fmt.Errorf("repository not found or no pull requests exist. Verify the repository exists and you have access")
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

	return fmt.Errorf("API request failed: %w", err)
}

func ParsePRID(prIDStr string) (int, error) {
	if prIDStr == "" {
		return 0, fmt.Errorf("pull request ID is required")
	}

	prIDStr = strings.TrimPrefix(prIDStr, "#")

	prID, err := strconv.Atoi(prIDStr)
	if err != nil {
		return 0, fmt.Errorf("invalid pull request ID '%s': must be a number", prIDStr)
	}

	if prID <= 0 {
		return 0, fmt.Errorf("invalid pull request ID '%d': must be positive", prID)
	}

	return prID, nil
}
