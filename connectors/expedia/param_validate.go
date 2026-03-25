package expedia

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *ExpediaConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"expedia.cancel_booking": makeParamValidator[cancelBookingParams](),
	"expedia.create_booking": makeParamValidator[createBookingParams](),
	"expedia.get_booking": makeParamValidator[getBookingParams](),
	"expedia.get_hotel": makeParamValidator[getHotelParams](),
	"expedia.price_check": makeParamValidator[priceCheckParams](),
	"expedia.search_hotels": makeParamValidator[searchHotelsParams](),
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
