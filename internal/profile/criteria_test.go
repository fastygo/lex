package profile

import (
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

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
	const questionSetHashPin = "1a68013416ed51e5eec91c165eabd2d3c534a8fc3351d836a75b17fc26a41a6b"
	const policyHashPin = "9ebb4842264e82d65390762b94d10664c9d03445ad1d0550c6baf379a277ebcc"
	if QuestionSetHash() != questionSetHashPin || PolicyHash() != policyHashPin {
		t.Fatalf("question set hash = %s policy hash = %s", QuestionSetHash(), PolicyHash())
	}
}
