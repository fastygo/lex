package claimvalidation

import (
	"errors"
	"math"
	"testing"

	"github.com/fastygo/lex/internal/verify"
)

func TestInterpretKeepsConflictAheadOfInsufficient(t *testing.T) {
	verdict, findings, err := Profile().Interpret(map[string]verify.Answer{
		"support":          noulAnswer(0.2),
		"established":      noulAnswer(0.2),
		"refuted":          noulAnswer(0.1),
		"conflict":         noulAnswer(0.9),
		"safe_to_auto_act": noulAnswer(0.1),
		"action":           choiceAnswer("manual_review"),
	})
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if verdict != verify.VerdictConflict {
		t.Fatalf("verdict = %s", verdict)
	}
	if len(findings) < 2 {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestInterpretAcceptsConsistentClaim(t *testing.T) {
	verdict, findings, err := Profile().Interpret(passingAnswers())
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if verdict != verify.VerdictValidated || len(findings) != 0 {
		t.Fatalf("verdict = %s findings = %#v", verdict, findings)
	}
}

func TestInterpretRejectsEstablishedNegativeClaim(t *testing.T) {
	answers := passingAnswers()
	answers["support"] = noulAnswer(0.1)
	answers["established"] = noulAnswer(0.1)
	answers["refuted"] = noulAnswer(0.9)
	answers["action"] = choiceAnswer("reject")
	verdict, _, err := Profile().Interpret(answers)
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if verdict != verify.VerdictRejected {
		t.Fatalf("verdict = %s", verdict)
	}
}

func TestInterpretCoversEveryVerdict(t *testing.T) {
	cases := []struct {
		name    string
		answers map[string]verify.Answer
		want    verify.Verdict
	}{
		{name: "validated", answers: passingAnswers(), want: verify.VerdictValidated},
		{name: "rejected", answers: withChoice(withNoul(withNoul(withNoul(passingAnswers(), "support", 0.1), "established", 0.1), "refuted", 0.9), "reject"), want: verify.VerdictRejected},
		{name: "insufficient", answers: withNoul(passingAnswers(), "established", 0.1), want: verify.VerdictInsufficient},
		{name: "conflict", answers: withNoul(passingAnswers(), "conflict", 0.9), want: verify.VerdictConflict},
		{name: "manual_review", answers: withChoice(passingAnswers(), "manual_review"), want: verify.VerdictManualReview},
		{name: "error", answers: map[string]verify.Answer{}, want: verify.VerdictError},
	}
	seen := map[verify.Verdict]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, _, err := Profile().Interpret(tc.answers)
			if err != nil {
				t.Fatal(err)
			}
			if verdict != tc.want {
				t.Fatalf("verdict = %s, want %s", verdict, tc.want)
			}
		})
		seen[tc.want] = true
	}
	for _, verdict := range []verify.Verdict{verify.VerdictValidated, verify.VerdictRejected, verify.VerdictInsufficient, verify.VerdictConflict, verify.VerdictManualReview, verify.VerdictError} {
		if !seen[verdict] {
			t.Fatalf("missing verdict %s", verdict)
		}
	}
}

func TestInterpretKeepsSafetyGateBesideConflict(t *testing.T) {
	answers := withNoul(passingAnswers(), "conflict", 0.9)
	answers = withNoul(answers, "safe_to_auto_act", 0.1)
	verdict, findings, err := Profile().Interpret(answers)
	if err != nil {
		t.Fatal(err)
	}
	if verdict != verify.VerdictConflict {
		t.Fatalf("verdict = %s", verdict)
	}
	codes := map[string]bool{}
	for _, finding := range findings {
		codes[finding.Code] = true
	}
	if !codes["evidence_conflict"] || !codes["safety_gate"] {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestInterpretRejectsUnusablePolicy(t *testing.T) {
	gate := Gate{SupportMin: math.NaN(), EstablishMin: 0.8, RefuteMin: 0.8, ConflictMin: 0.5, SafetyMin: 0.8}
	verdict, findings, err := verify.Interpret(Profile().TypedQuestions(), gate, passingAnswers())
	if verdict != "" || findings != nil || !errors.Is(err, verify.ErrUnusablePolicy) {
		t.Fatalf("verdict = %s findings = %#v err = %v", verdict, findings, err)
	}
}

func TestRefutationIsDistinctFromAbsenceAndConflict(t *testing.T) {
	cases := []struct {
		name                                    string
		support, established, refuted, conflict float64
		action                                  string
		want                                    verify.Verdict
	}{
		{"refuted", .1, .1, .9, .1, "reject", verify.VerdictRejected},
		{"absent", .1, .1, .1, .1, "reject", verify.VerdictInsufficient},
		{"positive", .9, .9, .1, .1, "proceed", verify.VerdictValidated},
		{"contradictory-predicates", .9, .9, .9, .1, "proceed", verify.VerdictConflict},
		{"conflicting-evidence", .1, .1, .9, .9, "reject", verify.VerdictConflict},
		{"refutation-keeps-review", .1, .1, .9, .1, "manual_review", verify.VerdictManualReview},
		{"refutation-blocks-proceed", .1, .1, .9, .1, "proceed", verify.VerdictManualReview},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answers := passingAnswers()
			for id, value := range map[string]float64{"support": tc.support, "established": tc.established, "refuted": tc.refuted, "conflict": tc.conflict} {
				answers[id] = noulAnswer(value)
			}
			answers["action"] = choiceAnswer(tc.action)
			got, _, err := Profile().Interpret(answers)
			if err != nil || got != tc.want {
				t.Fatalf("verdict=%s want=%s err=%v", got, tc.want, err)
			}
		})
	}
}

func withNoul(answers map[string]verify.Answer, id string, value float64) map[string]verify.Answer {
	answers[id] = noulAnswer(value)
	return answers
}

func withChoice(answers map[string]verify.Answer, choice string) map[string]verify.Answer {
	answers["action"] = choiceAnswer(choice)
	return answers
}

func passingAnswers() map[string]verify.Answer {
	return map[string]verify.Answer{
		"support":          noulAnswer(0.9),
		"established":      noulAnswer(0.9),
		"refuted":          noulAnswer(0.1),
		"conflict":         noulAnswer(0.1),
		"safe_to_auto_act": noulAnswer(0.9),
		"action":           choiceAnswer("proceed"),
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
