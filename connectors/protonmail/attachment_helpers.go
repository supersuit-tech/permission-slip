package protonmail

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime/quotedprintable"
	"strconv"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// formatPartPath renders an IMAP MIME part path as a stable string key (e.g. "2.1").
func formatPartPath(path []int) string {
	if len(path) == 0 {
		return ""
	}
	parts := make([]string, len(path))
	for i, p := range path {
		parts[i] = strconv.Itoa(p)
	}
	return strings.Join(parts, ".")
}

// parsePartPath parses a MIME part path string (e.g. "2.1") into IMAP part indices.
func parsePartPath(s string) ([]int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty attachment_id")
	}

	segments := strings.Split(s, ".")
	path := make([]int, len(segments))
	for i, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			return nil, fmt.Errorf("invalid attachment_id %q: empty segment", s)
		}
		n, err := strconv.Atoi(seg)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("invalid attachment_id %q: segment %q must be a positive integer", s, seg)
		}
		path[i] = n
	}
	return path, nil
}

func pathsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// findAttachmentPart locates a single-part attachment at the given MIME path.
func findAttachmentPart(bs imap.BodyStructure, path []int) (*imap.BodyStructureSinglePart, bool) {
	if bs == nil || len(path) == 0 {
		return nil, false
	}

	var found *imap.BodyStructureSinglePart
	bs.Walk(func(walkPath []int, part imap.BodyStructure) bool {
		if !pathsEqual(walkPath, path) {
			return true
		}
		sp, ok := part.(*imap.BodyStructureSinglePart)
		if !ok || sp.Filename() == "" {
			return true
		}
		found = sp
		return true
	})
	return found, found != nil
}

// decodeTransferEncoding decodes a MIME part body according to its Content-Transfer-Encoding.
func decodeTransferEncoding(raw []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "7bit", "8bit", "binary":
		return raw, nil
	case "base64", "b64":
		decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(raw))
		decoded, err := io.ReadAll(decoder)
		if err != nil {
			return nil, fmt.Errorf("base64 decode failed: %w", err)
		}
		return decoded, nil
	case "quoted-printable", "qp":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(raw)))
		if err != nil {
			return nil, fmt.Errorf("quoted-printable decode failed: %w", err)
		}
		return decoded, nil
	default:
		return nil, &connectors.ExternalError{
			Message: fmt.Sprintf("unsupported Content-Transfer-Encoding: %q", encoding),
		}
	}
}
