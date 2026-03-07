package google

import "net/url"

// Shared Docs API batchUpdate request types, used by both create_document
// (for inserting initial body) and update_document.

type docsBatchUpdateRequest struct {
	Requests []docsRequest `json:"requests"`
}

type docsRequest struct {
	InsertText *docsInsertTextRequest `json:"insertText,omitempty"`
}

type docsInsertTextRequest struct {
	Text                 string                    `json:"text"`
	Location             *docsLocation             `json:"location,omitempty"`
	EndOfSegmentLocation *docsEndOfSegmentLocation `json:"endOfSegmentLocation,omitempty"`
}

type docsLocation struct {
	Index int `json:"index"`
}

type docsEndOfSegmentLocation struct{}

// documentEditURL returns the Google Docs web editor URL for the given document ID.
func documentEditURL(documentID string) string {
	return "https://docs.google.com/document/d/" + url.PathEscape(documentID) + "/edit"
}
