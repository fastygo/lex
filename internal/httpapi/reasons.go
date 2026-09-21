package httpapi

// Problem reason codes are the stable machine identifiers in RFC 9457 responses.
const (
	reasonAdmissionLimited            = "admission_limited"
	reasonAuthenticationDenied        = "authentication_denied"
	reasonAuthenticationRequired      = "authentication_required"
	reasonBodyTooLarge                = "body_too_large"
	reasonClientCanceled              = "client_canceled"
	reasonDeadlineExceeded            = "deadline_exceeded"
	reasonDecisionError               = "decision_error"
	reasonDecisionProviderUnavailable = "decision_provider_unavailable"
	reasonInvalidBody                 = "invalid_body"
	reasonInvalidJSON                 = "invalid_json"
	reasonInvalidReplayBundle         = "invalid_replay_bundle"
	reasonJSONTooDeep                 = "json_too_deep"
	reasonMethodNotAllowed            = "method_not_allowed"
	reasonNotAcceptable               = "not_acceptable"
	reasonNotFound                    = "not_found"
	reasonPackError                   = "pack_error"
	reasonProjectForbidden            = "project_forbidden"
	reasonQuestionError               = "question_error"
	reasonResponseBudget              = "response_budget"
	reasonUnsupportedMediaType        = "unsupported_media_type"
	reasonVerificationError           = "verification_error"
)

func problemReasons() []string {
	return []string{
		reasonAdmissionLimited,
		reasonAuthenticationDenied,
		reasonAuthenticationRequired,
		reasonBodyTooLarge,
		reasonClientCanceled,
		reasonDeadlineExceeded,
		reasonDecisionError,
		reasonDecisionProviderUnavailable,
		reasonInvalidBody,
		reasonInvalidJSON,
		reasonInvalidReplayBundle,
		reasonJSONTooDeep,
		reasonMethodNotAllowed,
		reasonNotAcceptable,
		reasonNotFound,
		reasonPackError,
		reasonProjectForbidden,
		reasonQuestionError,
		reasonResponseBudget,
		reasonUnsupportedMediaType,
		reasonVerificationError,
	}
}
