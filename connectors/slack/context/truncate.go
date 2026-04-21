package context

const maxBodyRunes = 4096

// TruncateBody caps message text to 4 KB of UTF-8 / runes (Slack approval context).
func TruncateBody(text string) (truncated string, wasTruncated bool) {
	if text == "" {
		return "", false
	}
	runes := []rune(text)
	if len(runes) <= maxBodyRunes {
		return text, false
	}
	return string(runes[:maxBodyRunes]), true
}
