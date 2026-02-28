package api

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
)

// MaxRequestBodySize is the maximum allowed request body size (1 MB).
const MaxRequestBodySize = 1 << 20 // 1 MB

// DecodeJSON reads and decodes a JSON request body into dst.
// If Content-Type is omitted, JSON is assumed (this is a JSON-only API).
// If Content-Type is present but not application/json, it is rejected.
func DecodeJSON(r *http.Request, dst any) error {
	if ct := r.Header.Get("Content-Type"); ct != "" {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err != nil || mediaType != "application/json" {
			return errContentType
		}
	}

	body := http.MaxBytesReader(nil, r.Body, MaxRequestBodySize)
	defer body.Close()

	dec := json.NewDecoder(body)

	if err := dec.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return errBodyTooLarge
		}
		return errMalformedJSON
	}

	// Ensure there's no trailing content after the JSON value.
	// dec.More() only works inside arrays/objects; to detect extra top-level
	// values like `{"a":1}{"b":2}`, attempt a second decode and expect EOF.
	if err := dec.Decode(&json.RawMessage{}); err != io.EOF {
		return errMalformedJSON
	}

	return nil
}

// DecodeJSONOrReject is like DecodeJSON but writes an error response and returns false
// if decoding fails. Returns true on success.
func DecodeJSONOrReject(w http.ResponseWriter, r *http.Request, dst any) bool {
	err := DecodeJSON(r, dst)
	if err == nil {
		return true
	}

	switch err {
	case errContentType:
		RespondError(w, r, http.StatusUnsupportedMediaType, BadRequest(ErrInvalidRequest, "Content-Type must be application/json"))
	case errBodyTooLarge:
		RespondError(w, r, http.StatusRequestEntityTooLarge, BadRequest(ErrInvalidRequest, "Request body too large"))
	default:
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "Malformed or invalid JSON"))
	}
	return false
}

// RespondJSON encodes v as JSON and writes it with the given status code.
func RespondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("RespondJSON: failed to encode response: %v", err)
	}
}

// ValidateJSONObject checks that data is a valid JSON object. It returns nil
// if data is empty/null (callers should handle "required" checks separately),
// errNotJSON if the bytes aren't valid JSON, and errNotJSONObject if the value
// is valid JSON but not an object (e.g. array, string, number).
func ValidateJSONObject(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return errNotJSON
	}
	if raw == nil {
		return nil // explicit JSON null treated as absent
	}
	if _, ok := raw.(map[string]any); !ok {
		return errNotJSONObject
	}
	return nil
}

// isRawJSONNull returns true if data is the literal JSON value "null".
func isRawJSONNull(data json.RawMessage) bool {
	return len(data) == 4 && data[0] == 'n' && data[1] == 'u' && data[2] == 'l' && data[3] == 'l'
}

// Sentinel errors for DecodeJSON and ValidateJSONObject.
var (
	errContentType   = errors.New("content-type must be application/json")
	errBodyTooLarge  = errors.New("request body too large")
	errMalformedJSON = errors.New("malformed or invalid JSON")
	errNotJSON       = errors.New("not valid JSON")
	errNotJSONObject = errors.New("not a JSON object")
)
