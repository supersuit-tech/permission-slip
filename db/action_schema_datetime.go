package db

import (
	"encoding/json"
	"fmt"
)

// DateTimeFieldInfo describes a schema property that accepts temporal values.
type DateTimeFieldInfo struct {
	Format string // "date-time" or "date"
	Role   string // optional x-ui datetime_range_role: lower | upper
}

// ParseActionSchemaDateTimeFields returns date/date-time properties from an
// action parameters JSON Schema.
func ParseActionSchemaDateTimeFields(schemaJSON []byte) (map[string]DateTimeFieldInfo, error) {
	if len(schemaJSON) == 0 {
		return nil, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(schemaJSON, &root); err != nil {
		return nil, fmt.Errorf("parse action schema: %w", err)
	}

	out := make(map[string]DateTimeFieldInfo)
	if props := root["properties"]; len(props) > 0 {
		if err := mergeDateTimeFieldInfo(props, out); err != nil {
			return nil, err
		}
	}
	if anyOf := root["anyOf"]; len(anyOf) > 0 {
		var branches []map[string]json.RawMessage
		if err := json.Unmarshal(anyOf, &branches); err != nil {
			return nil, fmt.Errorf("parse action schema anyOf: %w", err)
		}
		for _, branch := range branches {
			if props := branch["properties"]; len(props) > 0 {
				if err := mergeDateTimeFieldInfo(props, out); err != nil {
					return nil, err
				}
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func mergeDateTimeFieldInfo(propsJSON json.RawMessage, out map[string]DateTimeFieldInfo) error {
	var props map[string]json.RawMessage
	if err := json.Unmarshal(propsJSON, &props); err != nil {
		return err
	}
	for name, propRaw := range props {
		info, ok, err := parsePropertyDateTimeInfo(propRaw)
		if err != nil {
			return fmt.Errorf("property %q: %w", name, err)
		}
		if ok {
			out[name] = info
		}
	}
	return nil
}

func parsePropertyDateTimeInfo(propRaw json.RawMessage) (DateTimeFieldInfo, bool, error) {
	var prop map[string]json.RawMessage
	if err := json.Unmarshal(propRaw, &prop); err != nil {
		return DateTimeFieldInfo{}, false, err
	}

	var format string
	if raw, ok := prop["format"]; ok {
		_ = json.Unmarshal(raw, &format)
	}
	switch format {
	case "date-time", "date":
	default:
		return DateTimeFieldInfo{}, false, nil
	}

	info := DateTimeFieldInfo{Format: format}
	if raw, ok := prop["x-ui"]; ok {
		var xui map[string]json.RawMessage
		if err := json.Unmarshal(raw, &xui); err == nil {
			if roleRaw, ok := xui["datetime_range_role"]; ok {
				_ = json.Unmarshal(roleRaw, &info.Role)
			}
		}
	}
	return info, true, nil
}
