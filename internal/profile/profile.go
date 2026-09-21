// Package profile pins the deployment-owned question set and policy.
package profile

import (
	"encoding/json"

	"github.com/fastygo/lex/internal/canonical"
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
	// VerifierVersion pins the deterministic interpreter.
	VerifierVersion = "0.2.0"
	// AdapterVersion is the only typed-decision adapter contract this verifier replays.
	AdapterVersion = "0.1.0"
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

type question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

type questionSetDocument struct {
	ID        string              `json:"id"`
	Version   string              `json:"version"`
	Questions map[string]question `json:"questions"`
}

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

var (
	questionSet = questionSetDocument{
		ID:      QuestionSetID,
		Version: QuestionSetVersion,
		Questions: map[string]question{
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
	policy = policyDocument{
		ID:           PolicyID,
		Version:      PolicyVersion,
		Calibration:  Calibration,
		SupportMin:   0.7,
		EstablishMin: 0.8,
		RefuteMin:    0.8,
		ConflictMin:  0.5,
		SafetyMin:    0.8,
	}
	questionSetHash string
	policyHash      string
)

func init() {
	var err error
	questionSetHash, err = canonical.HashValue(questionSet)
	if err != nil {
		panic(err)
	}
	policyHash, err = canonical.HashValue(policy)
	if err != nil {
		panic(err)
	}
	if !verify.ThresholdsUsable(Thresholds()) {
		panic("embedded policy thresholds are not usable")
	}
}

// QuestionSetHash is the canonical hash of the question document, excluding the wire hash field.
func QuestionSetHash() string { return questionSetHash }

// PolicyHash is the canonical hash of the threshold document.
func PolicyHash() string { return policyHash }

// Thresholds returns the embedded uncalibrated policy gates.
func Thresholds() verify.Thresholds {
	return verify.Thresholds{
		SupportMin:   policy.SupportMin,
		EstablishMin: policy.EstablishMin,
		RefuteMin:    policy.RefuteMin,
		ConflictMin:  policy.ConflictMin,
		SafetyMin:    policy.SafetyMin,
	}
}

// TypedQuestions returns the verifier view of the embedded questions.
func TypedQuestions() map[string]verify.Question {
	return map[string]verify.Question{
		questionSupport:   {Type: verify.QuestionNoul},
		questionEstablish: {Type: verify.QuestionNoul},
		questionRefute:    {Type: verify.QuestionNoul},
		questionConflict:  {Type: verify.QuestionNoul},
		questionSafety:    {Type: verify.QuestionNoul},
		questionAction: {Type: verify.QuestionChoice, Choices: []string{
			"proceed", "reject", "manual_review", "other",
		}},
	}
}

// ProviderQuestions returns the question map sent to a decision adapter.
func ProviderQuestions() map[string]any {
	raw, err := json.Marshal(questionSet.Questions)
	if err != nil {
		panic(err)
	}
	var questions map[string]any
	if err := json.Unmarshal(raw, &questions); err != nil {
		panic(err)
	}
	return questions
}

// QuestionSetBody returns the question document embedded in a replay bundle.
func QuestionSetBody() any { return questionSet }
