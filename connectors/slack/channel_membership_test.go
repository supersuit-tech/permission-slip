package slack

import "testing"

func TestChannelTypesIncludePrivate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		types string
		want  bool
	}{
		{"public_channel", false},
		{"public_channel,private_channel", true},
		{"im", true},
		{"mpim", true},
		{"public_channel, im", true},
		{"", false},
	}
	for _, tt := range tests {
		got := channelTypesIncludePrivate(tt.types)
		if got != tt.want {
			t.Errorf("channelTypesIncludePrivate(%q) = %v, want %v", tt.types, got, tt.want)
		}
	}
}

func TestFilterPrivateTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"public_channel,private_channel,mpim,im", "private_channel,mpim,im"},
		{"public_channel", ""},
		{"im", "im"},
		{"private_channel,mpim", "private_channel,mpim"},
		{"public_channel, im", "im"},
		{"", ""},
	}
	for _, tt := range tests {
		got := filterPrivateTypes(tt.input)
		if got != tt.want {
			t.Errorf("filterPrivateTypes(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
