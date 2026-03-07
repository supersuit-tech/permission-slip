package google

// presentationURL returns the Google Slides editor URL for a given
// presentation ID. Used by create_presentation and get_presentation
// to include a clickable link in their responses.
func presentationURL(presentationID string) string {
	return "https://docs.google.com/presentation/d/" + presentationID + "/edit"
}
