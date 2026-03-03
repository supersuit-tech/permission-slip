package notify

import "testing"

func TestSMSRegionForPhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		phone string
		want  SMSRegion
	}{
		// US/Canada
		{"+15551234567", SMSRegionUSCA},
		{"+14155551234", SMSRegionUSCA},

		// UK
		{"+447911123456", SMSRegionUKEU},

		// EU countries
		{"+33612345678", SMSRegionUKEU},   // France
		{"+491761234567", SMSRegionUKEU},   // Germany
		{"+393331234567", SMSRegionUKEU},   // Italy
		{"+34612345678", SMSRegionUKEU},    // Spain
		{"+31612345678", SMSRegionUKEU},    // Netherlands
		{"+358401234567", SMSRegionUKEU},   // Finland
		{"+353871234567", SMSRegionUKEU},   // Ireland
		{"+4201234567890", SMSRegionUKEU},  // Czech Republic

		// International
		{"+81312345678", SMSRegionInternational},   // Japan
		{"+61412345678", SMSRegionInternational},   // Australia
		{"+5511912345678", SMSRegionInternational}, // Brazil

		// Edge cases
		{"5551234567", SMSRegionInternational},   // no + prefix
		{"+", SMSRegionInternational},             // just +
		{"", SMSRegionInternational},              // empty
	}

	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			t.Parallel()
			got := SMSRegionForPhone(tt.phone)
			if got != tt.want {
				t.Errorf("SMSRegionForPhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}
