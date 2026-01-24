package run

import (
	"github.com/carlosarraes/bt/pkg/cmd/shared"
)

type RunContext = shared.CommandContext

func PipelineStateColor(state string) string {
	switch state {
	case "SUCCESSFUL":
		return "green"
	case "FAILED":
		return "red"
	case "ERROR":
		return "red"
	case "STOPPED":
		return "yellow"
	case "IN_PROGRESS":
		return "blue"
	case "PENDING":
		return "cyan"
	default:
		return "white"
	}
}
