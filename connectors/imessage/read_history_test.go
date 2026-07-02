package imessage

import "testing"

func TestFilterMessagesSince_RowID(t *testing.T) {
	t.Parallel()
	messages := []message{
		{ID: 1, GUID: "a"},
		{ID: 2, GUID: "b"},
		{ID: 3, GUID: "c"},
	}
	got, miss := filterMessagesSince(messages, "", 2)
	if miss {
		t.Fatal("unexpected cursor miss")
	}
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
	got, miss := filterMessagesSince(messages, "seen", 0)
	if miss {
		t.Fatal("unexpected cursor miss")
	}
	if len(got) != 1 || got[0].GUID != "new" {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterMessagesSince_NoCursor(t *testing.T) {
	t.Parallel()
	messages := []message{{ID: 1}}
	got, miss := filterMessagesSince(messages, "", 0)
	if miss {
		t.Fatal("unexpected cursor miss")
	}
	if len(got) != 1 {
		t.Fatalf("got %#v", got)
	}
}

func TestFilterMessagesSince_GUIDNotInWindow(t *testing.T) {
	t.Parallel()
	messages := []message{{ID: 10, GUID: "a"}, {ID: 11, GUID: "b"}}
	got, miss := filterMessagesSince(messages, "missing", 0)
	if !miss {
		t.Fatal("expected cursor miss")
	}
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}
