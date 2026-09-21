// Package verify provides deterministic LeX verification primitives.
package verify

import "fmt"

// Verdict is the primary validation disposition.
type Verdict string

const (
	VerdictValidated    Verdict = "validated"
	VerdictRejected     Verdict = "rejected"
	VerdictInsufficient Verdict = "insufficient"
	VerdictConflict     Verdict = "conflict"
	VerdictManualReview Verdict = "manual_review"
	VerdictError        Verdict = "error"
)

// Finding is retained evidence for a verifier conclusion.
type Finding struct {
	Code    string  `json:"code"`
	Verdict Verdict `json:"verdict"`
	Detail  string  `json:"detail"`
}

// ResolveVerdict selects the primary verdict without removing any findings.
// No findings means every declared deterministic obligation passed.
func ResolveVerdict(findings []Finding) (Verdict, error) {
	primary := VerdictValidated
	for _, finding := range findings {
		if finding.Code == "" {
			return "", fmt.Errorf("finding code is required")
		}
		if !validVerdict(finding.Verdict) {
			return "", fmt.Errorf("unknown finding verdict %q", finding.Verdict)
		}
		if precedence(finding.Verdict) > precedence(primary) {
			primary = finding.Verdict
		}
	}
	return primary, nil
}

func validVerdict(verdict Verdict) bool {
	switch verdict {
	case VerdictValidated, VerdictRejected, VerdictInsufficient, VerdictConflict, VerdictManualReview, VerdictError:
		return true
	default:
		return false
	}
}

func precedence(verdict Verdict) int {
	switch verdict {
	case VerdictError:
		return 6
	case VerdictConflict:
		return 5
	case VerdictInsufficient:
		return 4
	case VerdictManualReview:
		return 3
	case VerdictRejected:
		return 2
	case VerdictValidated:
		return 1
	default:
		return 0
	}
}
