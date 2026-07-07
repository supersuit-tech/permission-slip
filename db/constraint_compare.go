package db

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

func isComparisonOp(op string) bool {
	switch op {
	case OpLte, OpGte, OpLt, OpGt:
		return true
	default:
		return false
	}
}

func validateComparisonThreshold(field string, val json.RawMessage) error {
	if IsWildcard(val) {
		return fmt.Errorf("comparison threshold cannot be a wildcard")
	}
	if _, ok := extractPattern(val); ok {
		return fmt.Errorf("comparison threshold cannot be a pattern")
	}
	if _, _, err := parseComparableValue(val); err != nil {
		return fmt.Errorf("comparison threshold must be a number or RFC3339 datetime: %w", err)
	}
	return nil
}

func evaluateComparisonConstraint(param, op string, threshold, sourceValue json.RawMessage) error {
	thresholdKind, thresholdNum, thresholdTime, _, err := parseComparableValue(threshold)
	if err != nil {
		return fmt.Errorf("invalid comparison threshold for %s: %w", param, err)
	}

	sourceKind, sourceNum, sourceTime, _, err := parseComparableValue(sourceValue)
	if err != nil {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "value must be a number or RFC3339 datetime for comparison",
		}
	}

	if sourceKind != thresholdKind {
		return &ConfigValidationError{
			Parameter: param,
			Reason:    "value type does not match comparison threshold type",
		}
	}

	switch thresholdKind {
	case comparableNumber:
		if !compareNumbers(sourceNum, thresholdNum, op) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("value does not satisfy %s constraint", op),
			}
		}
	case comparableDatetime:
		if !compareTimes(sourceTime, thresholdTime, op) {
			return &ConfigValidationError{
				Parameter: param,
				Reason:    fmt.Sprintf("value does not satisfy %s constraint", op),
			}
		}
	}
	return nil
}

type comparableKind int

const (
	comparableNumber comparableKind = iota
	comparableDatetime
)

func parseComparableValue(raw json.RawMessage) (comparableKind, float64, time.Time, string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, 0, time.Time{}, "", fmt.Errorf("value is empty")
	}

	var num float64
	if err := json.Unmarshal(raw, &num); err == nil {
		if math.IsNaN(num) || math.IsInf(num, 0) {
			return 0, 0, time.Time{}, "", fmt.Errorf("invalid number")
		}
		return comparableNumber, num, time.Time{}, "", nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, 0, time.Time{}, "", fmt.Errorf("value must be a number or string")
	}
	if strings.TrimSpace(s) == "" {
		return 0, 0, time.Time{}, "", fmt.Errorf("value is empty")
	}

	if parsedNum, err := strconv.ParseFloat(s, 64); err == nil {
		if math.IsNaN(parsedNum) || math.IsInf(parsedNum, 0) {
			return 0, 0, time.Time{}, "", fmt.Errorf("invalid number")
		}
		return comparableNumber, parsedNum, time.Time{}, s, nil
	}

	t, err := parseRFC3339Timestamp(s)
	if err != nil {
		return 0, 0, time.Time{}, "", fmt.Errorf("not a valid number or RFC3339 datetime")
	}
	return comparableDatetime, 0, t, s, nil
}

func parseRFC3339Timestamp(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

func compareNumbers(source, threshold float64, op string) bool {
	switch op {
	case OpLte:
		return source <= threshold
	case OpGte:
		return source >= threshold
	case OpLt:
		return source < threshold
	case OpGt:
		return source > threshold
	default:
		return false
	}
}

func compareTimes(source, threshold time.Time, op string) bool {
	switch op {
	case OpLte:
		return !source.After(threshold)
	case OpGte:
		return !source.Before(threshold)
	case OpLt:
		return source.Before(threshold)
	case OpGt:
		return source.After(threshold)
	default:
		return false
	}
}
