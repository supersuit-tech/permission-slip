package dynamodb

import (
	"errors"
	"fmt"

	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	smithy "github.com/aws/smithy-go"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func mapDynamoError(err error) error {
	if err == nil {
		return nil
	}
	var rn *ddbtypes.ResourceNotFoundException
	if errors.As(err, &rn) {
		return &connectors.ValidationError{Message: fmt.Sprintf("DynamoDB resource not found: %v", err)}
	}
	var cc *ddbtypes.ConditionalCheckFailedException
	if errors.As(err, &cc) {
		return &connectors.ValidationError{Message: fmt.Sprintf("DynamoDB conditional check failed: %v", err)}
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "ProvisionedThroughputExceededException", "RequestLimitExceeded", "ThrottlingException":
			return &connectors.RateLimitError{Message: fmt.Sprintf("DynamoDB throttled: %v", err)}
		case "AccessDeniedException", "UnrecognizedClientException", "InvalidSignatureException":
			return &connectors.AuthError{Message: fmt.Sprintf("DynamoDB auth error: %v", err)}
		}
	}
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("DynamoDB request timed out: %v", err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("DynamoDB error: %v", err)}
}
