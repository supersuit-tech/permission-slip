package amadeus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// bookFlightAction implements connectors.Action for amadeus.book_flight.
// It creates a flight booking (PNR) via POST /v1/booking/flight-orders.
//
// This is a HIGH RISK action — it creates real reservations.
// Payment details are resolved server-side from stored payment methods;
// the agent never sees raw card data.
type bookFlightAction struct {
	conn *AmadeusConnector
}

type bookFlightTravelerContact struct {
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type bookFlightTravelerName struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

type bookFlightTraveler struct {
	Name        bookFlightTravelerName    `json:"name"`
	DateOfBirth string                    `json:"dateOfBirth"`
	Gender      string                    `json:"gender"`
	Contact     bookFlightTravelerContact `json:"contact"`
}

type bookFlightParams struct {
	FlightOffer     json.RawMessage      `json:"flight_offer"`
	Travelers       []bookFlightTraveler `json:"travelers"`
	PaymentMethodID string               `json:"payment_method_id"`
	Remarks         string               `json:"remarks"`
}

var validGenders = map[string]bool{
	"MALE":   true,
	"FEMALE": true,
}

func (p *bookFlightParams) validate() error {
	if len(p.FlightOffer) == 0 || string(p.FlightOffer) == "null" {
		return &connectors.ValidationError{Message: "missing required parameter: flight_offer"}
	}
	if len(p.Travelers) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: travelers"}
	}
	for i, t := range p.Travelers {
		if t.Name.FirstName == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: missing required field: name.firstName", i)}
		}
		if t.Name.LastName == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: missing required field: name.lastName", i)}
		}
		if t.DateOfBirth == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: missing required field: dateOfBirth", i)}
		}
		if !validDate(t.DateOfBirth) {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: dateOfBirth must be YYYY-MM-DD format", i)}
		}
		if t.Gender == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: missing required field: gender", i)}
		}
		if !validGenders[t.Gender] {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: gender must be MALE or FEMALE", i)}
		}
		if t.Contact.Email == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: missing required field: contact.email", i)}
		}
		if t.Contact.Phone == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("travelers[%d]: missing required field: contact.phone", i)}
		}
	}
	if p.PaymentMethodID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: payment_method_id"}
	}
	return nil
}

func (a *bookFlightAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params bookFlightParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build traveler objects in Amadeus format.
	travelers := make([]map[string]any, len(params.Travelers))
	for i, t := range params.Travelers {
		travelers[i] = map[string]any{
			"id":          fmt.Sprintf("%d", i+1),
			"dateOfBirth": t.DateOfBirth,
			"gender":      t.Gender,
			"name": map[string]string{
				"firstName": t.Name.FirstName,
				"lastName":  t.Name.LastName,
			},
			"contact": map[string]any{
				"emailAddress": t.Contact.Email,
				"phones": []map[string]string{
					{"number": t.Contact.Phone},
				},
			},
		}
	}

	// Build the flight order request.
	// payment_method_id is included in the request body so that the
	// server-side payment resolution layer (#199) can inject real
	// payment details before forwarding to Amadeus.
	body := map[string]any{
		"data": map[string]any{
			"type":         "flight-order",
			"flightOffers": []json.RawMessage{params.FlightOffer},
			"travelers":    travelers,
			"formOfPayment": map[string]any{
				"other": map[string]string{
					"method":          "CASH",
					"paymentMethodId": params.PaymentMethodID,
				},
			},
		},
	}

	if params.Remarks != "" {
		data := body["data"].(map[string]any)
		data["remarks"] = map[string]any{
			"general": []map[string]string{
				{"text": params.Remarks},
			},
		}
	}

	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/v1/booking/flight-orders", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
