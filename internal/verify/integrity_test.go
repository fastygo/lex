package verify

import "testing"

func TestRefutationIsDistinctFromAbsenceAndConflict(t *testing.T) {
	cases := []struct {
		name                                    string
		support, established, refuted, conflict float64
		action                                  string
		want                                    Verdict
	}{
		{"refuted", .1, .1, .9, .1, "reject", VerdictRejected},
		{"absent", .1, .1, .1, .1, "reject", VerdictInsufficient},
		{"positive", .9, .9, .1, .1, "proceed", VerdictValidated},
		{"contradictory-predicates", .9, .9, .9, .1, "proceed", VerdictConflict},
		{"conflicting-evidence", .1, .1, .9, .9, "reject", VerdictConflict},
		{"refutation-keeps-review", .1, .1, .9, .1, "manual_review", VerdictManualReview},
		{"refutation-blocks-proceed", .1, .1, .9, .1, "proceed", VerdictManualReview},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answers := passingAnswers()
			for id, value := range map[string]float64{"support": tc.support, "established": tc.established, "refuted": tc.refuted, "conflict": tc.conflict} {
				answers[id] = noulAnswer(value)
			}
			answers["action"] = choiceAnswer(tc.action)
			got, _, err := Interpret(claimQuestions(), testThresholds(), answers)
			if err != nil || got != tc.want {
				t.Fatalf("verdict=%s want=%s err=%v", got, tc.want, err)
			}
		})
	}
}

func TestChoiceRequiresAMaximumWithoutRewritingTies(t *testing.T) {
	for _, choice := range []string{"a", "b", "c"} {
		findings := ValidateAnswers(map[string]Question{"q": {Type: QuestionChoice, Choices: []string{"a", "b", "c"}}}, map[string]Answer{"q": {Type: QuestionChoice, Choice: choice, Probabilities: map[string]float64{"a": .5, "b": .5, "c": 0}}})
		if (len(findings) == 0) != (choice != "c") {
			t.Fatalf("choice=%s findings=%v", choice, findings)
		}
	}
}
