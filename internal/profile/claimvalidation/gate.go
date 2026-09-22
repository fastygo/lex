package claimvalidation

import "github.com/fastygo/lex/internal/verify"

// Gate holds the explicit claim-validation thresholds. They are policy, not a
// universal calibration, and they are external to the decision model.
type Gate struct {
	SupportMin   float64
	EstablishMin float64
	RefuteMin    float64
	ConflictMin  float64
	SafetyMin    float64
}

// Usable reports whether every threshold is a finite probability.
func (g Gate) Usable() bool {
	return verify.UnitInterval(g.SupportMin) && verify.UnitInterval(g.EstablishMin) && verify.UnitInterval(g.RefuteMin) && verify.UnitInterval(g.ConflictMin) && verify.UnitInterval(g.SafetyMin)
}

// Findings applies the gate. Predicates are read independently; a coherent
// refutation is rejected, not conflicting, and the action choice never
// overrides the predicates.
func (g Gate) Findings(answers map[string]verify.Answer) []verify.Finding {
	support, supportOK := verify.Noul(answers, questionSupport)
	established, establishedOK := verify.Noul(answers, questionEstablish)
	refuted, refutedOK := verify.Noul(answers, questionRefute)
	conflict, conflictOK := verify.Noul(answers, questionConflict)
	safety, safetyOK := verify.Noul(answers, questionSafety)
	action, actionOK := verify.Choice(answers, questionAction)
	findings := make([]verify.Finding, 0)

	negative := refutedOK && refuted >= g.RefuteMin
	contradictory := negative && ((establishedOK && established >= g.EstablishMin) || (supportOK && support >= g.SupportMin))
	if (conflictOK && conflict >= g.ConflictMin) || contradictory {
		findings = append(findings, verify.Finding{Code: verify.CodeEvidenceConflict, Verdict: verify.VerdictConflict, Detail: "admissible evidence supports mutually incompatible conclusions"})
	}
	if supportOK && support < g.SupportMin && !negative {
		findings = append(findings, verify.Finding{Code: verify.CodeSupportBelow, Verdict: verify.VerdictInsufficient, Detail: "support is below the policy threshold"})
	}
	if establishedOK && established < g.EstablishMin && !negative {
		findings = append(findings, verify.Finding{Code: verify.CodeEstablishmentBelow, Verdict: verify.VerdictInsufficient, Detail: "establishment is below the policy threshold"})
	}
	if negative {
		findings = append(findings, verify.Finding{Code: verify.CodeNegativeResult, Verdict: verify.VerdictRejected, Detail: "admissible evidence establishes the negation of the claim"})
	}
	if !actionOK {
		return findings
	}
	switch action {
	case "manual_review", "other":
		findings = append(findings, verify.Finding{Code: verify.CodeReviewRequired, Verdict: verify.VerdictManualReview, Detail: "the action choice requires review"})
	case "proceed":
		if safetyOK && safety < g.SafetyMin {
			findings = append(findings, verify.Finding{Code: verify.CodeSafetyGate, Verdict: verify.VerdictManualReview, Detail: "semantic safety is below the policy threshold"})
		}
		if (supportOK && support < g.SupportMin) || (establishedOK && established < g.EstablishMin) || (conflictOK && conflict >= g.ConflictMin) || negative {
			findings = append(findings, verify.Finding{Code: verify.CodeActionInconsistent, Verdict: verify.VerdictManualReview, Detail: "the proceed choice disagrees with the predicate signals"})
		}
	case "reject":
		if !negative || (establishedOK && established >= g.EstablishMin) {
			findings = append(findings, verify.Finding{Code: verify.CodeActionInconsistent, Verdict: verify.VerdictManualReview, Detail: "the reject choice lacks an established uncontested refutation"})
		}
	}
	return findings
}
