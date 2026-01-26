package pr

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type PRContext = shared.CommandContext

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
	return shared.HandleAPIError(err, shared.DomainPullRequest)
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
