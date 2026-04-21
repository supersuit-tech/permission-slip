package google

import (
	"errors"
	"strings"
)

// Sentinel errors for validateChatSpaceName — mapped to user-facing ValidationError
// messages in sendChatMessageParams.validate and to fmt.Errorf in resolveChatSpace.
var (
	errChatSpaceNotPrefixed  = errors.New("chat space: must use spaces/ prefix")
	errChatSpaceEmptyID      = errors.New("chat space: empty id after prefix")
	errChatSpaceInvalidChars = errors.New("chat space: invalid characters in id")
)

// validateChatSpaceName checks that s matches Google Chat space resource names
// (e.g. spaces/AAAAMpdlehY). Shared by send_chat_message execution and
// resolveChatSpace so URL-building rules stay in one place.
func validateChatSpaceName(s string) (spaceID string, err error) {
	if !strings.HasPrefix(s, "spaces/") {
		return "", errChatSpaceNotPrefixed
	}
	spaceID = strings.TrimPrefix(s, "spaces/")
	if spaceID == "" {
		return "", errChatSpaceEmptyID
	}
	if strings.ContainsAny(spaceID, "/?#") || strings.Contains(spaceID, "..") {
		return "", errChatSpaceInvalidChars
	}
	return spaceID, nil
}
