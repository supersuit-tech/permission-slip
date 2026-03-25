package microsoft

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *MicrosoftConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"microsoft.create_calendar_event": makeParamValidator[createCalendarEventParams](),
	"microsoft.create_document": makeParamValidator[createDocumentParams](),
	"microsoft.create_presentation": makeParamValidator[createPresentationParams](),
	"microsoft.create_spreadsheet": makeParamValidator[createSpreadsheetParams](),
	"microsoft.delete_drive_file": makeParamValidator[deleteDriveFileParams](),
	"microsoft.excel_append_rows": makeParamValidator[excelAppendRowsParams](),
	"microsoft.excel_list_worksheets": makeParamValidator[excelListWorksheetsParams](),
	"microsoft.excel_read_range": makeParamValidator[excelReadRangeParams](),
	"microsoft.excel_write_range": makeParamValidator[excelWriteRangeParams](),
	"microsoft.get_document": makeParamValidator[getDocumentParams](),
	"microsoft.get_drive_file": makeParamValidator[getDriveFileParams](),
	"microsoft.get_presentation": makeParamValidator[getPresentationParams](),
	"microsoft.list_channel_messages": makeParamValidator[listChannelMessagesParams](),
	"microsoft.list_channels": makeParamValidator[listChannelsParams](),
	"microsoft.list_documents": makeParamValidator[listDocumentsParams](),
	"microsoft.list_presentations": makeParamValidator[listPresentationsParams](),
	"microsoft.send_channel_message": makeParamValidator[sendChannelMessageParams](),
	"microsoft.send_email": makeParamValidator[sendEmailParams](),
	"microsoft.update_document": makeParamValidator[updateDocumentParams](),
	"microsoft.upload_drive_file": makeParamValidator[uploadDriveFileParams](),
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
