package shared

import (
	"encoding/json"
	"fmt"

	"github.com/carlosarraes/bt/pkg/api"
)

func ParsePaginatedResults[T any](result *api.PaginatedResponse) ([]*T, error) {
	if result.Values == nil {
		return nil, nil
	}

	var values []json.RawMessage
	if err := json.Unmarshal(result.Values, &values); err != nil {
		return nil, fmt.Errorf("failed to unmarshal values: %w", err)
	}

	items := make([]*T, len(values))
	for i, raw := range values {
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("failed to unmarshal item %d: %w", i, err)
		}
		items[i] = &item
	}

	return items, nil
}
