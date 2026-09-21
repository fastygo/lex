package profile

import (
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

func TestQuestionsNameTheFrozenState(t *testing.T) {
	if QuestionSetVersion != "0.2.0" {
		t.Fatalf("question set version = %s", QuestionSetVersion)
	}
	for id, raw := range ProviderQuestions() {
		question, _ := raw.(map[string]any)
		instructions, _ := question["instructions"].(string)
		if !strings.Contains(instructions, "`claim`") || !strings.Contains(instructions, "`evidence`") {
			t.Fatalf("%s instructions = %q", id, instructions)
		}
		if question["type"] == "noul" {
			criteria, _ := question["criteria"].(map[string]any)
			if criteria["true"] == nil || criteria["false"] == nil {
				t.Fatalf("%s criteria = %#v", id, criteria)
			}
		}
	}
}

func TestCriteriaChangeProducesNewQuestionSetHash(t *testing.T) {
	changed := questionSet
	questions := make(map[string]question, len(questionSet.Questions))
	for id, item := range questionSet.Questions {
		questions[id] = item
	}
	support := questions[questionSupport]
	support.Instructions = support.Instructions + " extra criterion"
	questions[questionSupport] = support
	changed.Questions = questions

	hash, err := canonical.HashValue(changed)
	if err != nil {
		t.Fatal(err)
	}
	if hash == QuestionSetHash() || QuestionSetHash() == "" {
		t.Fatalf("question set hash = %s changed = %s", QuestionSetHash(), hash)
	}
	const questionSetHashPin = "b053e518f4aea6381d0cb4dd0597087406da262051325f1d0dff4536353fbca8"
	const policyHashPin = "367f5b8ee7e0eb1106756737aec56bd9471145c1a38ec3df3949beb64fea4669"
	if QuestionSetHash() != questionSetHashPin || PolicyHash() != policyHashPin {
		t.Fatalf("question set hash = %s policy hash = %s", QuestionSetHash(), PolicyHash())
	}
}
