package connectors

import "testing"

func TestRegisterBuiltIn_And_BuiltInConnectors(t *testing.T) {
	restore := saveAndResetBuiltInConnectors()
	defer restore()

	a := newStubConnector("alpha")
	b := newStubConnector("beta")

	RegisterBuiltIn(a)
	RegisterBuiltIn(b)

	got := BuiltInConnectors()
	if len(got) != 2 {
		t.Fatalf("expected 2 built-in connectors, got %d", len(got))
	}
	if got[0].ID() != "alpha" || got[1].ID() != "beta" {
		t.Errorf("unexpected IDs: %s, %s", got[0].ID(), got[1].ID())
	}
}

func TestBuiltInConnectors_ReturnsCopy(t *testing.T) {
	restore := saveAndResetBuiltInConnectors()
	defer restore()

	RegisterBuiltIn(newStubConnector("original"))

	got := BuiltInConnectors()
	got[0] = newStubConnector("modified")

	// The internal slice must be unaffected.
	again := BuiltInConnectors()
	if again[0].ID() != "original" {
		t.Fatalf("BuiltInConnectors returned a reference, not a copy")
	}
}

func TestRegisterBuiltIn_PanicsOnEmptyID(t *testing.T) {
	restore := saveAndResetBuiltInConnectors()
	defer restore()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for empty ID, got none")
		}
	}()
	RegisterBuiltIn(newStubConnector(""))
}

func TestRegisterBuiltIn_PanicsOnDuplicateID(t *testing.T) {
	restore := saveAndResetBuiltInConnectors()
	defer restore()

	RegisterBuiltIn(newStubConnector("dup"))

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic for duplicate ID, got none")
		}
	}()
	RegisterBuiltIn(newStubConnector("dup"))
}

func TestBuiltInConnectors_EmptyAfterReset(t *testing.T) {
	restore := saveAndResetBuiltInConnectors()
	defer restore()

	got := BuiltInConnectors()
	if len(got) != 0 {
		t.Fatalf("expected 0 built-in connectors after reset, got %d", len(got))
	}
}
