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
