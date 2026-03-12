package amadeus

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Verify bookFlightAction implements Normalizer at compile time.
var _ connectors.Normalizer = (*bookFlightAction)(nil)

func TestBookFlightNormalize_SnakeCaseToCamelCase(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	input := json.RawMessage(`{
		"flight_offer": {"id":"1"},
		"travelers": [
			{
				"name": {"first_name": "John", "last_name": "Doe"},
				"date_of_birth": "1990-01-15",
				"gender": "MALE",
				"contact": {"email": "john@example.com", "phone": "+1234567890"}
			}
		],
		"payment_method_id": "pm_abc",
		"idempotency_key": "key-1"
	}`)

	result := action.Normalize(input)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	var travelers []map[string]json.RawMessage
	if err := json.Unmarshal(m["travelers"], &travelers); err != nil {
		t.Fatalf("unmarshal travelers: %v", err)
	}

	if len(travelers) != 1 {
		t.Fatalf("expected 1 traveler, got %d", len(travelers))
	}

	traveler := travelers[0]

	// Check dateOfBirth (top-level rewrite).
	if _, ok := traveler["dateOfBirth"]; !ok {
		t.Error("expected dateOfBirth, not found")
	}
	if _, ok := traveler["date_of_birth"]; ok {
		t.Error("date_of_birth should have been removed")
	}

	// Check name sub-object.
	var name map[string]json.RawMessage
	if err := json.Unmarshal(traveler["name"], &name); err != nil {
		t.Fatalf("unmarshal name: %v", err)
	}
	if _, ok := name["firstName"]; !ok {
		t.Error("expected firstName in name, not found")
	}
	if _, ok := name["lastName"]; !ok {
		t.Error("expected lastName in name, not found")
	}
	if _, ok := name["first_name"]; ok {
		t.Error("first_name should have been removed from name")
	}
	if _, ok := name["last_name"]; ok {
		t.Error("last_name should have been removed from name")
	}
}

func TestBookFlightNormalize_CamelCasePassthrough(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	// Already camelCase — should pass through unchanged.
	input := json.RawMessage(`{
		"flight_offer": {"id":"1"},
		"travelers": [
			{
				"name": {"firstName": "John", "lastName": "Doe"},
				"dateOfBirth": "1990-01-15",
				"gender": "MALE",
				"contact": {"email": "john@example.com", "phone": "+1234567890"}
			}
		],
		"payment_method_id": "pm_abc",
		"idempotency_key": "key-1"
	}`)

	result := action.Normalize(input)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	var travelers []map[string]json.RawMessage
	if err := json.Unmarshal(m["travelers"], &travelers); err != nil {
		t.Fatalf("unmarshal travelers: %v", err)
	}

	traveler := travelers[0]

	if _, ok := traveler["dateOfBirth"]; !ok {
		t.Error("expected dateOfBirth to remain")
	}

	var name map[string]json.RawMessage
	if err := json.Unmarshal(traveler["name"], &name); err != nil {
		t.Fatalf("unmarshal name: %v", err)
	}
	if _, ok := name["firstName"]; !ok {
		t.Error("expected firstName to remain")
	}
	if _, ok := name["lastName"]; !ok {
		t.Error("expected lastName to remain")
	}
}

func TestBookFlightNormalize_CamelCaseTakesPrecedence(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	// Both snake_case and camelCase present — camelCase takes precedence.
	input := json.RawMessage(`{
		"travelers": [
			{
				"name": {"firstName": "John", "first_name": "Jonathan", "lastName": "Doe", "last_name": "Smith"},
				"dateOfBirth": "1990-01-15",
				"date_of_birth": "2000-01-01",
				"gender": "MALE",
				"contact": {"email": "j@e.com", "phone": "+1"}
			}
		],
		"flight_offer": {"id":"1"},
		"payment_method_id": "pm_1",
		"idempotency_key": "key-1"
	}`)

	result := action.Normalize(input)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	var travelers []map[string]json.RawMessage
	if err := json.Unmarshal(m["travelers"], &travelers); err != nil {
		t.Fatalf("unmarshal travelers: %v", err)
	}

	traveler := travelers[0]

	// camelCase should win.
	var dob string
	if err := json.Unmarshal(traveler["dateOfBirth"], &dob); err != nil {
		t.Fatalf("unmarshal dateOfBirth: %v", err)
	}
	if dob != "1990-01-15" {
		t.Errorf("dateOfBirth = %q, want 1990-01-15 (camelCase should take precedence)", dob)
	}

	var name map[string]json.RawMessage
	if err := json.Unmarshal(traveler["name"], &name); err != nil {
		t.Fatalf("unmarshal name: %v", err)
	}
	var firstName string
	if err := json.Unmarshal(name["firstName"], &firstName); err != nil {
		t.Fatalf("unmarshal firstName: %v", err)
	}
	if firstName != "John" {
		t.Errorf("firstName = %q, want John (camelCase should take precedence)", firstName)
	}

	// Assert the snake_case duplicates were removed.
	if _, ok := traveler["date_of_birth"]; ok {
		t.Error("date_of_birth should have been removed when dateOfBirth was present")
	}
	if _, ok := name["first_name"]; ok {
		t.Error("first_name should have been removed when firstName was present")
	}
	if _, ok := name["last_name"]; ok {
		t.Error("last_name should have been removed when lastName was present")
	}
}

func TestBookFlightNormalize_MultipleTravelers(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	input := json.RawMessage(`{
		"travelers": [
			{
				"name": {"first_name": "Alice", "last_name": "A"},
				"date_of_birth": "1985-03-10",
				"gender": "FEMALE",
				"contact": {"email": "a@e.com", "phone": "+1"}
			},
			{
				"name": {"firstName": "Bob", "lastName": "B"},
				"dateOfBirth": "1990-07-20",
				"gender": "MALE",
				"contact": {"email": "b@e.com", "phone": "+2"}
			}
		],
		"flight_offer": {"id":"1"},
		"payment_method_id": "pm_1",
		"idempotency_key": "key-1"
	}`)

	result := action.Normalize(input)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(result, &m); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	var travelers []map[string]json.RawMessage
	if err := json.Unmarshal(m["travelers"], &travelers); err != nil {
		t.Fatalf("unmarshal travelers: %v", err)
	}

	if len(travelers) != 2 {
		t.Fatalf("expected 2 travelers, got %d", len(travelers))
	}

	// First traveler: was snake_case, should be rewritten.
	if _, ok := travelers[0]["dateOfBirth"]; !ok {
		t.Error("traveler[0]: expected dateOfBirth")
	}
	var name0 map[string]json.RawMessage
	if err := json.Unmarshal(travelers[0]["name"], &name0); err != nil {
		t.Fatalf("unmarshal name[0]: %v", err)
	}
	if _, ok := name0["firstName"]; !ok {
		t.Error("traveler[0]: expected firstName")
	}

	// Second traveler: was already camelCase, should be unchanged.
	if _, ok := travelers[1]["dateOfBirth"]; !ok {
		t.Error("traveler[1]: expected dateOfBirth")
	}
}

func TestBookFlightNormalize_NoTravelers(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	input := json.RawMessage(`{"flight_offer": {"id":"1"}, "payment_method_id": "pm_1"}`)
	result := action.Normalize(input)

	// Should return unchanged since no travelers key.
	if string(result) != string(input) {
		t.Errorf("expected unchanged output when no travelers key")
	}
}

func TestBookFlightNormalize_EmptyParams(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	result := action.Normalize(nil)
	if result != nil {
		t.Errorf("expected nil for nil input, got %s", string(result))
	}

	result = action.Normalize(json.RawMessage{})
	if len(result) != 0 {
		t.Errorf("expected empty for empty input, got %s", string(result))
	}
}

func TestBookFlightNormalize_InvalidJSON(t *testing.T) {
	t.Parallel()

	action := &bookFlightAction{}

	input := json.RawMessage(`{invalid}`)
	result := action.Normalize(input)

	// Should return input unchanged on invalid JSON.
	if string(result) != string(input) {
		t.Errorf("expected unchanged output for invalid JSON")
	}
}
