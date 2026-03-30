package google

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *GoogleConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"google.add_slide": makeParamValidator[addSlideParams](),
	"google.archive_email": makeParamValidator[archiveEmailParams](),
	"google.create_calendar_event": makeParamValidator[createCalendarEventParams](),
	"google.create_document": makeParamValidator[createDocumentParams](),
	"google.create_drive_folder": makeParamValidator[createDriveFolderParams](),
	"google.create_meeting": makeParamValidator[createMeetingParams](),
	"google.create_presentation": makeParamValidator[createPresentationParams](),
	"google.delete_calendar_event": makeParamValidator[deleteCalendarEventParams](),
	"google.delete_drive_file": makeParamValidator[deleteDriveFileParams](),
	"google.get_document": makeParamValidator[getDocumentParams](),
	"google.get_drive_file": makeParamValidator[getDriveFileParams](),
	"google.get_presentation": makeParamValidator[getPresentationParams](),
	"google.list_calendar_events": makeParamValidator[listCalendarEventsParams](),
	"google.list_drive_files": makeParamValidator[listDriveFilesParams](),
	"google.read_email": makeParamValidator[readEmailParams](),
	"google.search_drive": makeParamValidator[searchDriveParams](),
	"google.send_chat_message": makeParamValidator[sendChatMessageParams](),
	"google.send_email": makeParamValidator[sendEmailParams](),
	"google.send_email_reply": makeParamValidator[sendEmailReplyParams](),
	"google.sheets_append_rows": makeParamValidator[sheetsAppendRowsParams](),
	"google.sheets_list_sheets": makeParamValidator[sheetsListSheetsParams](),
	"google.sheets_read_range": makeParamValidator[sheetsReadRangeParams](),
	"google.sheets_write_range": makeParamValidator[sheetsWriteRangeParams](),
	"google.update_calendar_event": makeParamValidator[updateCalendarEventParams](),
	"google.update_document": makeParamValidator[updateDocumentParams](),
	"google.upload_drive_file": makeParamValidator[uploadDriveFileParams](),
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
