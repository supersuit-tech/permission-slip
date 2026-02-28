package api

// normalizeConfirmationCode strips hyphens and uppercases for comparison.
// This allows agents to submit codes in any format: "XK7-M9P", "XK7M9P", "xk7-m9p".
func normalizeConfirmationCode(code string) string {
	var out []byte
	for i := 0; i < len(code); i++ {
		c := code[i]
		if c == '-' {
			continue
		}
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		out = append(out, c)
	}
	return string(out)
}
