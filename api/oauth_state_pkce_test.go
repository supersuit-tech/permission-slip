package api

import "testing"

func TestSealOpenOAuthStatePKCE_RoundTrip(t *testing.T) {
	t.Parallel()
	secret := "test-oauth-state-secret-at-least-32-chars"
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	sealed, err := sealOAuthStatePKCE(secret, verifier)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if sealed == "" {
		t.Fatal("expected non-empty sealed blob")
	}
	got, err := openOAuthStatePKCE(secret, sealed)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if got != verifier {
		t.Errorf("got %q, want %q", got, verifier)
	}
}

func TestOpenOAuthStatePKCE_WrongSecret(t *testing.T) {
	t.Parallel()
	secret := "test-oauth-state-secret-at-least-32-chars"
	wrong := "different-secret-at-least-32-chars-long"
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"

	sealed, err := sealOAuthStatePKCE(secret, verifier)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	_, err = openOAuthStatePKCE(wrong, sealed)
	if err == nil {
		t.Fatal("expected error opening with wrong secret")
	}
}

func TestSealOpenOAuthStatePKCE_EmptyVerifier(t *testing.T) {
	t.Parallel()
	s, err := sealOAuthStatePKCE("any-secret-at-least-32-chars-long!!", "")
	if err != nil {
		t.Fatalf("seal empty: %v", err)
	}
	if s != "" {
		t.Errorf("expected empty sealed string, got %q", s)
	}
	got, err := openOAuthStatePKCE("any-secret-at-least-32-chars-long!!", "")
	if err != nil {
		t.Fatalf("open empty: %v", err)
	}
	if got != "" {
		t.Errorf("expected empty open, got %q", got)
	}
}
