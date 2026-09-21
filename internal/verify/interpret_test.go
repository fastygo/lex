package verify

import "testing"

func TestInterpretKeepsConflictAheadOfInsufficient(t *testing.T) {
	verdict, findings, err := Interpret(claimQuestions(), Thresholds{
		SupportMin: 0.7, EstablishMin: 0.8, ConflictMin: 0.5, SafetyMin: 0.8,
	}, map[string]Answer{
		"support":          noulAnswer(0.2),
		"established":      noulAnswer(0.2),
		"conflict":         noulAnswer(0.9),
		"safe_to_auto_act": noulAnswer(0.1),
		"action":           choiceAnswer("manual_review"),
	})
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if verdict != VerdictConflict {
		t.Fatalf("verdict = %s", verdict)
	}
	if len(findings) < 2 {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestInterpretAcceptsConsistentClaim(t *testing.T) {
	verdict, findings, err := Interpret(claimQuestions(), Thresholds{
		SupportMin: 0.7, EstablishMin: 0.8, ConflictMin: 0.5, SafetyMin: 0.8,
	}, passingAnswers())
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if verdict != VerdictValidated || len(findings) != 0 {
		t.Fatalf("verdict = %s findings = %#v", verdict, findings)
	}
}

func TestInterpretRejectsEstablishedNegativeClaim(t *testing.T) {
	answers := passingAnswers()
	answers["support"] = noulAnswer(0.1)
	answers["action"] = choiceAnswer("reject")
	verdict, _, err := Interpret(claimQuestions(), testThresholds(), answers)
	if err != nil {
		t.Fatalf("Interpret() error = %v", err)
	}
	if verdict != VerdictRejected {
		t.Fatalf("verdict = %s", verdict)
	}
}

func TestInterpretCoversEveryVerdict(t *testing.T) {
	thresholds := testThresholds()
	questions := claimQuestions()
	cases := []struct {
		name    string
		answers map[string]Answer
		want    Verdict
	}{
		{name: "validated", answers: passingAnswers(), want: VerdictValidated},
		{name: "rejected", answers: withChoice(withNoul(passingAnswers(), "support", 0.1), "reject"), want: VerdictRejected},
		{name: "insufficient", answers: withNoul(passingAnswers(), "established", 0.1), want: VerdictInsufficient},
		{name: "conflict", answers: withNoul(passingAnswers(), "conflict", 0.9), want: VerdictConflict},
		{name: "manual_review", answers: withChoice(passingAnswers(), "manual_review"), want: VerdictManualReview},
		{name: "error", answers: map[string]Answer{}, want: VerdictError},
	}
	seen := map[Verdict]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			verdict, _, err := Interpret(questions, thresholds, tc.answers)
			if err != nil {
				t.Fatal(err)
			}
			if verdict != tc.want {
				t.Fatalf("verdict = %s, want %s", verdict, tc.want)
			}
		})
		seen[tc.want] = true
	}
	for _, verdict := range []Verdict{VerdictValidated, VerdictRejected, VerdictInsufficient, VerdictConflict, VerdictManualReview, VerdictError} {
		if !seen[verdict] {
			t.Fatalf("missing verdict %s", verdict)
		}
	}
}

func testThresholds() Thresholds {
	return Thresholds{SupportMin: 0.7, EstablishMin: 0.8, ConflictMin: 0.5, SafetyMin: 0.8}
}

func withNoul(answers map[string]Answer, id string, value float64) map[string]Answer {
	answers[id] = noulAnswer(value)
	return answers
}

func withChoice(answers map[string]Answer, choice string) map[string]Answer {
	answers["action"] = choiceAnswer(choice)
	return answers
}

func claimQuestions() map[string]Question {
	return map[string]Question{
		"support":          {Type: QuestionNoul},
		"established":      {Type: QuestionNoul},
		"conflict":         {Type: QuestionNoul},
		"safe_to_auto_act": {Type: QuestionNoul},
		"action":           {Type: QuestionChoice, Choices: []string{"proceed", "reject", "manual_review", "other"}},
	}
}

func passingAnswers() map[string]Answer {
	return map[string]Answer{
		"support":          noulAnswer(0.9),
		"established":      noulAnswer(0.9),
		"conflict":         noulAnswer(0.1),
		"safe_to_auto_act": noulAnswer(0.9),
		"action":           choiceAnswer("proceed"),
	}
}

func noulAnswer(value float64) Answer {
	return Answer{Type: QuestionNoul, Noul: &value}
}

func choiceAnswer(choice string) Answer {
	probabilities := map[string]float64{"proceed": 0, "reject": 0, "manual_review": 0, "other": 0}
	probabilities[choice] = 1
	return Answer{Type: QuestionChoice, Choice: choice, Probabilities: probabilities}
}
