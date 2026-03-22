package microsoft

import "testing"

func TestGraphRelativePathFromNextLink(t *testing.T) {
	t.Parallel()
	base := "https://graph.microsoft.com/v1.0"
	next := "https://graph.microsoft.com/v1.0/me/calendars?$skiptoken=abc"
	got, err := graphRelativePathFromNextLink(base, next)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "/me/calendars?$skiptoken=abc" {
		t.Errorf("got %q", got)
	}
}

func TestGraphRelativePathFromNextLink_WrongHost(t *testing.T) {
	t.Parallel()
	_, err := graphRelativePathFromNextLink("https://graph.microsoft.com/v1.0", "https://evil.example/me/calendars")
	if err == nil {
		t.Fatal("expected error for mismatched host")
	}
}
