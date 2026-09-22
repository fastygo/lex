package claimvalidation

import (
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/profile"
)

func TestQuestionsNameTheFrozenState(t *testing.T) {
	if QuestionSetVersion != "0.2.0" {
		t.Fatalf("question set version = %s", QuestionSetVersion)
	}
	for id, raw := range Profile().ProviderQuestions() {
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
	state := Profile().State("the claim", []string{"surface"})
	if state["claim"] != "the claim" || state["evidence"] == nil {
		t.Fatalf("state = %#v", state)
	}
}

func TestCriteriaChangeProducesNewQuestionSetHash(t *testing.T) {
	changed := questionSet
	questions := make(map[string]profile.Question, len(questionSet.Questions))
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
	if hash == Profile().QuestionSetHash() || Profile().QuestionSetHash() == "" {
		t.Fatalf("question set hash = %s changed = %s", Profile().QuestionSetHash(), hash)
	}
	const questionSetHashPin = "b053e518f4aea6381d0cb4dd0597087406da262051325f1d0dff4536353fbca8"
	const policyHashPin = "367f5b8ee7e0eb1106756737aec56bd9471145c1a38ec3df3949beb64fea4669"
	if Profile().QuestionSetHash() != questionSetHashPin || Profile().PolicyHash() != policyHashPin {
		t.Fatalf("question set hash = %s policy hash = %s", Profile().QuestionSetHash(), Profile().PolicyHash())
	}
}

func TestRegistryResolvesTheBundleReference(t *testing.T) {
	registry := profile.MustRegistry(Profile())
	ref := profile.Ref{QuestionSetID: QuestionSetID, QuestionSetVersion: QuestionSetVersion, PolicyID: PolicyID, PolicyVersion: PolicyVersion}
	if _, ok := registry.Lookup(ref); !ok {
		t.Fatal("registry does not resolve the embedded profile")
	}
	ref.PolicyVersion = "0.1"
	if _, ok := registry.Lookup(ref); ok {
		t.Fatal("registry resolved a policy version it does not pin")
	}
	if _, err := profile.NewRegistry(Profile(), Profile()); err == nil {
		t.Fatal("registry accepted the same profile twice")
	}
}
