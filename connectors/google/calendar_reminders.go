package google

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	maxReminderOverrides = 5
	maxReminderMinutes   = 40320 // 4 weeks; Google Calendar API limit
)

// eventRemindersParams is the user-facing reminders object (snake_case).
type eventRemindersParams struct {
	UseDefault *bool                   `json:"use_default"`
	Overrides  []eventReminderOverride `json:"overrides"`
}

type eventReminderOverride struct {
	Method  string `json:"method"`
	Minutes *int   `json:"minutes"`
}

// calendarRemindersPayload is the Google Calendar API reminders object.
type calendarRemindersPayload struct {
	UseDefault bool                       `json:"useDefault"`
	Overrides  []calendarReminderOverride `json:"overrides,omitempty"`
}

type calendarReminderOverride struct {
	Method  string `json:"method"`
	Minutes int    `json:"minutes"`
}

func hasReminderInput(reminders *eventRemindersParams, reminderMinutes []int) bool {
	return reminders != nil || len(reminderMinutes) > 0
}

// resolveEventReminders maps agent-facing reminder params onto the Calendar API
// reminders object. Returns nil when neither field is set so the calendar
// default reminders apply.
func resolveEventReminders(reminders *eventRemindersParams, reminderMinutes []int) (*calendarRemindersPayload, error) {
	hasShorthand := len(reminderMinutes) > 0
	if hasShorthand && reminders != nil {
		return nil, &connectors.ValidationError{Message: "reminders and reminder_minutes are mutually exclusive"}
	}
	if hasShorthand {
		if len(reminderMinutes) > maxReminderOverrides {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("reminder_minutes must contain at most %d values", maxReminderOverrides),
			}
		}
		overrides := make([]calendarReminderOverride, len(reminderMinutes))
		for i, minutes := range reminderMinutes {
			if err := validateReminderMinutes(minutes); err != nil {
				return nil, err
			}
			overrides[i] = calendarReminderOverride{Method: "popup", Minutes: minutes}
		}
		return &calendarRemindersPayload{UseDefault: false, Overrides: overrides}, nil
	}
	if reminders == nil {
		return nil, nil
	}
	if reminders.UseDefault == nil && len(reminders.Overrides) == 0 {
		return nil, &connectors.ValidationError{Message: "reminders must include use_default or overrides"}
	}

	useDefault := reminders.UseDefault != nil && *reminders.UseDefault
	if useDefault && len(reminders.Overrides) > 0 {
		return nil, &connectors.ValidationError{Message: "reminders.overrides cannot be set when reminders.use_default is true"}
	}
	if useDefault {
		return &calendarRemindersPayload{UseDefault: true}, nil
	}

	if len(reminders.Overrides) > maxReminderOverrides {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("reminders.overrides must contain at most %d entries", maxReminderOverrides),
		}
	}
	overrides := make([]calendarReminderOverride, 0, len(reminders.Overrides))
	for i, override := range reminders.Overrides {
		method := strings.ToLower(strings.TrimSpace(override.Method))
		if method != "popup" && method != "email" {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("reminders.overrides[%d].method must be popup or email", i),
			}
		}
		if override.Minutes == nil {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("reminders.overrides[%d].minutes is required", i),
			}
		}
		if err := validateReminderMinutes(*override.Minutes); err != nil {
			return nil, err
		}
		overrides = append(overrides, calendarReminderOverride{Method: method, Minutes: *override.Minutes})
	}
	return &calendarRemindersPayload{UseDefault: false, Overrides: overrides}, nil
}

func validateReminderMinutes(minutes int) error {
	if minutes < 0 {
		return &connectors.ValidationError{Message: "reminder minutes must be non-negative"}
	}
	if minutes > maxReminderMinutes {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("reminder minutes must be at most %d (4 weeks)", maxReminderMinutes),
		}
	}
	return nil
}
