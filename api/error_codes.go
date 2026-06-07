package api

// ErrorCode represents a machine-readable error code from the API spec.
type ErrorCode string

const (
	ErrInvalidRequest               ErrorCode = "invalid_request"
	ErrInvalidActionType            ErrorCode = "invalid_action_type"
	ErrUnsupportedActionType        ErrorCode = "unsupported_action_type"
	ErrUnsupportedActionVersion     ErrorCode = "unsupported_action_version"
	ErrAgentIDMismatch              ErrorCode = "agent_id_mismatch"
	ErrInvalidPublicKey             ErrorCode = "invalid_public_key"
	ErrInvalidSignature             ErrorCode = "invalid_signature"
	ErrTimestampExpired             ErrorCode = "timestamp_expired"
	ErrInvalidCode                  ErrorCode = "invalid_code"
	ErrInvalidToken                 ErrorCode = "invalid_token"
	ErrAgentNotAuthorized           ErrorCode = "agent_not_authorized"
	ErrApprovalDenied               ErrorCode = "approval_denied"
	ErrInvalidParameters            ErrorCode = "invalid_parameters"
	ErrAgentNotFound                ErrorCode = "agent_not_found"
	ErrApproverNotFound             ErrorCode = "approver_not_found"
	ErrApprovalNotFound             ErrorCode = "approval_not_found"
	ErrProfileNotFound              ErrorCode = "profile_not_found"
	ErrCredentialNotFound           ErrorCode = "credential_not_found"
	ErrConnectorNotFound            ErrorCode = "connector_not_found"
	ErrCredentialsNotFound          ErrorCode = "credentials_not_found"
	ErrNoMatchingStanding           ErrorCode = "no_matching_standing_approval"
	ErrConstraintViolation          ErrorCode = "constraint_violation"
	ErrStandingExpired              ErrorCode = "standing_approval_expired"
	ErrAgentAlreadyRegistered       ErrorCode = "agent_already_registered"
	ErrDuplicateRequestID           ErrorCode = "duplicate_request_id"
	ErrApprovalAlreadyResolved      ErrorCode = "approval_already_resolved"
	ErrDuplicateCredential          ErrorCode = "duplicate_credential"
	ErrRegistrationExpired          ErrorCode = "registration_expired"
	ErrVerificationLocked           ErrorCode = "verification_locked"
	ErrApprovalExpired              ErrorCode = "approval_expired"
	ErrApprovalCancelled            ErrorCode = "approval_cancelled"
	ErrApprovalRecentlyDenied       ErrorCode = "approval_recently_denied"
	ErrRateLimited                  ErrorCode = "rate_limited"
	ErrInternalError                ErrorCode = "internal_error"
	ErrInternalPanic                ErrorCode = "internal_panic"
	ErrUpstreamError                ErrorCode = "upstream_error"
	ErrActionConfigNotFound         ErrorCode = "action_config_not_found"
	ErrActionConfigTemplateNotFound ErrorCode = "action_config_template_not_found"
	ErrInvalidConfiguration         ErrorCode = "invalid_configuration"
	ErrConfigurationDisabled        ErrorCode = "configuration_disabled"
	ErrConfigActionTypeMismatch     ErrorCode = "configuration_action_type_mismatch"
	ErrInvalidReference             ErrorCode = "invalid_reference"
	ErrServiceUnavailable           ErrorCode = "service_unavailable"
	// OAuth
	ErrOAuthProviderNotFound     ErrorCode = "oauth_provider_not_found"
	ErrOAuthProviderUnconfigured ErrorCode = "oauth_provider_unconfigured"
	ErrOAuthConnectionNotFound   ErrorCode = "oauth_connection_not_found"
	ErrOAuthConnectionExists     ErrorCode = "oauth_connection_exists"
	ErrOAuthStateMismatch        ErrorCode = "oauth_state_mismatch"
	ErrOAuthExchangeFailed       ErrorCode = "oauth_exchange_failed"
	ErrOAuthRefreshFailed        ErrorCode = "oauth_refresh_failed"
	// Payment Methods
	ErrPaymentMethodNotFound ErrorCode = "payment_method_not_found"
	ErrPaymentMethodRequired ErrorCode = "payment_method_required"
	ErrPaymentLimitExceeded  ErrorCode = "payment_limit_exceeded"

	// Parameter validation (request-time, distinct from execution-time invalid_parameters)
	ErrMissingRequiredParameters ErrorCode = "missing_required_parameters"

	// Standing approval constraint validation
	ErrInvalidConstraints ErrorCode = "invalid_constraints"

	ErrConnectorInstanceRequired  ErrorCode = "connector_instance_required"
	ErrConnectorInstanceNotFound  ErrorCode = "connector_instance_not_found"
	ErrConnectorInstanceAmbiguous ErrorCode = "connector_instance_ambiguous"

	// Notification channels (e.g. web push when not enabled for this deployment)
	ErrChannelNotConfigured ErrorCode = "channel_not_configured"
)
