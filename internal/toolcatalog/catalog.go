package toolcatalog

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// Annotations contains transport-neutral MCP tool hints.
type Annotations struct {
	ReadOnlyHint    bool  `json:"readOnlyHint"`
	IdempotentHint  bool  `json:"idempotentHint,omitempty"`
	DestructiveHint *bool `json:"destructiveHint,omitempty"`
	OpenWorldHint   *bool `json:"openWorldHint,omitempty"`
}

// Definition is the authoritative metadata for one MCP tool.
type Definition struct {
	Name        string      `json:"name"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Annotations Annotations `json:"annotations"`
}

type document struct {
	Tools []Definition `json:"tools"`
}

//go:embed catalog.json
var catalogJSON []byte

var (
	definitions       []Definition
	definitionsByName map[string]Definition
)

func init() {
	var parsed document
	if err := json.Unmarshal(catalogJSON, &parsed); err != nil {
		panic(fmt.Sprintf("parse embedded tool catalog: %v", err))
	}
	if len(parsed.Tools) == 0 {
		panic("embedded tool catalog is empty")
	}

	definitions = make([]Definition, 0, len(parsed.Tools))
	definitionsByName = make(map[string]Definition, len(parsed.Tools))
	for index, definition := range parsed.Tools {
		if err := validateDefinition(definition); err != nil {
			panic(fmt.Sprintf("invalid tool catalog entry %d: %v", index, err))
		}
		if _, exists := definitionsByName[definition.Name]; exists {
			panic(fmt.Sprintf("duplicate tool catalog name %q", definition.Name))
		}
		definition = cloneDefinition(definition)
		definitions = append(definitions, definition)
		definitionsByName[definition.Name] = definition
	}
}

func validateDefinition(definition Definition) error {
	if strings.TrimSpace(definition.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(definition.Title) == "" {
		return fmt.Errorf("title is required for %q", definition.Name)
	}
	if strings.TrimSpace(definition.Description) == "" {
		return fmt.Errorf("description is required for %q", definition.Name)
	}
	return nil
}

// All returns an independent copy in registration order.
func All() []Definition {
	result := make([]Definition, len(definitions))
	for index, definition := range definitions {
		result[index] = cloneDefinition(definition)
	}
	return result
}

// Lookup returns an independent copy of the named definition.
func Lookup(name string) (Definition, bool) {
	definition, ok := definitionsByName[name]
	if !ok {
		return Definition{}, false
	}
	return cloneDefinition(definition), true
}

// Must returns the named definition or panics for a programmer error.
func Must(name string) Definition {
	definition, ok := Lookup(name)
	if !ok {
		panic(fmt.Sprintf("tool %q is not present in the catalog", name))
	}
	return definition
}

func cloneDefinition(definition Definition) Definition {
	definition.Annotations.DestructiveHint = cloneBool(definition.Annotations.DestructiveHint)
	definition.Annotations.OpenWorldHint = cloneBool(definition.Annotations.OpenWorldHint)
	return definition
}

func cloneBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
