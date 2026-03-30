package expedia

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestValidateDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid date", value: "2024-06-15", wantErr: false},
		{name: "valid leap day", value: "2024-02-29", wantErr: false},
		{name: "invalid month", value: "2024-13-01", wantErr: true},
		{name: "invalid day", value: "2024-06-32", wantErr: true},
		{name: "wrong format", value: "06/15/2024", wantErr: true},
		{name: "partial date", value: "2024-06", wantErr: true},
		{name: "not a date", value: "tomorrow", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDate("checkin", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDate(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("validateDate() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestValidateDateRange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		checkin  string
		checkout string
		wantErr  bool
	}{
		{name: "valid range", checkin: "2024-06-15", checkout: "2024-06-17", wantErr: false},
		{name: "same day", checkin: "2024-06-15", checkout: "2024-06-15", wantErr: true},
		{name: "checkout before checkin", checkin: "2024-06-17", checkout: "2024-06-15", wantErr: true},
		{name: "invalid checkin skipped", checkin: "bad", checkout: "2024-06-15", wantErr: false},
		{name: "invalid checkout skipped", checkin: "2024-06-15", checkout: "bad", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDateRange(tt.checkin, tt.checkout)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateDateRange(%q, %q) error = %v, wantErr %v", tt.checkin, tt.checkout, err, tt.wantErr)
			}
		})
	}
}

func TestValidateOccupancy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "adults only", value: "2", wantErr: false},
		{name: "adults and child", value: "2-0,4", wantErr: false},
		{name: "adults and multiple children", value: "2-0,4,7", wantErr: false},
		{name: "single adult", value: "1", wantErr: false},
		{name: "empty string", value: "", wantErr: true},
		{name: "text", value: "two adults", wantErr: true},
		{name: "just hyphen", value: "-", wantErr: true},
		{name: "trailing comma", value: "2-0,", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOccupancy(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOccupancy(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestValidateEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "valid email", value: "john@example.com", wantErr: false},
		{name: "simple email", value: "a@b", wantErr: false},
		{name: "missing @", value: "john.example.com", wantErr: true},
		{name: "@ at start", value: "@example.com", wantErr: true},
		{name: "@ at end", value: "john@", wantErr: true},
		{name: "empty string", value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
		})
	}
}
