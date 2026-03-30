package zoom

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *ZoomConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"zoom.add_registrant": makeParamValidator[addRegistrantParams](),
	"zoom.create_meeting": makeParamValidator[createMeetingParams](),
	"zoom.delete_meeting": makeParamValidator[deleteMeetingParams](),
	"zoom.get_meeting": makeParamValidator[getMeetingParams](),
	"zoom.get_meeting_participants": makeParamValidator[getMeetingParticipantsParams](),
	"zoom.get_recording_transcript": makeParamValidator[getRecordingTranscriptParams](),
	"zoom.list_meetings": makeParamValidator[listMeetingsParams](),
	"zoom.list_recordings": makeParamValidator[listRecordingsParams](),
	"zoom.send_chat_message": makeParamValidator[sendChatMessageParams](),
	"zoom.update_meeting": makeParamValidator[updateMeetingParams](),
}

func makeParamValidator[T any, PT interface {
	*T
	validate() error
}]() connectors.ParamValidatorFunc {
	return func(params json.RawMessage) error {
		p := PT(new(T))
		if err := json.Unmarshal(params, p); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
		return p.validate()
	}
}
