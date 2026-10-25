package internal_utils

import (
	"encoding/csv"
	"fmt"
	"strings"
)

var supportedBuildArchitectures = map[string]struct{}{
	"android_aarch64": {},
	"linux_aarch64":   {},
	"linux_amd64":     {},
}

// ParseBuildArchitectures validates a CSV architecture selection. An empty
// value selects every supported architecture.
func ParseBuildArchitectures(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{"linux_amd64", "linux_aarch64", "android_aarch64"}, nil
	}

	reader := csv.NewReader(strings.NewReader(value))
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse targetBuilds parameter: %w", err)
	}
	if len(records) != 1 || len(records[0]) == 0 {
		return nil, fmt.Errorf("targetBuilds must contain at least one supported architecture")
	}
	record := records[0]

	selected := make([]string, 0, len(record))
	seen := make(map[string]struct{}, len(record))
	for _, item := range record {
		architecture := strings.TrimSpace(item)
		if _, ok := supportedBuildArchitectures[architecture]; !ok {
			return nil, fmt.Errorf("invalid target build %q; allowed values are android_aarch64, linux_aarch64, linux_amd64", architecture)
		}
		if _, duplicate := seen[architecture]; duplicate {
			return nil, fmt.Errorf("targetBuilds contains duplicate architecture %q", architecture)
		}
		seen[architecture] = struct{}{}
		selected = append(selected, architecture)
	}
	return selected, nil
}
