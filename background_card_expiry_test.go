package main

import (
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
