package api

// normalizeConfirmationCode strips hyphens and uppercases for comparison.
// This allows agents to submit codes in any format: "XK7M9-PQRST", "XK7M9PQRST", "xk7m9-pqrst".
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
