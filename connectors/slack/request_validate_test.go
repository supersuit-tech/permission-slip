package slack

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// TestRequestValidator_ChannelActions verifies that all channel-based actions
// implement RequestValidator and reject invalid channel IDs at request time.
func TestRequestValidator_ChannelActions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	channelActions := []string{
		"slack.read_channel_messages",
		"slack.send_message",
		"slack.schedule_message",
		"slack.invite_to_channel",
		"slack.set_topic",
		"slack.read_thread",
		"slack.upload_file",
	}

	tests := []struct {
		name    string
		channel string
		wantErr bool
	}{
		{name: "valid C prefix", channel: "C01234567", wantErr: false},
		{name: "valid G prefix", channel: "G01234567", wantErr: false},
		{name: "valid D prefix", channel: "D01234567", wantErr: false},
		{name: "user ID instead", channel: "U0AM0RW432Q", wantErr: true},
		{name: "channel name", channel: "general", wantErr: true},
		{name: "hash channel name", channel: "#general", wantErr: true},
		{name: "empty is ok (schema validates required)", channel: "", wantErr: false},
	}

	for _, actionType := range channelActions {
		action := actions[actionType]
		rv, ok := action.(connectors.RequestValidator)
		if !ok {
			t.Errorf("%s does not implement RequestValidator", actionType)
			continue
		}

		for _, tt := range tests {
			t.Run(actionType+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				params, _ := json.Marshal(map[string]string{"channel": tt.channel})
				err := rv.ValidateRequest(params)
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T", err)
				}
			})
		}
	}
}

// TestRequestValidator_ChannelAndTSActions verifies that actions with both channel
// and timestamp parameters validate both fields at request time.
func TestRequestValidator_ChannelAndTSActions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	tsActions := []string{
		"slack.update_message",
		"slack.delete_message",
	}

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
	}{
		{
			name:    "valid channel and ts",
			params:  map[string]string{"channel": "C01234567", "ts": "1234567890.123456"},
			wantErr: false,
		},
		{
			name:    "invalid channel with valid ts",
			params:  map[string]string{"channel": "U0AM0RW432Q", "ts": "1234567890.123456"},
			wantErr: true,
		},
		{
			name:    "valid channel with invalid ts",
			params:  map[string]string{"channel": "C01234567", "ts": "not-a-timestamp"},
			wantErr: true,
		},
	}

	for _, actionType := range tsActions {
		action := actions[actionType]
		rv, ok := action.(connectors.RequestValidator)
		if !ok {
			t.Errorf("%s does not implement RequestValidator", actionType)
			continue
		}

		for _, tt := range tests {
			t.Run(actionType+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				params, _ := json.Marshal(tt.params)
				err := rv.ValidateRequest(params)
				if (err != nil) != tt.wantErr {
					t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
				}
				if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
					t.Errorf("expected ValidationError, got %T", err)
				}
			})
		}
	}
}

// TestRequestValidator_SendDM verifies that send_dm validates user_id format.
func TestRequestValidator_SendDM(t *testing.T) {
	t.Parallel()
	c := New()
	action := c.Actions()["slack.send_dm"]

	rv, ok := action.(connectors.RequestValidator)
	if !ok {
		t.Fatal("slack.send_dm does not implement RequestValidator")
	}

	tests := []struct {
		name    string
		userID  string
		wantErr bool
	}{
		{name: "valid U prefix", userID: "U01234567", wantErr: false},
		{name: "valid W prefix", userID: "W01234567", wantErr: false},
		{name: "channel ID instead", userID: "C01234567", wantErr: true},
		{name: "username", userID: "john.doe", wantErr: true},
		{name: "empty is ok (schema validates required)", userID: "", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(map[string]string{"user_id": tt.userID})
			err := rv.ValidateRequest(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

// TestRequestValidator_AddReaction verifies that add_reaction validates both channel
// and the "timestamp" parameter (not "ts") at request time.
func TestRequestValidator_AddReaction(t *testing.T) {
	t.Parallel()
	c := New()
	action := c.Actions()["slack.add_reaction"]

	rv, ok := action.(connectors.RequestValidator)
	if !ok {
		t.Fatal("slack.add_reaction does not implement RequestValidator")
	}

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
	}{
		{name: "valid channel and timestamp", params: map[string]string{"channel": "C01234567", "timestamp": "1234567890.123456"}, wantErr: false},
		{name: "invalid channel", params: map[string]string{"channel": "general", "timestamp": "1234567890.123456"}, wantErr: true},
		{name: "invalid timestamp", params: map[string]string{"channel": "C01234567", "timestamp": "not-a-timestamp"}, wantErr: true},
		{name: "empty timestamp is ok", params: map[string]string{"channel": "C01234567", "timestamp": ""}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(tt.params)
			err := rv.ValidateRequest(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

// TestRequestValidator_ReadThread verifies that read_thread validates both channel
// and thread_ts parameters at request time.
func TestRequestValidator_ReadThread(t *testing.T) {
	t.Parallel()
	c := New()
	action := c.Actions()["slack.read_thread"]

	rv, ok := action.(connectors.RequestValidator)
	if !ok {
		t.Fatal("slack.read_thread does not implement RequestValidator")
	}

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
	}{
		{name: "valid channel and thread_ts", params: map[string]string{"channel": "C01234567", "thread_ts": "1234567890.123456"}, wantErr: false},
		{name: "invalid channel", params: map[string]string{"channel": "general", "thread_ts": "1234567890.123456"}, wantErr: true},
		{name: "invalid thread_ts", params: map[string]string{"channel": "C01234567", "thread_ts": "not-a-timestamp"}, wantErr: true},
		{name: "empty thread_ts is ok", params: map[string]string{"channel": "C01234567", "thread_ts": ""}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(tt.params)
			err := rv.ValidateRequest(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

// TestRequestValidator_InviteUsers verifies that invite_to_channel validates the
// comma-separated users field in addition to channel.
func TestRequestValidator_InviteUsers(t *testing.T) {
	t.Parallel()
	c := New()
	action := c.Actions()["slack.invite_to_channel"]

	rv, ok := action.(connectors.RequestValidator)
	if !ok {
		t.Fatal("slack.invite_to_channel does not implement RequestValidator")
	}

	tests := []struct {
		name    string
		params  map[string]string
		wantErr bool
	}{
		{name: "valid channel and users", params: map[string]string{"channel": "C01234567", "users": "U111,U222"}, wantErr: false},
		{name: "valid W-prefix user", params: map[string]string{"channel": "C01234567", "users": "W01234567"}, wantErr: false},
		{name: "username instead of ID", params: map[string]string{"channel": "C01234567", "users": "john.doe"}, wantErr: true},
		{name: "mixed valid and invalid", params: map[string]string{"channel": "C01234567", "users": "U111,john.doe"}, wantErr: true},
		{name: "email instead of ID", params: map[string]string{"channel": "C01234567", "users": "john@example.com"}, wantErr: true},
		{name: "empty users is ok", params: map[string]string{"channel": "C01234567", "users": ""}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(tt.params)
			err := rv.ValidateRequest(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

// TestRequestValidator_SearchMessages verifies that search_messages validates
// parameters at request time so malformed queries are rejected before the user
// sees the approval.
func TestRequestValidator_SearchMessages(t *testing.T) {
	t.Parallel()
	c := New()
	action := c.Actions()["slack.search_messages"]

	rv, ok := action.(connectors.RequestValidator)
	if !ok {
		t.Fatal("slack.search_messages does not implement RequestValidator")
	}

	tests := []struct {
		name    string
		params  map[string]any
		wantErr bool
	}{
		{name: "valid query", params: map[string]any{"query": "hello world"}, wantErr: false},
		{name: "missing query", params: map[string]any{}, wantErr: true},
		{name: "empty query", params: map[string]any{"query": ""}, wantErr: true},
		{name: "valid count", params: map[string]any{"query": "test", "count": 50}, wantErr: false},
		{name: "count too high", params: map[string]any{"query": "test", "count": 200}, wantErr: true},
		{name: "count too low", params: map[string]any{"query": "test", "count": -1}, wantErr: true},
		{name: "valid sort score", params: map[string]any{"query": "test", "sort": "score"}, wantErr: false},
		{name: "valid sort timestamp", params: map[string]any{"query": "test", "sort": "timestamp"}, wantErr: false},
		{name: "invalid sort", params: map[string]any{"query": "test", "sort": "relevance"}, wantErr: true},
		{name: "negative page", params: map[string]any{"query": "test", "page": -1}, wantErr: true},
		{name: "valid page", params: map[string]any{"query": "test", "page": 2}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			params, _ := json.Marshal(tt.params)
			err := rv.ValidateRequest(params)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

// TestRequestValidator_NotImplemented verifies that actions without ID parameters
// do not implement RequestValidator (no unnecessary interface).
func TestRequestValidator_NotImplemented(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	noValidation := []string{
		"slack.create_channel",
		"slack.list_channels",
		"slack.list_users",
	}

	for _, actionType := range noValidation {
		action := actions[actionType]
		if _, ok := action.(connectors.RequestValidator); ok {
			t.Errorf("%s implements RequestValidator but shouldn't need to", actionType)
		}
	}
}
