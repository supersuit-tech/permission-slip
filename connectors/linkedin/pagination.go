package linkedin

import (
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchPaging is the LinkedIn paging envelope returned by list and search
// endpoints. It is embedded in response structs across multiple actions.
type searchPaging struct {
	Total int `json:"total"`
	Start int `json:"start"`
	Count int `json:"count"`
}

// defaultSearchCount is the default page size for search endpoints (people,
// companies). List endpoints (connections) use their own default.
const defaultSearchCount = 10

// maxSearchCount is the maximum page size for search endpoints (people,
// companies). List endpoints (connections) use their own maximum.
const maxSearchCount = 50

// nextStart returns the start offset for the next page of results. When the
// caller receives fewer results than requested, they have reached the end of
// the result set and next_start will equal total (or the current position).
// Including next_start in every paginated response removes the need for callers
// to compute it manually.
func nextStart(start, returned int) int {
	return start + returned
}

// validateCountStart validates the count and start pagination parameters
// shared by search and list actions.
func validateCountStart(count, maxCount, start int) error {
	if count < 0 {
		return &connectors.ValidationError{Message: "count must be non-negative"}
	}
	if count > maxCount {
		return &connectors.ValidationError{Message: fmt.Sprintf("count must not exceed %d", maxCount)}
	}
	if start < 0 {
		return &connectors.ValidationError{Message: "start must be non-negative"}
	}
	return nil
}

// resolveCount returns count if it is non-zero, otherwise defaultCount.
// This applies the convention that count=0 means "use the default page size".
func resolveCount(count, defaultCount int) int {
	if count == 0 {
		return defaultCount
	}
	return count
}
