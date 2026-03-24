package coinbaseagentkit

import (
	"strings"
	"testing"
)

func TestValidateSwapQuoteTx(t *testing.T) {
	goodTo := "0x" + strings.Repeat("1", 40)
	badTo := "0x123"

	t.Run("valid", func(t *testing.T) {
		if err := validateSwapQuoteTx(goodTo, 21000, []byte{1, 2, 3}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("empty to", func(t *testing.T) {
		if err := validateSwapQuoteTx("", 21000, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("invalid to", func(t *testing.T) {
		if err := validateSwapQuoteTx(badTo, 21000, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("gas zero", func(t *testing.T) {
		if err := validateSwapQuoteTx(goodTo, 0, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("gas too high", func(t *testing.T) {
		if err := validateSwapQuoteTx(goodTo, maxSwapGasLimit+1, nil); err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("calldata too large", func(t *testing.T) {
		if err := validateSwapQuoteTx(goodTo, 100000, make([]byte, maxSwapCalldataBytes+1)); err == nil {
			t.Fatal("expected error")
		}
	})
}
