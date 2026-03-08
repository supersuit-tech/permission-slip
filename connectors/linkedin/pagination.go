package linkedin

// searchPaging is the LinkedIn paging envelope returned by list and search
// endpoints. It is embedded in response structs across multiple actions.
type searchPaging struct {
	Total int `json:"total"`
	Start int `json:"start"`
	Count int  `json:"count"`
}

// nextStart returns the start offset for the next page of results. When the
// caller receives fewer results than requested, they have reached the end of
// the result set and next_start will equal total (or the current position).
// Including next_start in every paginated response removes the need for callers
// to compute it manually.
func nextStart(start, returned int) int {
	return start + returned
}
