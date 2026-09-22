package verify

import (
	"errors"
	"math"
)

// Gate is one deployment-owned policy applied to validated typed answers.
// The verifier does not know which questions a gate reads; the profile does.
type Gate interface {
	// Usable reports whether the gate can be applied at all. An unusable gate
	// is a technical failure, not a denial finding.
	Usable() bool
	// Findings returns the policy findings for one answer set. It must be
	// deterministic and must not rewrite the answers.
	Findings(answers map[string]Answer) []Finding
}

// ErrUnusablePolicy means the policy evaluator cannot apply its thresholds.
// It is not a denial finding.
var ErrUnusablePolicy = errors.New("policy thresholds are not usable")

// Interpret applies answer-domain checks and one policy gate, then selects a verdict.
func Interpret(questions map[string]Question, gate Gate, answers map[string]Answer) (Verdict, []Finding, error) {
	if gate == nil || !gate.Usable() {
		return "", nil, ErrUnusablePolicy
	}
	findings := ValidateAnswers(questions, answers)
	findings = append(findings, gate.Findings(answers)...)
	verdict, err := ResolveVerdict(findings)
	if err != nil {
		return "", nil, err
	}
	if findings == nil {
		findings = []Finding{}
	}
	return verdict, findings, nil
}

// UnitInterval reports whether value is a finite probability.
func UnitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

// Noul returns the probability answered for id when it is a valid Noul answer.
func Noul(answers map[string]Answer, id string) (float64, bool) {
	answer, ok := answers[id]
	if !ok || answer.Type != QuestionNoul || answer.Noul == nil || !probability(*answer.Noul) {
		return 0, false
	}
	return *answer.Noul, true
}

// Choice returns the option answered for id when it is a valid Choice answer.
func Choice(answers map[string]Answer, id string) (string, bool) {
	answer, ok := answers[id]
	if !ok || answer.Type != QuestionChoice || answer.Choice == "" {
		return "", false
	}
	return answer.Choice, true
}
