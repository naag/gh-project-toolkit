package sync_fields

import (
	"fmt"
	"strings"
)

// FieldMapping represents a mapping between source and target field names
type FieldMapping struct {
	SourceField string
	TargetField string
}

// ParseFieldMappings parses a list of field mapping strings in the format "source:target"
// and returns a slice of FieldMapping structs
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
