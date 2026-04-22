package slack

import (
	"bytes"
	"encoding/json"
)

// slackNullableText unmarshals JSON null as empty string. Slack sometimes
// returns "text": null on messages that use only blocks/attachments or when
// the text field is absent in the serialized form the API returns.
type slackNullableText string

func (t *slackNullableText) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = ""
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*t = slackNullableText(s)
	return nil
}

func (t slackNullableText) String() string { return string(t) }
