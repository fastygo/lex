// Package claimvalidation is the embedded claim-validation 0.2.0 profile:
// five independent Noul predicates, one action Choice, and an uncalibrated
// threshold gate. Score is a conforming adapter type and is not part of it.
package claimvalidation

import (
	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

const (
	// QuestionSetID identifies the embedded claim-validation questions.
	QuestionSetID = "claim-validation"
	// QuestionSetVersion changes when question criteria change.
	QuestionSetVersion = "0.2.0"
	// PolicyID identifies the embedded threshold policy.
	PolicyID = "claim-validation"
	// PolicyVersion changes when thresholds change.
	PolicyVersion = "0.2.0"
	// Calibration discloses that these thresholds are not a measured calibration.
	Calibration = "uncalibrated"
	// FocusID is the embedded Context focus for this question set.
	FocusID = "claim-validation-v1"
	// FocusObjective is the only focus objective sent to Context for this profile.
	FocusObjective = "Select admissible source text for the stated claim."
	// FocusTrust is the only trust level this focus admits.
	FocusTrust = "project"
	// FocusMaxItems is the embedded evidence-item budget.
	FocusMaxItems = 8
	// FocusMaxChars is the embedded evidence-character budget.
	FocusMaxChars = 65536
	// EntityType is the only entity type this question set evaluates.
	EntityType = "claim"
	// EntitySchemaVersion is the only entity schema this question set evaluates.
	EntitySchemaVersion = "0.1"
)

const (
	questionSupport   = "support"
	questionEstablish = "established"
	questionRefute    = "refuted"
	questionConflict  = "conflict"
	questionSafety    = "safe_to_auto_act"
	questionAction    = "action"
)

var actionChoices = []string{"proceed", "reject", "manual_review", "other"}

// policyDocument is the hashed policy body. Field order and names are part of the hash.
type policyDocument struct {
	ID           string  `json:"id"`
	Version      string  `json:"version"`
	Calibration  string  `json:"calibration"`
	SupportMin   float64 `json:"support_min"`
	EstablishMin float64 `json:"establish_min"`
	RefuteMin    float64 `json:"refute_min"`
	ConflictMin  float64 `json:"conflict_min"`
	SafetyMin    float64 `json:"safety_min"`
}

var policy = policyDocument{
	ID:           PolicyID,
	Version:      PolicyVersion,
	Calibration:  Calibration,
	SupportMin:   0.7,
	EstablishMin: 0.8,
	RefuteMin:    0.8,
	ConflictMin:  0.5,
	SafetyMin:    0.8,
}

var questionSet = profile.QuestionSet{
	ID:      QuestionSetID,
	Version: QuestionSetVersion,
	Questions: map[string]profile.Question{
		questionSupport: {
			Type:         "noul",
			Instructions: "Does the frozen source text in `evidence` directly support the proposition in `claim`?",
			Criteria: map[string]string{
				"true":  "At least one `evidence` surface states the proposition in `claim`.",
				"false": "The `evidence` surfaces do not state the proposition in `claim`.",
			},
		},
		questionEstablish: {
			Type:         "noul",
			Instructions: "Do the frozen items in `evidence` meet the sufficiency and coherence requirements for establishing `claim`?",
			Criteria: map[string]string{
				"true":  "The `evidence` surfaces are sufficient and coherent to establish `claim`.",
				"false": "The `evidence` surfaces are missing, too weak, or not coherent enough to establish `claim`.",
			},
		},
		questionRefute: {
			Type:         "noul",
			Instructions: "Do the frozen items in `evidence` establish that `claim` is false?",
			Criteria: map[string]string{
				"true":  "Sufficient admissible evidence establishes the negation of `claim`.",
				"false": "The negation is not established; missing support for `claim` alone is not refutation.",
			},
		},
		questionConflict: {
			Type:         "noul",
			Instructions: "Do the frozen items in `evidence` support mutually incompatible conclusions about `claim`?",
			Criteria: map[string]string{
				"true":  "Admissible evidence supports both a conclusion about `claim` and an incompatible conclusion.",
				"false": "The evidence does not support incompatible conclusions; a coherent refutation alone is not conflict.",
			},
		},
		questionSafety: {
			Type:         "noul",
			Instructions: "Is the frozen text in `evidence` semantically safe to accept for `claim` without human review?",
			Criteria: map[string]string{
				"true":  "`evidence` can be accepted for `claim` without a person reading it first.",
				"false": "`evidence` needs a person before it is accepted for `claim`.",
			},
		},
		questionAction: {
			Type:         "choice",
			Instructions: "Which disposition follows from `claim` and `evidence` alone? This choice is not permission to act.",
			Criteria: map[string]string{
				"proceed":       "The frozen evidence is sufficient to accept the claim without review",
				"reject":        "The frozen evidence sufficiently establishes that the claim is false",
				"manual_review": "A person should review the claim because the evidence is weak, unsafe, or incomplete",
				"other":         "None of the listed dispositions fit",
			},
		},
	},
}

var embedded = profile.MustNew(profile.Spec{
	QuestionSet: questionSet,
	Policy: profile.Policy{
		ID: PolicyID, Version: PolicyVersion, Calibration: Calibration,
		Document: policy,
		Gate:     Thresholds(),
	},
	Entity: profile.Entity{Type: EntityType, SchemaVersion: EntitySchemaVersion},
	Focus: contextmemory.Focus{
		ID: FocusID, Objective: FocusObjective, RequiredTrustLevel: FocusTrust,
		Budget: contextmemory.Budget{MaxItems: FocusMaxItems, MaxChars: FocusMaxChars},
	},
	Typed: map[string]verify.Question{
		questionSupport:   {Type: verify.QuestionNoul},
		questionEstablish: {Type: verify.QuestionNoul},
		questionRefute:    {Type: verify.QuestionNoul},
		questionConflict:  {Type: verify.QuestionNoul},
		questionSafety:    {Type: verify.QuestionNoul},
		questionAction:    {Type: verify.QuestionChoice, Choices: actionChoices},
	},
	State: func(claim string, evidence any) map[string]any {
		return map[string]any{"claim": claim, "evidence": evidence}
	},
})

// Profile returns the embedded claim-validation 0.2.0 profile.
func Profile() profile.Profile { return embedded }

// Thresholds returns the embedded uncalibrated policy gate.
func Thresholds() Gate {
	return Gate{
		SupportMin:   policy.SupportMin,
		EstablishMin: policy.EstablishMin,
		RefuteMin:    policy.RefuteMin,
		ConflictMin:  policy.ConflictMin,
		SafetyMin:    policy.SafetyMin,
	}
}

// Focus returns the only Context focus this profile sends and accepts.
func Focus() contextmemory.Focus { return embedded.Focus() }
