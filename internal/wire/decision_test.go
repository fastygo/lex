package wire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/verify"
)

func TestGenericDecisionRequestFixturesValidate(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", ".project", ".jev", "generic", "*-decision-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 2 {
		t.Fatalf("generic fixture count = %d, want 2", len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := ValidateDecisionRequest(raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func genericFixtureSet() GenericQuestionSet {
	return GenericQuestionSet{
		ID: "agent.example", Version: "1",
		Questions: map[string]GenericQuestion{
			"intent": {
				Type: verify.QuestionChoice, Instructions: "Which intent applies?",
				Options: map[string]string{"products": "Products", "portfolio": "Portfolio"},
			},
			"needs_db": {Type: verify.QuestionNoul, Instructions: "Does the state need a database?"},
			"fit": {
				Type: verify.QuestionScore, Instructions: "How strong is the fit?",
				Levels: []string{"weak", "strong"},
			},
		},
	}
}

func genericFixtureAnswers() json.RawMessage {
	return json.RawMessage(`{
		"intent":{"type":"choice","choice":"portfolio","probabilities":{"products":0.2,"portfolio":0.8}},
		"needs_db":{"type":"noul","noul":0.9},
		"fit":{"type":"score","score":0.5}
	}`)
}

func TestGenericDecisionBundleReplaysAllPrimitives(t *testing.T) {
	bundle, err := BuildDecisionBundle(DecisionBundleInput{
		ProjectID: "project-test", Decision: DecisionIdentity{ID: "agent-step", Version: "1"},
		State:       json.RawMessage(`{"request":"choose a database","requirements":{"transactions":true}}`),
		QuestionSet: genericFixtureSet(),
		AdapterID:   typesafe.AdapterID, AdapterVersion: typesafe.AdapterVersion,
		ResolvedModel: typesafe.Model, Answers: genericFixtureAnswers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDecisionBundleHash(bundle); err != nil {
		t.Fatal(err)
	}
	report, err := NewDecisionVerifier(map[string]AdapterPin{
		typesafe.AdapterID: {Version: typesafe.AdapterVersion, Model: typesafe.Model},
	}).Replay(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if report.StructuralStatus != "valid" || len(report.Findings) != 0 {
		t.Fatalf("report = %#v", report)
	}
}

func TestGenericDecisionBundleRejectsMalformedAnswerWithoutSemanticVerdict(t *testing.T) {
	answers := genericFixtureAnswers()
	answers = json.RawMessage(`{
		"intent":{"type":"choice","choice":"portfolio","probabilities":{"products":0.9,"portfolio":0.1}},
		"needs_db":{"type":"noul","noul":0.9},
		"fit":{"type":"score","score":0.5}
	}`)
	bundle, err := BuildDecisionBundle(DecisionBundleInput{
		ProjectID: "project-test", Decision: DecisionIdentity{ID: "agent-step", Version: "1"},
		State: json.RawMessage(`{"request":"choose a database"}`), QuestionSet: genericFixtureSet(),
		AdapterID: typesafe.AdapterID, AdapterVersion: typesafe.AdapterVersion,
		ResolvedModel: typesafe.Model, Answers: answers,
	})
	if err != nil {
		t.Fatal(err)
	}
	report, err := NewDecisionVerifier(map[string]AdapterPin{
		typesafe.AdapterID: {Version: typesafe.AdapterVersion, Model: typesafe.Model},
	}).Replay(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if report.StructuralStatus != "invalid" || len(report.Findings) == 0 || report.Findings[0].Code != verify.CodePrefixInvalidChoice+":intent" {
		t.Fatalf("report = %#v", report)
	}
}

func TestGenericDecisionBundleDetectsStateTampering(t *testing.T) {
	bundle, err := BuildDecisionBundle(DecisionBundleInput{
		ProjectID: "project-test", Decision: DecisionIdentity{ID: "agent-step", Version: "1"},
		State: json.RawMessage(`{"request":"original"}`), QuestionSet: genericFixtureSet(),
		AdapterID: typesafe.AdapterID, AdapterVersion: typesafe.AdapterVersion,
		ResolvedModel: typesafe.Model, Answers: genericFixtureAnswers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(bundle, &document); err != nil {
		t.Fatal(err)
	}
	document["state"].(map[string]any)["value"] = map[string]any{"request": "changed"}
	tampered, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyDecisionBundleHash(tampered); err == nil {
		t.Fatal("tampered state passed bundle verification")
	}
}

func TestGenericDecisionRequestRejectsAmbiguousOrInvalidPrimitives(t *testing.T) {
	cases := []string{
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"noul","instructions":"x","levels":["wrong","domain"]}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"choice","instructions":"x","options":{"same":"one"}}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"score","instructions":"x","levels":["same","same"]}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":[],"question_set":{"id":"set","version":"1","questions":{"q":{"type":"noul","instructions":"x"}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"bad id":{"type":"noul","instructions":"x"}}}}`,
	}
	for _, raw := range cases {
		if err := ValidateDecisionRequest([]byte(raw)); err == nil {
			t.Fatalf("invalid request passed: %s", raw)
		}
	}
}

func TestGenericDecisionReplayRejectsSubstitutedModel(t *testing.T) {
	bundle, err := BuildDecisionBundle(DecisionBundleInput{
		ProjectID: "project-test", Decision: DecisionIdentity{ID: "agent-step", Version: "1"},
		State: json.RawMessage(`{"request":"original"}`), QuestionSet: genericFixtureSet(),
		AdapterID: typesafe.AdapterID, AdapterVersion: typesafe.AdapterVersion,
		ResolvedModel: typesafe.Model, Answers: genericFixtureAnswers(),
	})
	if err != nil {
		t.Fatal(err)
	}
	document, err := DecodeDecisionBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	document.DecisionSet.ResolvedModel = "jev-1.13.1"
	document.BundleHash = ""
	resigned, err := sealDecisionBundle(document)
	if err != nil {
		t.Fatal(err)
	}
	report, err := NewDecisionVerifier(map[string]AdapterPin{
		typesafe.AdapterID: {Version: typesafe.AdapterVersion, Model: typesafe.Model},
	}).Replay(resigned)
	if err != nil {
		t.Fatal(err)
	}
	if report.StructuralStatus != "invalid" || len(report.Findings) != 1 || report.Findings[0].Code != verify.CodeUnresolvedModel {
		t.Fatalf("report = %#v", report)
	}
}
