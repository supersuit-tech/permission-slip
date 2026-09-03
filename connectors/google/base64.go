package google

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// decodeBase64Bytes decodes standard or URL-safe base64, with or without padding.
// Gmail returns unpadded base64url; agents uploading to Drive typically send
// standard padded base64 (e.g. from protonmail.download_attachment).
func decodeBase64Bytes(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty base64 payload")
	}
	encodings := []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	}
	var lastErr error
	for _, enc := range encodings {
		b, err := enc.DecodeString(s)
		if err == nil {
			return b, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		return nil, fmt.Errorf("invalid base64")
	}
	return nil, lastErr
}
