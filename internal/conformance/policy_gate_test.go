package conformance

import (
	"math"
	"testing"

	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

// specGate is a separate reading of the v0.1 policy gates. It does not call
// the verifier's finding builder.
func specGate(support, established, conflict, safety float64, action string) (string, []string) {
	thresholds := profile.Thresholds()
	var codes []string
	negative := support < thresholds.SupportMin && established >= thresholds.EstablishMin && conflict < thresholds.ConflictMin
	if conflict >= thresholds.ConflictMin {
		codes = append(codes, "evidence_conflict")
	}
	if support < thresholds.SupportMin && !negative {
		codes = append(codes, "support_below_threshold")
	}
	if established < thresholds.EstablishMin {
		codes = append(codes, "establishment_below_threshold")
	}
	if negative {
		codes = append(codes, "negative_result")
	}
	switch action {
	case "manual_review", "other":
		codes = append(codes, "review_required")
	case "proceed":
		if safety < thresholds.SafetyMin {
			codes = append(codes, "safety_gate")
		}
		if support < thresholds.SupportMin || established < thresholds.EstablishMin || conflict >= thresholds.ConflictMin {
			codes = append(codes, "action_inconsistent")
		}
	case "reject":
		if support >= thresholds.SupportMin {
			codes = append(codes, "action_inconsistent")
		}
	}
	return specPrimary(codes), codes
}

func specPrimary(codes []string) string {
	verdictOf := map[string]string{
		"evidence_conflict":             "conflict",
		"support_below_threshold":       "insufficient",
		"establishment_below_threshold": "insufficient",
		"negative_result":               "rejected",
		"review_required":               "manual_review",
		"safety_gate":                   "manual_review",
		"action_inconsistent":           "manual_review",
	}
	rank := map[string]int{
		"error": 6, "conflict": 5, "insufficient": 4, "manual_review": 3, "rejected": 2, "validated": 1,
	}
	primary := "validated"
	for _, code := range codes {
		verdict := verdictOf[code]
		if rank[verdict] > rank[primary] {
			primary = verdict
		}
	}
	return primary
}

func TestIndependentPolicyGateAgreesWithVerifier(t *testing.T) {
	questions := map[string]verify.Question{
		"support":          {Type: verify.QuestionNoul},
		"established":      {Type: verify.QuestionNoul},
		"conflict":         {Type: verify.QuestionNoul},
		"safe_to_auto_act": {Type: verify.QuestionNoul},
		"action":           {Type: verify.QuestionChoice, Choices: []string{"proceed", "reject", "manual_review", "other"}},
	}
	cases := []struct {
		name                                   string
		support, established, conflict, safety float64
		action                                 string
	}{
		{name: "validated", support: 0.9, established: 0.9, conflict: 0.1, safety: 0.9, action: "proceed"},
		{name: "thresholds-hold", support: 0.7, established: 0.8, conflict: math.Nextafter(0.5, 0), safety: 0.8, action: "proceed"},
		{name: "conflict-at-threshold", support: 0.9, established: 0.9, conflict: 0.5, safety: 0.9, action: "manual_review"},
		{name: "safety-just-below", support: 0.9, established: 0.9, conflict: 0.1, safety: math.Nextafter(0.8, 0), action: "proceed"},
		{name: "negative", support: 0.2, established: 0.9, conflict: 0.1, safety: 0.9, action: "reject"},
		{name: "weak-and-review", support: 0.2, established: 0.2, conflict: 0.1, safety: 0.9, action: "manual_review"},
		{name: "conflict-keeps-review", support: 0.9, established: 0.2, conflict: 0.5, safety: 0.2, action: "proceed"},
		{name: "safety-gate", support: 0.9, established: 0.9, conflict: 0.1, safety: 0.2, action: "proceed"},
		{name: "reject-disagrees", support: 0.9, established: 0.9, conflict: 0.1, safety: 0.9, action: "reject"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantVerdict, wantCodes := specGate(tc.support, tc.established, tc.conflict, tc.safety, tc.action)
			gotVerdict, findings, err := verify.Interpret(questions, profile.Thresholds(), map[string]verify.Answer{
				"support":          noulAnswer(tc.support),
				"established":      noulAnswer(tc.established),
				"conflict":         noulAnswer(tc.conflict),
				"safe_to_auto_act": noulAnswer(tc.safety),
				"action":           choiceAnswer(tc.action),
			})
			if err != nil {
				t.Fatal(err)
			}
			if string(gotVerdict) != wantVerdict {
				t.Fatalf("verdict = %s, independent gate = %s", gotVerdict, wantVerdict)
			}
			got := map[string]bool{}
			for _, finding := range findings {
				got[finding.Code] = true
			}
			if len(got) != len(wantCodes) {
				t.Fatalf("findings = %v, independent codes = %v", findings, wantCodes)
			}
			for _, code := range wantCodes {
				if !got[code] {
					t.Fatalf("missing %s in %v", code, findings)
				}
			}
		})
	}
}

func TestPolicyTruthTableAgreesWithVerifier(t *testing.T) {
	questions := map[string]verify.Question{
		"support":          {Type: verify.QuestionNoul},
		"established":      {Type: verify.QuestionNoul},
		"conflict":         {Type: verify.QuestionNoul},
		"safe_to_auto_act": {Type: verify.QuestionNoul},
		"action":           {Type: verify.QuestionChoice, Choices: []string{"proceed", "reject", "manual_review", "other"}},
	}
	thresholds := profile.Thresholds()
	supports := []float64{math.Nextafter(thresholds.SupportMin, 0), thresholds.SupportMin}
	established := []float64{math.Nextafter(thresholds.EstablishMin, 0), thresholds.EstablishMin}
	conflicts := []float64{math.Nextafter(thresholds.ConflictMin, 0), thresholds.ConflictMin}
	safety := []float64{math.Nextafter(thresholds.SafetyMin, 0), thresholds.SafetyMin}
	actions := []string{"proceed", "reject", "manual_review", "other"}
	seen := 0
	for _, support := range supports {
		for _, establish := range established {
			for _, conflict := range conflicts {
				for _, safe := range safety {
					for _, action := range actions {
						seen++
						wantVerdict, wantCodes := specGate(support, establish, conflict, safe, action)
						gotVerdict, findings, err := verify.Interpret(questions, thresholds, map[string]verify.Answer{
							"support":          noulAnswer(support),
							"established":      noulAnswer(establish),
							"conflict":         noulAnswer(conflict),
							"safe_to_auto_act": noulAnswer(safe),
							"action":           choiceAnswer(action),
						})
						if err != nil {
							t.Fatal(err)
						}
						if conflict >= thresholds.ConflictMin && gotVerdict != verify.VerdictConflict {
							t.Fatalf("conflict %.17g action %s verdict = %s, want conflict", conflict, action, gotVerdict)
						}
						if string(gotVerdict) != wantVerdict {
							t.Fatalf("support %.17g established %.17g conflict %.17g safety %.17g action %s verdict = %s, independent gate = %s", support, establish, conflict, safe, action, gotVerdict, wantVerdict)
						}
						got := map[string]bool{}
						for _, finding := range findings {
							got[finding.Code] = true
						}
						if len(got) != len(wantCodes) {
							t.Fatalf("findings = %v, independent codes = %v", findings, wantCodes)
						}
						for _, code := range wantCodes {
							if !got[code] {
								t.Fatalf("missing %s in %v", code, findings)
							}
						}
					}
				}
			}
		}
	}
	if seen != 64 {
		t.Fatalf("truth table cells = %d, want 64", seen)
	}
}

func noulAnswer(value float64) verify.Answer {
	return verify.Answer{Type: verify.QuestionNoul, Noul: &value}
}

func choiceAnswer(choice string) verify.Answer {
	probabilities := map[string]float64{"proceed": 0, "reject": 0, "manual_review": 0, "other": 0}
	probabilities[choice] = 1
	return verify.Answer{Type: verify.QuestionChoice, Choice: choice, Probabilities: probabilities}
}
