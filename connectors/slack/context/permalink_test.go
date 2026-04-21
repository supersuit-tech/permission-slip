package context

import "testing"

func TestBuildPermalink(t *testing.T) {
	t.Parallel()
	got := BuildPermalink("acme", "C123", "1711234567.000100")
	want := "https://acme.slack.com/archives/C123/p1711234567000100"
	if got != want {
		t.Fatalf("BuildPermalink = %q, want %q", got, want)
	}
	if BuildPermalink("", "C1", "1.2") != "" {
		t.Fatal("expected empty for missing domain")
	}
}

func TestBuildChannelPermalink(t *testing.T) {
	t.Parallel()
	got := BuildChannelPermalink("acme", "C123")
	want := "https://acme.slack.com/archives/C123"
	if got != want {
		t.Fatalf("BuildChannelPermalink = %q, want %q", got, want)
	}
}
