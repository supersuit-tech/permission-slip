package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

// injectConnectorInstanceIntoParametersSchema adds an optional connector_instance
// property (UUID enum + human-readable description) when the agent has multiple
// instances for this connector. Single-instance connectors leave the schema unchanged.
func injectConnectorInstanceIntoParametersSchema(schema json.RawMessage, instances []connectorInstanceCapability) json.RawMessage {
	if len(instances) < 2 {
		return schema
	}

	uuids := make([]string, len(instances))
	descParts := make([]string, 0, len(instances))
	for i, inst := range instances {
		uuids[i] = inst.ID
		label := inst.Display
		if label == "" {
			label = "(no display name)"
		}
		descParts = append(descParts, fmt.Sprintf("%s — %s", inst.ID, label))
	}
	description := strings.Join(descParts, "; ")

	prop := map[string]any{
		"type":        "string",
		"format":      "uuid",
		"enum":        uuids,
		"description": description,
	}

	var root map[string]any
	if len(schema) > 0 && string(schema) != "null" {
		_ = json.Unmarshal(schema, &root)
	}
	if root == nil {
		root = make(map[string]any)
	}
	if _, ok := root["type"]; !ok {
		root["type"] = "object"
	}

	props, _ := root["properties"].(map[string]any)
	if props == nil {
		props = make(map[string]any)
		root["properties"] = props
	}
	props["connector_instance"] = prop

	out, err := json.Marshal(root)
	if err != nil {
		return schema
	}
	return out
}
