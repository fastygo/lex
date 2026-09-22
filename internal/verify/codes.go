package verify

import "strings"

// Stable finding codes. Answer failures append ":" and the question id.
const (
	CodeBindingMismatch           = "binding_mismatch"
	CodeDecisionChecksumMismatch  = "decision_checksum_mismatch"
	CodeInvalidAnswers            = "invalid_answers"
	CodePackRebuild               = "pack_rebuild"
	CodePackShape                 = "pack_shape"
	CodeProjectBinding            = "project_binding"
	CodeSnapshotIdentity          = "snapshot_identity"
	CodeUnknownAdapter            = "unknown_adapter"
	CodeUnpinnedAdapter           = "unpinned_adapter"
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
		CodeBindingMismatch,
		CodeDecisionChecksumMismatch,
		CodeInvalidAnswers,
		CodePackRebuild,
		CodePackShape,
		CodeProjectBinding,
		CodeSnapshotIdentity,
		CodeUnknownAdapter,
		CodeUnpinnedAdapter,
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

// KnownFinding reports whether code is in the published catalog.
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
