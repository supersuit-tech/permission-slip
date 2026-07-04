package connectors

// DataWindowParams names the connector action parameters used as the
// inclusive/exclusive datetime window pair for $data_window enforcement.
type DataWindowParams struct {
	StartParam string `json:"start_param"`
	EndParam   string `json:"end_param"`
}
