package verify

import "strings"

// Stable finding codes. Answer failures append ":" and the question id.
const (
	CodeActionInconsistent        = "action_inconsistent"
	CodeBindingMismatch           = "binding_mismatch"
	CodeChecksumMismatch          = "checksum_mismatch"
	CodeChunkIdentity             = "chunk_identity"
	CodeEntityChecksumMismatch    = "entity_checksum_mismatch"
	CodeEstablishmentBelow        = "establishment_below_threshold"
	CodeEvidenceConflict          = "evidence_conflict"
	CodeInferenceOnly             = "inference_only"
	CodeInstructionInEvidence     = "instruction_in_evidence"
	CodeInvalidAnswers            = "invalid_answers"
	CodeMissingProvenance         = "missing_provenance"
	CodeNegativeResult            = "negative_result"
	CodeNoEligibleEvidence        = "no_eligible_evidence"
	CodePackEnvelope              = "pack_envelope"
	CodePackRebuild               = "pack_rebuild"
	CodePackRequestIdentity       = "pack_request_identity"
	CodePackShape                 = "pack_shape"
	CodePartialSurface            = "partial_surface"
	CodeProjectBinding            = "project_binding"
	CodeQueryMismatch             = "query_mismatch"
	CodeRejectionMismatch         = "rejection_mismatch"
	CodeReviewRequired            = "review_required"
	CodeSafetyGate                = "safety_gate"
	CodeSnapshotIdentity          = "snapshot_identity"
	CodeSourceAdmission           = "source_admission"
	CodeSupportBelow              = "support_below_threshold"
	CodeUnknownAdapter            = "unknown_adapter"
	CodeUnpinnedAdapter           = "unpinned_adapter"
	CodeUnpinnedEntity            = "unpinned_entity"
	CodeUnpinnedFocus             = "unpinned_focus"
	CodeUnpinnedPolicy            = "unpinned_policy"
	CodeUnpinnedQuestionSet       = "unpinned_question_set"
	CodeUnpinnedVerifier          = "unpinned_verifier"
	CodeUnresolvedModel           = "unresolved_model"
	CodePrefixAnswerTypeMismatch  = "answer_type_mismatch"
	CodePrefixInvalidChoice       = "invalid_choice"
	CodePrefixInvalidNoul         = "invalid_noul"
	CodePrefixInvalidScore        = "invalid_score"
	CodePrefixMissingAnswer       = "missing_answer"
	CodePrefixUnexpectedAnswer    = "unexpected_answer"
	CodePrefixUnsupportedQuestion = "unsupported_question_type"
)

// FindingCodes returns the stable codes that do not include a question id.
func FindingCodes() []string {
	return []string{
		CodeActionInconsistent,
		CodeBindingMismatch,
		CodeChecksumMismatch,
		CodeChunkIdentity,
		CodeEntityChecksumMismatch,
		CodeEstablishmentBelow,
		CodeEvidenceConflict,
		CodeInferenceOnly,
		CodeInstructionInEvidence,
		CodeInvalidAnswers,
		CodeMissingProvenance,
		CodeNegativeResult,
		CodeNoEligibleEvidence,
		CodePackEnvelope,
		CodePackRebuild,
		CodePackRequestIdentity,
		CodePackShape,
		CodePartialSurface,
		CodeProjectBinding,
		CodeQueryMismatch,
		CodeRejectionMismatch,
		CodeReviewRequired,
		CodeSafetyGate,
		CodeSnapshotIdentity,
		CodeSourceAdmission,
		CodeSupportBelow,
		CodeUnknownAdapter,
		CodeUnpinnedAdapter,
		CodeUnpinnedEntity,
		CodeUnpinnedFocus,
		CodeUnpinnedPolicy,
		CodeUnpinnedQuestionSet,
		CodeUnpinnedVerifier,
		CodeUnresolvedModel,
	}
}

// AnswerCodePrefixes returns codes completed as prefix + ":" + question id.
func AnswerCodePrefixes() []string {
	return []string{
		CodePrefixAnswerTypeMismatch,
		CodePrefixInvalidChoice,
		CodePrefixInvalidNoul,
		CodePrefixInvalidScore,
		CodePrefixMissingAnswer,
		CodePrefixUnexpectedAnswer,
		CodePrefixUnsupportedQuestion,
	}
}

// FindingCodePattern is the JSON Schema pattern for a finding code.
func FindingCodePattern() string {
	return "^(" + strings.Join(FindingCodes(), "|") + "|(" + strings.Join(AnswerCodePrefixes(), "|") + "):.+)$"
}

// KnownFinding reports whether code is in the v0.1 catalog.
func KnownFinding(code string) bool {
	for _, item := range FindingCodes() {
		if code == item {
			return true
		}
	}
	prefix, suffix, ok := strings.Cut(code, ":")
	if !ok || suffix == "" || strings.Contains(prefix, ":") {
		return false
	}
	for _, item := range AnswerCodePrefixes() {
		if prefix == item {
			return true
		}
	}
	return false
}
