package google

import "net/url"

// applySupportsAllDrives marks a Drive files.* request as Shared Drive capable.
// Without this flag Google 404s Shared Drive folder/file IDs as "File not found"
// even when the ID is valid.
func applySupportsAllDrives(q url.Values) {
	q.Set("supportsAllDrives", "true")
}

// applyDriveListScope adds Shared Drive list/search flags and scopes the
// corpus. When driveID is set, results are limited to that Shared Drive
// (corpora=drive). Otherwise corpora=allDrives so Shared Drive items are
// visible — the Drive default corpora=user only sees My Drive.
func applyDriveListScope(q url.Values, driveID string) {
	applySupportsAllDrives(q)
	q.Set("includeItemsFromAllDrives", "true")
	if driveID != "" {
		q.Set("corpora", "drive")
		q.Set("driveId", driveID)
		return
	}
	q.Set("corpora", "allDrives")
}
