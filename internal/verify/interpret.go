package verify

import (
	"errors"
	"math"
)

// Thresholds are explicit policy gates. They are not a universal calibration.
type Thresholds struct {
	SupportMin   float64
	EstablishMin float64
	RefuteMin    float64
	ConflictMin  float64
	SafetyMin    float64
}

// ErrUnusablePolicy means the policy evaluator cannot apply its thresholds.
// It is not a denial finding.
var ErrUnusablePolicy = errors.New("policy thresholds are not usable")

// Interpret applies answer-domain checks and policy gates, then selects a verdict.
func Interpret(questions map[string]Question, thresholds Thresholds, answers map[string]Answer) (Verdict, []Finding, error) {
	if !ThresholdsUsable(thresholds) {
		return "", nil, ErrUnusablePolicy
	}
	findings := ValidateAnswers(questions, answers)
	findings = append(findings, policyFindings(thresholds, answers)...)
	verdict, err := ResolveVerdict(findings)
	if err != nil {
		return "", nil, err
	}
	if findings == nil {
		findings = []Finding{}
	}
	return verdict, findings, nil
}

// ThresholdsUsable reports whether every gate is a finite probability.
func ThresholdsUsable(thresholds Thresholds) bool {
	return unitInterval(thresholds.SupportMin) && unitInterval(thresholds.EstablishMin) && unitInterval(thresholds.RefuteMin) && unitInterval(thresholds.ConflictMin) && unitInterval(thresholds.SafetyMin)
}

func unitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func policyFindings(thresholds Thresholds, answers map[string]Answer) []Finding {
	support, supportOK := noul(answers, "support")
	established, establishedOK := noul(answers, "established")
	refuted, refutedOK := noul(answers, "refuted")
	conflict, conflictOK := noul(answers, "conflict")
	safety, safetyOK := noul(answers, "safe_to_auto_act")
	action, actionOK := choice(answers, "action")
	findings := make([]Finding, 0)

	negative := refutedOK && refuted >= thresholds.RefuteMin
	contradictory := negative && ((establishedOK && established >= thresholds.EstablishMin) || (supportOK && support >= thresholds.SupportMin))
	if (conflictOK && conflict >= thresholds.ConflictMin) || contradictory {
		findings = append(findings, Finding{Code: CodeEvidenceConflict, Verdict: VerdictConflict, Detail: "admissible evidence supports mutually incompatible conclusions"})
	}
	if supportOK && support < thresholds.SupportMin && !negative {
		findings = append(findings, Finding{Code: CodeSupportBelow, Verdict: VerdictInsufficient, Detail: "support is below the policy threshold"})
	}
	if establishedOK && established < thresholds.EstablishMin && !negative {
		findings = append(findings, Finding{Code: CodeEstablishmentBelow, Verdict: VerdictInsufficient, Detail: "establishment is below the policy threshold"})
	}
	if negative {
		findings = append(findings, Finding{Code: CodeNegativeResult, Verdict: VerdictRejected, Detail: "admissible evidence establishes the negation of the claim"})
	}
	if !actionOK {
		return findings
	}
	switch action {
	case "manual_review", "other":
		findings = append(findings, Finding{Code: CodeReviewRequired, Verdict: VerdictManualReview, Detail: "the action choice requires review"})
	case "proceed":
		if safetyOK && safety < thresholds.SafetyMin {
			findings = append(findings, Finding{Code: CodeSafetyGate, Verdict: VerdictManualReview, Detail: "semantic safety is below the policy threshold"})
		}
		if (supportOK && support < thresholds.SupportMin) || (establishedOK && established < thresholds.EstablishMin) || (conflictOK && conflict >= thresholds.ConflictMin) || negative {
			findings = append(findings, Finding{Code: CodeActionInconsistent, Verdict: VerdictManualReview, Detail: "the proceed choice disagrees with the predicate signals"})
		}
	case "reject":
		if !negative || (establishedOK && established >= thresholds.EstablishMin) {
			findings = append(findings, Finding{Code: CodeActionInconsistent, Verdict: VerdictManualReview, Detail: "the reject choice lacks an established uncontested refutation"})
		}
	}
	return findings
}

func noul(answers map[string]Answer, id string) (float64, bool) {
	answer, ok := answers[id]
	if !ok || answer.Type != QuestionNoul || answer.Noul == nil || !probability(*answer.Noul) {
		return 0, false
	}
	return *answer.Noul, true
}

func choice(answers map[string]Answer, id string) (string, bool) {
	answer, ok := answers[id]
	if !ok || answer.Type != QuestionChoice || answer.Choice == "" {
		return "", false
	}
	return answer.Choice, true
}
