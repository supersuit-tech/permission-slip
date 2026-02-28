package api

import (
	"net/http"
	"strconv"
)

// parsePaginationLimit extracts and validates the "limit" query parameter.
// Returns the parsed limit (default 50, max 100) and true on success.
// On invalid input, writes a 400 response and returns 0, false.
func parsePaginationLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be a positive integer"))
			return 0, false
		}
		if n > 100 {
			RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, "limit must be between 1 and 100"))
			return 0, false
		}
		limit = n
	}
	return limit, true
}
