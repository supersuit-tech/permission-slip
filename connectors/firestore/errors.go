package firestore

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func mapFirestoreError(err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.NotFound:
			return &connectors.ValidationError{Message: fmt.Sprintf("Firestore not found: %v", err)}
		case codes.AlreadyExists, codes.FailedPrecondition, codes.InvalidArgument:
			return &connectors.ValidationError{Message: fmt.Sprintf("Firestore request rejected: %v", err)}
		case codes.PermissionDenied, codes.Unauthenticated:
			return &connectors.AuthError{Message: fmt.Sprintf("Firestore auth error: %v", err)}
		case codes.ResourceExhausted:
			return &connectors.RateLimitError{Message: fmt.Sprintf("Firestore quota or rate limit: %v", err)}
		case codes.DeadlineExceeded:
			return &connectors.TimeoutError{Message: fmt.Sprintf("Firestore deadline exceeded: %v", err)}
		}
	}
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Firestore request timed out: %v", err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Firestore error: %v", err)}
}
