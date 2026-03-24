package slack

import (
	"strings"
	"testing"
)

func TestToSlackTimestamp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "empty", in: "", want: ""},
		{name: "numeric passthrough", in: "1678800000.000000", want: "1678800000.000000"},
		{name: "integer passthrough", in: "1678800000", want: "1678800000"},
		{name: "rfc3339", in: "2023-03-14T12:00:00Z", want: "1678795200.000000"},
		{name: "rfc3339 with fractional seconds", in: "2023-03-14T12:00:00.500Z", want: "1678795200.500000"},
		{name: "rfc3339 with offset", in: "2023-03-14T08:00:00-04:00", want: "1678795200.000000"},
		{name: "invalid", in: "not-a-time", wantErr: true},
		{name: "date only", in: "2023-03-14", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := toSlackTimestamp(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Errorf("toSlackTimestamp(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Errorf("toSlackTimestamp(%q) error: %v", tc.in, err)
				return
			}
			if tc.want != "" && got != tc.want {
				t.Errorf("toSlackTimestamp(%q) = %q, want %q", tc.in, got, tc.want)
			}
			// For RFC 3339 inputs, verify output is "seconds.microseconds" format.
			if strings.Contains(tc.in, "T") && !tc.wantErr && got != "" {
				if !strings.Contains(got, ".") {
					t.Errorf("toSlackTimestamp(%q) = %q, expected dot-separated format", tc.in, got)
				}
			}
		})
	}
}
