package amadeus

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *AmadeusConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"amadeus.book_flight": makeParamValidator[bookFlightParams](),
	"amadeus.price_flight": makeParamValidator[priceFlightParams](),
	"amadeus.search_airports": makeParamValidator[searchAirportsParams](),
	"amadeus.search_cars": makeParamValidator[searchCarsParams](),
	"amadeus.search_flights": makeParamValidator[searchFlightsParams](),
	"amadeus.search_hotels": makeParamValidator[searchHotelsParams](),
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
