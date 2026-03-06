package main

import (
	"log/slog"
	"testing"
	"time"
)

func TestIsCardExpired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		expMonth int
		expYear  int
		now      time.Time
		want     bool
	}{
		{
			name:     "expired last year",
			expMonth: 12,
			expYear:  2024,
			now:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			want:     true,
		},
		{
			name:     "expired earlier this year",
			expMonth: 1,
			expYear:  2025,
			now:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			want:     true,
		},
		{
			name:     "expires this month - not yet expired",
			expMonth: 3,
			expYear:  2025,
			now:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			want:     false,
		},
		{
			name:     "expires next month",
			expMonth: 4,
			expYear:  2025,
			now:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			want:     false,
		},
		{
			name:     "expires next year",
			expMonth: 1,
			expYear:  2026,
			now:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isCardExpired(tt.expMonth, tt.expYear, tt.now)
			if got != tt.want {
				t.Errorf("isCardExpired(%d, %d, %v) = %v, want %v",
					tt.expMonth, tt.expYear, tt.now, got, tt.want)
			}
		})
	}
}

func TestCardExpiryCheckInterval(t *testing.T) {
	// Not parallel: subtests use t.Setenv which modifies process env.
	logger := slog.Default()

	tests := []struct {
		name     string
		envVal   string
		expected time.Duration
	}{
		{
			name:     "default when unset",
			envVal:   "",
			expected: 24 * time.Hour,
		},
		{
			name:     "valid custom interval",
			envVal:   "6h",
			expected: 6 * time.Hour,
		},
		{
			name:     "below minimum falls back to default",
			envVal:   "30m",
			expected: 24 * time.Hour,
		},
		{
			name:     "invalid format falls back to default",
			envVal:   "notaduration",
			expected: 24 * time.Hour,
		},
		{
			name:     "exactly 1 hour is accepted",
			envVal:   "1h",
			expected: time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				t.Setenv("CARD_EXPIRY_CHECK_INTERVAL", tt.envVal)
			}
			got := cardExpiryCheckInterval(logger)
			if got != tt.expected {
				t.Errorf("cardExpiryCheckInterval() = %v, want %v", got, tt.expected)
			}
		})
	}
}
