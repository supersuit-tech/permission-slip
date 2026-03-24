package slack

import "testing"

func TestSlackChannelPickerLabel(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		ch   listChannelSummary
		want string
	}{
		{
			name: "public with name",
			ch:   listChannelSummary{ID: "C1", Name: "general", IsPrivate: false},
			want: "#general",
		},
		{
			name: "private with name",
			ch:   listChannelSummary{ID: "G1", Name: "secret", IsPrivate: true},
			want: "secret",
		},
		{
			name: "im with user fallback",
			ch:   listChannelSummary{ID: "D1", User: "U99"},
			want: "DM · U99",
		},
		{
			name: "im with display name",
			ch:   listChannelSummary{ID: "D1", Name: "alice", User: "U99"},
			want: "alice",
		},
		{
			name: "mpim without name",
			ch:   listChannelSummary{ID: "G2", IsMPIM: true},
			want: "Group DM",
		},
		{
			name: "empty name falls back to id",
			ch:   listChannelSummary{ID: "C9", IsPrivate: false},
			want: "C9",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := slackChannelPickerLabel(tc.ch); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
