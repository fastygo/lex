// Package profile pins the deployment-owned question set and policy.
package profile

import (
	"encoding/json"
	"strings"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/verify"
)

const (
	// QuestionSetID identifies the embedded claim-validation questions.
	QuestionSetID = "claim-validation"
	// QuestionSetVersion changes when question criteria change.
	QuestionSetVersion = "0.1.0"
	// PolicyID identifies the embedded threshold policy.
	PolicyID = "claim-validation"
	// PolicyVersion changes when thresholds change.
	PolicyVersion = "0.1.0"
	// VerifierVersion pins the deterministic interpreter.
	VerifierVersion = "0.1.0"
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
	// DirectModel is the only reproducible model identity for the direct adapter.
	DirectModel = "jev-1.13.0"
	// HostedModel is the request selector for the hosted adapter. A response
	// must add a resolved suffix; the selector alone is not reproducible.
	HostedModel = "typesafe/jev-1.13"
)

const (
	questionSupport   = "support"
	questionEstablish = "established"
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
				Instructions: "Does admissible frozen evidence directly support the claim?",
			},
			questionEstablish: {
				Type:         "noul",
				Instructions: "Do the frozen evidence items meet the sufficiency and coherence requirements for establishing the claim?",
			},
			questionConflict: {
				Type:         "noul",
				Instructions: "Does admissible frozen evidence support a conclusion incompatible with the claim?",
			},
			questionSafety: {
				Type:         "noul",
				Instructions: "Is the frozen evidence semantically safe to accept without human review?",
			},
			questionAction: {
				Type:         "choice",
				Instructions: "Which operational disposition follows from the frozen evidence alone?",
				Criteria: map[string]string{
					"proceed":       "Accept the claim because support, establishment, and safety all hold",
					"reject":        "Reject the claim because admissible evidence establishes that it is not supported",
					"manual_review": "Send the claim to a person because the evidence is weak, unsafe, or incomplete",
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
}

// ReproducibleModel reports whether a sealed model identity can support replay
// for the named adapter. An alias containing "latest", an echoed hosted
// selector, and a different model line are not reproducible.
func ReproducibleModel(adapterID, model string) bool {
	if model == "" || strings.Contains(model, "latest") {
		return false
	}
	switch adapterID {
	case "direct-systemone":
		return model == DirectModel
	case "hosted-systemone":
		return hostedResolved(model)
	default:
		return false
	}
}

func hostedResolved(model string) bool {
	for _, separator := range []string{"-", "."} {
		rest, ok := strings.CutPrefix(model, HostedModel+separator)
		if ok && rest != "" && !strings.ContainsAny(rest, "/ \t") {
			return true
		}
	}
	return false
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
		ConflictMin:  policy.ConflictMin,
		SafetyMin:    policy.SafetyMin,
	}
}

// TypedQuestions returns the verifier view of the embedded questions.
func TypedQuestions() map[string]verify.Question {
	return map[string]verify.Question{
		questionSupport:   {Type: verify.QuestionNoul},
		questionEstablish: {Type: verify.QuestionNoul},
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
