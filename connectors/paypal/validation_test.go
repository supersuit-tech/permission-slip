package paypal

import (
	"net/url"
	"testing"
)

func TestPathSegment_EscapesSpecialChars(t *testing.T) {
	t.Parallel()
	// "|" is allowed by validatePayPalPathID but must be percent-encoded in path segments.
	raw := "CAP|9X"
	seg, err := pathSegment("id", raw)
	if err != nil {
		t.Fatal(err)
	}
	want := url.PathEscape(raw)
	if seg != want {
		t.Errorf("got %q, want %q", seg, want)
	}
	if seg == raw {
		t.Error("expected escaped segment")
	}
}

func TestPathSegment_RejectsUnsafe(t *testing.T) {
	t.Parallel()
	if _, err := pathSegment("id", "x/y"); err == nil {
		t.Error("slash in id: want error")
	}
}
