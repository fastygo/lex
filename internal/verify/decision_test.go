package verify

import "testing"

func TestValidateAnswersAcceptsTypedAnswers(t *testing.T) {
	noul := 0.9
	score := 1.25
	findings := ValidateAnswers(
		map[string]Question{
			"support": {Type: QuestionNoul},
			"action":  {Type: QuestionChoice, Choices: []string{"accept", "manual_review"}},
			"score":   {Type: QuestionScore, Levels: 3},
		},
		map[string]Answer{
			"support": {Type: QuestionNoul, Noul: &noul},
			"action":  {Type: QuestionChoice, Choice: "accept", Probabilities: map[string]float64{"accept": 0.8, "manual_review": 0.2}},
			"score":   {Type: QuestionScore, Score: &score},
		},
	)
	if len(findings) != 0 {
		t.Fatalf("findings = %#v, want none", findings)
	}
}

func TestValidateAnswersReportsMalformedAnswers(t *testing.T) {
	noul := 1.1
	findings := ValidateAnswers(
		map[string]Question{"support": {Type: QuestionNoul}},
		map[string]Answer{"support": {Type: QuestionNoul, Noul: &noul}, "extra": {Type: QuestionNoul, Noul: &noul}},
	)
	if len(findings) != 2 {
		t.Fatalf("finding count = %d, want 2", len(findings))
	}
	for _, finding := range findings {
		if finding.Verdict != VerdictError {
			t.Fatalf("finding verdict = %q, want error", finding.Verdict)
		}
	}
}

func TestValidateAnswersOrdersFindingsByIdentifier(t *testing.T) {
	questions := map[string]Question{
		"support":     {Type: QuestionNoul},
		"established": {Type: QuestionNoul},
	}
	answers := map[string]Answer{
		"zebra": {Type: QuestionNoul},
		"extra": {Type: QuestionNoul},
	}
	var first []string
	for range 32 {
		findings := ValidateAnswers(questions, answers)
		got := make([]string, len(findings))
		for i, finding := range findings {
			got[i] = finding.Code
		}
		if first == nil {
			first = got
			continue
		}
		if len(got) != len(first) {
			t.Fatalf("finding count changed: %v", got)
		}
		for i := range got {
			if got[i] != first[i] {
				t.Fatalf("finding order changed: %v", got)
			}
		}
	}
	want := []string{
		"missing_answer:established",
		"missing_answer:support",
		"unexpected_answer:extra",
		"unexpected_answer:zebra",
	}
	if len(first) != len(want) {
		t.Fatalf("findings = %v", first)
	}
	for i := range want {
		if first[i] != want[i] {
			t.Fatalf("findings = %v", first)
		}
	}
}
