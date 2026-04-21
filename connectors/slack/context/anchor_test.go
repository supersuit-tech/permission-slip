package context

import "testing"

func TestSliceAroundAnchor(t *testing.T) {
	t.Parallel()
	msgs := []slackMessage{
		{TS: "1.0", Text: "a"},
		{TS: "2.0", Text: "b"},
		{TS: "3.0", Text: "c"},
		{TS: "4.0", Text: "d"},
		{TS: "5.0", Text: "e"},
	}
	got := sliceAroundAnchor(msgs, "3.0", 3)
	if len(got) != 3 || got[0].TS != "2.0" || got[2].TS != "4.0" {
		t.Fatalf("got %#v", got)
	}
}
