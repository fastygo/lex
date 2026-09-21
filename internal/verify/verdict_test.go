package verify

import "testing"

func TestResolveVerdictPrecedence(t *testing.T) {
	tests := []struct {
		name     string
		findings []Finding
		want     Verdict
	}{
		{name: "no findings", want: VerdictValidated},
		{name: "rejected", findings: []Finding{{Code: "negative", Verdict: VerdictRejected}}, want: VerdictRejected},
		{name: "review outranks rejected", findings: []Finding{{Code: "negative", Verdict: VerdictRejected}, {Code: "approval", Verdict: VerdictManualReview}}, want: VerdictManualReview},
		{name: "insufficient outranks review", findings: []Finding{{Code: "approval", Verdict: VerdictManualReview}, {Code: "missing", Verdict: VerdictInsufficient}}, want: VerdictInsufficient},
		{name: "conflict outranks insufficient", findings: []Finding{{Code: "missing", Verdict: VerdictInsufficient}, {Code: "conflict", Verdict: VerdictConflict}}, want: VerdictConflict},
		{name: "error outranks conflict", findings: []Finding{{Code: "conflict", Verdict: VerdictConflict}, {Code: "broken", Verdict: VerdictError}}, want: VerdictError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveVerdict(test.findings)
			if err != nil {
				t.Fatalf("ResolveVerdict() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("ResolveVerdict() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestResolveVerdictRejectsUnknownVerdict(t *testing.T) {
	_, err := ResolveVerdict([]Finding{{Code: "unknown", Verdict: "unknown"}})
	if err == nil {
		t.Fatal("expected unknown verdict error")
	}
}
