package sync_fields

import (
	"fmt"
	"strings"
)

type FieldMapping struct {
	SourceField string
	TargetField string
}

func ParseFieldMappings(fieldMappings []string) ([]FieldMapping, error) {
	mappings := make([]FieldMapping, 0, len(fieldMappings))
	for _, mapping := range fieldMappings {
		parts := strings.Split(mapping, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field mapping format: %s", mapping)
		}
		mappings = append(mappings, FieldMapping{
			SourceField: strings.TrimSpace(parts[0]),
			TargetField: strings.TrimSpace(parts[1]),
		})
	}
	return mappings, nil
}
