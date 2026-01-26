package shared

import (
	"fmt"
	"strings"
)

var AllowedPRStates = []string{"open", "merged", "declined", "all"}

var AllowedPipelineStatuses = []string{"PENDING", "IN_PROGRESS", "SUCCESSFUL", "FAILED", "ERROR", "STOPPED"}

func ValidateAllowedValue(value string, allowed []string, fieldName string) error {
	valueLower := strings.ToLower(value)
	for _, v := range allowed {
		if valueLower == strings.ToLower(v) {
			return nil
		}
	}

	return fmt.Errorf("invalid %s '%s'. Valid values are: %s",
		fieldName, value, strings.Join(allowed, ", "))
}
