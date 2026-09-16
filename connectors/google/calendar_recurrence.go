package google

import (
	"fmt"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func validateRecurrence(lines []string) error {
	if len(lines) == 0 {
		return &connectors.ValidationError{Message: "recurrence must include at least one RRULE:... string"}
	}
	hasRRULE := false
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("recurrence[%d] is empty", i)}
		}
		switch {
		case strings.HasPrefix(line, "RRULE:"):
			hasRRULE = true
		case strings.HasPrefix(line, "EXDATE:"), strings.HasPrefix(line, "RDATE:"):
			// Google Calendar API recurrence array members.
		default:
			return &connectors.ValidationError{
				Message: fmt.Sprintf("recurrence[%d] must start with RRULE:, EXDATE:, or RDATE:", i),
			}
		}
	}
	if !hasRRULE {
		return &connectors.ValidationError{Message: "recurrence must include at least one RRULE:... string"}
	}
	return nil
}

func formatRecurrenceUntil(cutoff time.Time, allDay bool) string {
	if allDay {
		// UNTIL is inclusive; use the day before so this instance is excluded.
		return cutoff.AddDate(0, 0, -1).Format("20060102")
	}
	return cutoff.Add(-time.Second).UTC().Format("20060102T150405Z")
}

// endRecurrenceBefore rewrites RRULE lines so the series stops just before
// cutoff (the original start of the first instance that should move to a
// new series). COUNT is dropped in favor of UNTIL. EXDATE/RDATE lines are
// kept as-is.
func endRecurrenceBefore(recurrence []string, cutoff time.Time, allDay bool) ([]string, error) {
	if len(recurrence) == 0 {
		return nil, &connectors.ValidationError{
			Message: "cannot split a series that has no recurrence rule",
		}
	}
	until := formatRecurrenceUntil(cutoff, allDay)
	out := make([]string, 0, len(recurrence))
	hasRRULE := false
	for _, line := range recurrence {
		if !strings.HasPrefix(line, "RRULE:") {
			out = append(out, line)
			continue
		}
		hasRRULE = true
		rewritten, err := rewriteRRULEUntil(line, until)
		if err != nil {
			return nil, err
		}
		out = append(out, rewritten)
	}
	if !hasRRULE {
		return nil, &connectors.ValidationError{
			Message: "cannot split a series that has no RRULE",
		}
	}
	return out, nil
}

func rewriteRRULEUntil(line, until string) (string, error) {
	body := strings.TrimPrefix(line, "RRULE:")
	if body == "" {
		return "", &connectors.ValidationError{Message: "RRULE is empty"}
	}
	parts := strings.Split(body, ";")
	kept := make([]string, 0, len(parts)+1)
	for _, part := range parts {
		if part == "" {
			continue
		}
		upper := strings.ToUpper(part)
		if strings.HasPrefix(upper, "UNTIL=") || strings.HasPrefix(upper, "COUNT=") {
			continue
		}
		kept = append(kept, part)
	}
	kept = append(kept, "UNTIL="+until)
	return "RRULE:" + strings.Join(kept, ";"), nil
}

func setRecurrenceCount(recurrence []string, count int) []string {
	if count <= 0 {
		return recurrence
	}
	out := make([]string, 0, len(recurrence))
	for _, line := range recurrence {
		if !strings.HasPrefix(line, "RRULE:") {
			out = append(out, line)
			continue
		}
		body := strings.TrimPrefix(line, "RRULE:")
		parts := strings.Split(body, ";")
		kept := make([]string, 0, len(parts)+1)
		replaced := false
		for _, part := range parts {
			if part == "" {
				continue
			}
			if strings.HasPrefix(strings.ToUpper(part), "COUNT=") {
				kept = append(kept, fmt.Sprintf("COUNT=%d", count))
				replaced = true
				continue
			}
			kept = append(kept, part)
		}
		if !replaced {
			kept = append(kept, fmt.Sprintf("COUNT=%d", count))
		}
		out = append(out, "RRULE:"+strings.Join(kept, ";"))
	}
	return out
}
