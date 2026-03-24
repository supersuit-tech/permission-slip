package connectors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestIsTimeout_DeadlineExceeded(t *testing.T) {
	t.Parallel()
	if !IsTimeout(context.DeadlineExceeded) {
		t.Error("expected true for context.DeadlineExceeded")
	}
}

func TestIsTimeout_WrappedDeadlineExceeded(t *testing.T) {
	t.Parallel()
	wrapped := fmt.Errorf("outer: %w", context.DeadlineExceeded)
	if !IsTimeout(wrapped) {
		t.Error("expected true for wrapped context.DeadlineExceeded")
	}
}

func TestIsTimeout_NetTimeout(t *testing.T) {
	t.Parallel()
	err := &net.OpError{Op: "read", Err: &timeoutErr{}}
	if !IsTimeout(err) {
		t.Error("expected true for net.Error with Timeout()")
	}
}

func TestIsTimeout_NonTimeout(t *testing.T) {
	t.Parallel()
	if IsTimeout(errors.New("some error")) {
		t.Error("expected false for generic error")
	}
}

func TestIsTimeout_Nil(t *testing.T) {
	t.Parallel()
	if IsTimeout(nil) {
		t.Error("expected false for nil")
	}
}

// timeoutErr implements net.Error with Timeout() returning true.
type timeoutErr struct{}

func (e *timeoutErr) Error() string   { return "timeout" }
func (e *timeoutErr) Timeout() bool   { return true }
func (e *timeoutErr) Temporary() bool { return false }

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		val      string
		fallback time.Duration
		want     time.Duration
	}{
		{"60", 0, 60 * time.Second},
		{"30", 0, 30 * time.Second},
		{"1", 0, 1 * time.Second},
		{"", 0, 0},
		{"invalid", 0, 0},
		{"", 30 * time.Second, 30 * time.Second},
		{"invalid", 30 * time.Second, 30 * time.Second},
		{"5", 30 * time.Second, 5 * time.Second},
	}

	for _, tt := range tests {
		got := ParseRetryAfter(tt.val, tt.fallback)
		if got != tt.want {
			t.Errorf("ParseRetryAfter(%q, %v) = %v, want %v", tt.val, tt.fallback, got, tt.want)
		}
	}
}

func TestJSONResult(t *testing.T) {
	t.Parallel()

	result, err := JSONResult(map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Data) != `{"key":"value"}` {
		t.Errorf("Data = %s, want %s", result.Data, `{"key":"value"}`)
	}
}

func TestJSONResult_MarshalError(t *testing.T) {
	t.Parallel()

	// Channels cannot be marshaled to JSON.
	_, err := JSONResult(make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable value")
	}
}

func TestTruncateUTF8_Short(t *testing.T) {
	t.Parallel()
	got := TruncateUTF8("hello", 10)
	if got != "hello" {
		t.Errorf("TruncateUTF8(short) = %q, want %q", got, "hello")
	}
}

func TestTruncateUTF8_ExactLength(t *testing.T) {
	t.Parallel()
	got := TruncateUTF8("hello", 5)
	if got != "hello" {
		t.Errorf("TruncateUTF8(exact) = %q, want %q", got, "hello")
	}
}

func TestTruncateUTF8_Truncated(t *testing.T) {
	t.Parallel()
	got := TruncateUTF8("hello world", 5)
	want := "hello...(truncated)"
	if got != want {
		t.Errorf("TruncateUTF8(long) = %q, want %q", got, want)
	}
}

func TestTruncateUTF8_MultibyteSafe(t *testing.T) {
	t.Parallel()
	// "日本語テスト" = 6 runes, each 3 bytes = 18 bytes
	input := "日本語テスト"
	got := TruncateUTF8(input, 3)
	want := "日本語...(truncated)"
	if got != want {
		t.Errorf("TruncateUTF8(multibyte) = %q, want %q", got, want)
	}
}

func TestTruncateUTF8_Emoji(t *testing.T) {
	t.Parallel()
	input := "😀😁😂🤣😃"
	got := TruncateUTF8(input, 2)
	want := "😀😁...(truncated)"
	if got != want {
		t.Errorf("TruncateUTF8(emoji) = %q, want %q", got, want)
	}
}

func TestRuneLen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"日本語", 3},
		{"😀😁", 2},
		{"", 0},
		{"abc日本", 5},
	}
	for _, tt := range tests {
		got := RuneLen(tt.input)
		if got != tt.want {
			t.Errorf("RuneLen(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestTrimIndent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "removes common tab indentation",
			in:   "\t\t{\"a\": 1}\n\t\t{\"b\": 2}",
			want: "{\"a\": 1}\n{\"b\": 2}",
		},
		{
			name: "preserves relative indentation",
			in:   "\t\t{\n\t\t\t\"key\": \"val\"\n\t\t}",
			want: "{\n\t\"key\": \"val\"\n}",
		},
		{
			name: "skips empty lines when computing indent",
			in:   "\t\tline1\n\n\t\tline2",
			want: "line1\n\nline2",
		},
		{
			name: "trims surrounding whitespace",
			in:   "\n\t\thello\n\t\tworld\n",
			want: "hello\nworld",
		},
		{
			name: "no indentation",
			in:   "no tabs here",
			want: "no tabs here",
		},
		{
			name: "single line with tabs",
			in:   "\t\t\tsingle",
			want: "single",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
		{
			name: "only whitespace",
			in:   "\t\t\n\t\t",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := TrimIndent(tt.in)
			if got != tt.want {
				t.Errorf("TrimIndent() =\n%q\nwant:\n%q", got, tt.want)
			}
		})
	}
}
