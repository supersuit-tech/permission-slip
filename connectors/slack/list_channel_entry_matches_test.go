package slack

import "testing"

func TestListChannelEntryMatchesTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		types string
		ch    listChannelEntry
		want  bool
	}{
		{
			name:  "im D prefix",
			types: "im",
			ch:    listChannelEntry{ID: "D0123"},
			want:  true,
		},
		{
			name:  "im is_im",
			types: "im",
			ch:    listChannelEntry{ID: "D0123", IsIM: true},
			want:  true,
		},
		{
			name:  "im not G",
			types: "im",
			ch:    listChannelEntry{ID: "G0123"},
			want:  false,
		},
		{
			name:  "mpim is_mpim",
			types: "mpim",
			ch:    listChannelEntry{ID: "G0123", IsMPIM: true},
			want:  true,
		},
		{
			name:  "mpim G prefix heuristic",
			types: "mpim",
			ch:    listChannelEntry{ID: "G0123"},
			want:  true,
		},
		{
			name:  "mpim G prefix heuristic excludes private channels",
			types: "mpim",
			ch:    listChannelEntry{ID: "G0123", IsPrivate: true},
			want:  false,
		},
		{
			name:  "mpim C channel",
			types: "mpim",
			ch:    listChannelEntry{ID: "C0123", IsPrivate: true},
			want:  false,
		},
		{
			name:  "private_channel C private",
			types: "private_channel",
			ch:    listChannelEntry{ID: "C0123", IsPrivate: true},
			want:  true,
		},
		{
			name:  "private_channel G legacy private",
			types: "private_channel",
			ch:    listChannelEntry{ID: "G0123", IsPrivate: true},
			want:  true,
		},
		{
			name:  "private_channel G mpim excluded",
			types: "private_channel",
			ch:    listChannelEntry{ID: "G0123", IsPrivate: true, IsMPIM: true},
			want:  false,
		},
		{
			name:  "private_channel DM excluded",
			types: "private_channel",
			ch:    listChannelEntry{ID: "D0123", IsPrivate: true, IsIM: true},
			want:  false,
		},
		{
			name:  "private_channel public C",
			types: "private_channel",
			ch:    listChannelEntry{ID: "C0123", IsPrivate: false},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := listChannelEntryMatchesTypes(tt.types, tt.ch); got != tt.want {
				t.Errorf("listChannelEntryMatchesTypes(%q, %+v) = %v, want %v", tt.types, tt.ch, got, tt.want)
			}
		})
	}
}
