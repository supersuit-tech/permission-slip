package google

import (
	"net/url"
	"strings"
)

func spreadsheetURL(spreadsheetID string) string {
	return "https://docs.google.com/spreadsheets/d/" + url.PathEscape(spreadsheetID)
}

func driveFolderURL(folderID string) string {
	return "https://drive.google.com/drive/folders/" + url.PathEscape(folderID)
}

func gmailMessageURL(messageID string) string {
	return "https://mail.google.com/mail/u/0/#all/" + url.QueryEscape(messageID)
}

func chatSpaceURL(spaceName string) string {
	id := strings.TrimPrefix(spaceName, "spaces/")
	return "https://mail.google.com/chat/u/0/#chat/space/" + url.PathEscape(id)
}
