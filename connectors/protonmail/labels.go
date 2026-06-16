package protonmail

import (
	"fmt"
	"strings"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	labelMailboxPrefix  = "Labels/"
	folderMailboxPrefix = "Folders/"
)

// resolveLabelMailbox maps a short label name or full IMAP path to the label
// mailbox Bridge/hydroxide expose under the Labels/ namespace.
func resolveLabelMailbox(label string) string {
	label = strings.TrimSpace(label)
	if strings.HasPrefix(label, labelMailboxPrefix) {
		return label
	}
	return labelMailboxPrefix + label
}

func isLabelMailbox(name string) bool {
	return strings.HasPrefix(name, labelMailboxPrefix)
}

func validateLabelParam(label string) (string, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return "", &connectors.ValidationError{Message: "missing required parameter: label"}
	}
	if strings.HasPrefix(label, folderMailboxPrefix) {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("%q is a folder path — use protonmail.move_to_folder to move messages between folders", label)}
	}
	resolved := resolveLabelMailbox(label)
	if !isLabelMailbox(resolved) {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("%q is not a Proton label mailbox — labels must live under the Labels/ namespace", label)}
	}
	return resolved, nil
}

func labelDisplayName(mailbox string) string {
	if strings.HasPrefix(mailbox, labelMailboxPrefix) {
		return strings.TrimPrefix(mailbox, labelMailboxPrefix)
	}
	return mailbox
}

func searchUIDsByMessageID(session *imapSession, messageID string) ([]imap.UID, error) {
	if messageID == "" {
		return nil, nil
	}
	criteria := &imap.SearchCriteria{
		Header: []imap.SearchCriteriaHeaderField{{
			Key:   "Message-ID",
			Value: messageID,
		}},
	}
	searchData, err := session.client.UIDSearch(criteria, nil).Wait()
	if err != nil {
		return nil, mapIMAPError(err)
	}
	return searchData.AllUIDs(), nil
}

func findLabelMailboxUIDs(session *imapSession, summaries []emailSummary) ([]uint32, error) {
	seenMessageIDs := make(map[string]struct{})
	var labelUIDs []uint32

	for _, summary := range summaries {
		messageID := summary.MessageIDHeader
		if messageID == "" {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("message UID %d in source folder has no Message-ID header — cannot locate it in the label mailbox", summary.UID),
			}
		}
		if _, ok := seenMessageIDs[messageID]; ok {
			continue
		}
		seenMessageIDs[messageID] = struct{}{}

		matches, err := searchUIDsByMessageID(session, messageID)
		if err != nil {
			return nil, err
		}
		if len(matches) == 0 {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("message with Message-ID %q is not present in the label mailbox", messageID),
			}
		}
		for _, uid := range matches {
			labelUIDs = append(labelUIDs, uint32(uid))
		}
	}
	return deduplicateUint32(labelUIDs), nil
}

func mapLabelMailboxError(err error, labelMailbox string) error {
	if err == nil {
		return nil
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "TRYCREATE") || strings.Contains(errMsg, "Mailbox doesn't exist") {
		return &connectors.ExternalError{
			Message: fmt.Sprintf("label mailbox %q not found on server — ensure the label exists (call protonmail.list_labels to discover valid names): %v", labelMailbox, err),
		}
	}
	return mapIMAPError(err)
}
