package connectors

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// IsTimeout reports whether err represents a timeout — either a context
// deadline exceeded or a net.Error with Timeout() == true.
func IsTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

// ParseRetryAfter parses a Retry-After header value (seconds) into a
// time.Duration. Returns fallback if the value is empty or unparseable.
func ParseRetryAfter(val string, fallback time.Duration) time.Duration {
	if val == "" {
		return fallback
	}
	secs, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return time.Duration(secs) * time.Second
}

// JSONResult marshals v into an ActionResult. Shared by all connector
// actions to avoid repeating the marshal-and-wrap boilerplate.
func JSONResult(v any) (*ActionResult, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshaling result: %w", err)
	}
	return &ActionResult{Data: data}, nil
}

// TrimIndent removes common leading tab indentation from a multi-line string
// and trims surrounding whitespace. This lets inline JSON schema literals in Go
// source use natural indentation without embedding extra tabs in the output.
func TrimIndent(s string) string {
	lines := strings.Split(s, "\n")
	minTabs := len(s)
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, "\t")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if indent < minTabs {
			minTabs = indent
		}
	}
	for i, line := range lines {
		if len(line) >= minTabs {
			lines[i] = line[minTabs:]
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
