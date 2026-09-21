package profile

import (
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

func TestReproducibleModelPinsAdapterIdentity(t *testing.T) {
	if !ReproducibleModel("direct-systemone", DirectModel) {
		t.Fatal("direct model was rejected")
	}
	if !ReproducibleModel("hosted-systemone", HostedModel+"-20260901") || !ReproducibleModel("hosted-systemone", HostedModel+".0") {
		t.Fatal("resolved hosted model was rejected")
	}
	rejected := []struct{ adapter, model string }{
		{"direct-systemone", "jev-latest"},
		{"direct-systemone", "jev-1.14.0"},
		{"hosted-systemone", HostedModel},
		{"hosted-systemone", "typesafe/jev-1.130"},
		{"hosted-systemone", HostedModel + "-latest"},
		{"hosted-systemone", ""},
	}
	for _, tc := range rejected {
		if ReproducibleModel(tc.adapter, tc.model) {
			t.Fatalf("accepted %s %q", tc.adapter, tc.model)
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
	const questionSetHashPin = "1a68013416ed51e5eec91c165eabd2d3c534a8fc3351d836a75b17fc26a41a6b"
	const policyHashPin = "9ebb4842264e82d65390762b94d10664c9d03445ad1d0550c6baf379a277ebcc"
	if QuestionSetHash() != questionSetHashPin || PolicyHash() != policyHashPin {
		t.Fatalf("question set hash = %s policy hash = %s", QuestionSetHash(), PolicyHash())
	}
}
