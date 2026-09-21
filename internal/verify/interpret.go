package verify

// Thresholds are explicit policy gates. They are not a universal calibration.
type Thresholds struct {
	SupportMin   float64
	EstablishMin float64
	ConflictMin  float64
	SafetyMin    float64
}

// Interpret applies answer-domain checks and policy gates, then selects a verdict.
func Interpret(questions map[string]Question, thresholds Thresholds, answers map[string]Answer) (Verdict, []Finding, error) {
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

func policyFindings(thresholds Thresholds, answers map[string]Answer) []Finding {
	support, supportOK := noul(answers, "support")
	established, establishedOK := noul(answers, "established")
	conflict, conflictOK := noul(answers, "conflict")
	safety, safetyOK := noul(answers, "safe_to_auto_act")
	action, actionOK := choice(answers, "action")
	findings := make([]Finding, 0)

	negative := supportOK && establishedOK && conflictOK && support < thresholds.SupportMin && established >= thresholds.EstablishMin && conflict < thresholds.ConflictMin
	if conflictOK && conflict >= thresholds.ConflictMin {
		findings = append(findings, Finding{Code: CodeEvidenceConflict, Verdict: VerdictConflict, Detail: "admissible evidence supports an incompatible conclusion"})
	}
	if supportOK && support < thresholds.SupportMin && !negative {
		findings = append(findings, Finding{Code: CodeSupportBelow, Verdict: VerdictInsufficient, Detail: "support is below the policy threshold"})
	}
	if establishedOK && established < thresholds.EstablishMin {
		findings = append(findings, Finding{Code: CodeEstablishmentBelow, Verdict: VerdictInsufficient, Detail: "establishment is below the policy threshold"})
	}
	if negative {
		findings = append(findings, Finding{Code: CodeNegativeResult, Verdict: VerdictRejected, Detail: "admissible evidence establishes that the claim is not supported"})
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
		if (supportOK && support < thresholds.SupportMin) || (establishedOK && established < thresholds.EstablishMin) || (conflictOK && conflict >= thresholds.ConflictMin) {
			findings = append(findings, Finding{Code: CodeActionInconsistent, Verdict: VerdictManualReview, Detail: "the proceed choice disagrees with the predicate signals"})
		}
	case "reject":
		if supportOK && support >= thresholds.SupportMin {
			findings = append(findings, Finding{Code: CodeActionInconsistent, Verdict: VerdictManualReview, Detail: "the reject choice disagrees with support"})
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
