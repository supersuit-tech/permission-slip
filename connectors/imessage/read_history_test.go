package imessage

import "testing"

func TestFilterMessagesSince_RowID(t *testing.T) {
	t.Parallel()
	messages := []message{
		{ID: 1, GUID: "a"},
		{ID: 2, GUID: "b"},
		{ID: 3, GUID: "c"},
	}
	got := filterMessagesSince(messages, "", 2)
	if len(got) != 1 || got[0].ID != 3 {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterMessagesSince_GUID(t *testing.T) {
	t.Parallel()
	messages := []message{
		{ID: 10, GUID: "seen"},
		{ID: 11, GUID: "new"},
	}
	got := filterMessagesSince(messages, "seen", 0)
	if len(got) != 1 || got[0].GUID != "new" {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterMessagesSince_NoCursor(t *testing.T) {
	t.Parallel()
	messages := []message{{ID: 1}}
	got := filterMessagesSince(messages, "", 0)
	if len(got) != 1 {
		t.Fatalf("got %#v", got)
	}
}
