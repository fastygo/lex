package wire

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/verify"
)

func TestDecisionRequestFixturesValidate(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", ".project", "examples", "*-decision-request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("fixture count = %d, want 3", len(paths))
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
			var request struct {
				ProjectID string         `json:"project_id"`
				Context   *ContextRecord `json:"context"`
			}
			if err := json.Unmarshal(raw, &request); err != nil {
				t.Fatal(err)
			}
			if request.Context == nil {
				return
			}
			findings, err := CheckContext(context.Background(), request.ProjectID, *request.Context)
			if err != nil || len(findings) != 0 {
				t.Fatalf("context fixture does not reproduce: %v %v", findings, err)
			}
		})
	}
}

func fixtureSet() QuestionSet {
	return QuestionSet{
		ID: "agent.example", Version: "1",
		Questions: map[string]Question{
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

func fixtureAnswers() json.RawMessage {
	return json.RawMessage(`{
		"intent":{"type":"choice","choice":"portfolio","probabilities":{"products":0.2,"portfolio":0.8}},
		"needs_db":{"type":"noul","noul":0.9},
		"fit":{"type":"score","score":0.5}
	}`)
}

func fixtureVerifier() DecisionVerifier {
	return NewDecisionVerifier(map[string]AdapterPin{
		typesafe.AdapterID: {Version: typesafe.AdapterVersion, Model: typesafe.Model},
	})
}

func fixtureBundle(t *testing.T, mutate func(*DecisionBundleInput)) []byte {
	t.Helper()
	input := DecisionBundleInput{
		ProjectID: "project-test", Decision: DecisionIdentity{ID: "agent-step", Version: "1"},
		State:       json.RawMessage(`{"request":"choose a database","requirements":{"transactions":true}}`),
		QuestionSet: fixtureSet(),
		AdapterID:   typesafe.AdapterID, AdapterVersion: typesafe.AdapterVersion,
		ResolvedModel: typesafe.Model, Answers: fixtureAnswers(),
	}
	if mutate != nil {
		mutate(&input)
	}
	bundle, err := BuildDecisionBundle(input)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

func reseal(t *testing.T, bundle []byte, mutate func(*DecisionBundle)) []byte {
	t.Helper()
	document, err := DecodeDecisionBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	mutate(&document)
	resealed, err := sealDecisionBundle(document)
	if err != nil {
		t.Fatal(err)
	}
	return resealed
}

func frozenContext(t *testing.T, projectID string) ContextRecord {
	t.Helper()
	runtime, err := contextmemory.New(context.Background(), contextmemory.Config{
		ProjectID: projectID,
		Sources: []contextmemory.Source{{
			SourceID: "source-1", Version: "v1", Text: "The account is locked.",
			TrustLevel: "project", EvidenceClass: "source_text",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := contextmemory.PackRequest{
		ProjectID: projectID, Query: "account",
		Focus: contextmemory.Focus{
			ID: "example-focus", Objective: "Select source text.", RequiredTrustLevel: "project",
			Budget: contextmemory.Budget{MaxItems: 8, MaxChars: 65536},
		},
	}
	result, err := runtime.ContextPack(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, _ := json.Marshal(result.Snapshot)
	packRequest, _ := json.Marshal(request)
	hash, err := canonical.HashJSON(result.ContextPack)
	if err != nil {
		t.Fatal(err)
	}
	return ContextRecord{Pack: result.ContextPack, Snapshot: snapshot, PackRequest: packRequest, PackHash: hash}
}

func expectReport(t *testing.T, bundle []byte, status string, codes ...string) {
	t.Helper()
	report, err := fixtureVerifier().Replay(bundle)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		got = append(got, finding.Code)
	}
	if report.StructuralStatus != status || !reflect.DeepEqual(got, append([]string{}, codes...)) {
		t.Fatalf("status = %s findings = %v, want %s %v", report.StructuralStatus, got, status, codes)
	}
}

func TestDecisionBundleReplaysAllPrimitives(t *testing.T) {
	bundle := fixtureBundle(t, nil)
	if err := VerifyDecisionBundleHash(bundle); err != nil {
		t.Fatal(err)
	}
	expectReport(t, bundle, "valid")
}

func TestMalformedAnswerIsStructuralFindingNotVerdict(t *testing.T) {
	bundle := fixtureBundle(t, func(input *DecisionBundleInput) {
		input.Answers = json.RawMessage(`{
			"intent":{"type":"choice","choice":"portfolio","probabilities":{"products":0.9,"portfolio":0.1}},
			"needs_db":{"type":"noul","noul":0.9},
			"fit":{"type":"score","score":0.5}
		}`)
	})
	expectReport(t, bundle, "invalid", verify.CodePrefixInvalidChoice+":intent")
	if strings.Contains(string(bundle), `"verdict"`) {
		t.Fatal("bundle carries a semantic verdict")
	}
}

func TestCaseFoldedAnswerFieldIsRefusedNotMerged(t *testing.T) {
	bundle := fixtureBundle(t, func(input *DecisionBundleInput) {
		input.Answers = json.RawMessage(strings.Replace(string(fixtureAnswers()), `"noul":0.9`, `"noul":0.9,"NOUL":1`, 1))
	})
	expectReport(t, bundle, "invalid", verify.CodeInvalidAnswers)
}

func TestSealedStateTamperingBreaksTheBundleHash(t *testing.T) {
	bundle := fixtureBundle(t, nil)
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

func TestDecisionRequestRejectsAmbiguousOrInvalidPrimitives(t *testing.T) {
	cases := []string{
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"noul","instructions":"x","levels":["wrong","domain"]}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"choice","instructions":"x","options":{"same":"one"}}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"score","instructions":"x","levels":["same","same"]}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":[],"question_set":{"id":"set","version":"1","questions":{"q":{"type":"noul","instructions":"x"}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"bad id":{"type":"noul","instructions":"x"}}}}`,
		`{"project_id":"project-test","decision":{"id":"step","version":"1"},"state":{},"question_set":{"id":"set","version":"1","questions":{"q":{"type":"noul","instructions":"x"}}},"policy":{"threshold":0}}`,
	}
	for _, raw := range cases {
		if err := ValidateDecisionRequest([]byte(raw)); err == nil {
			t.Fatalf("invalid request passed: %s", raw)
		}
	}
}

func TestCriteriaChangeProducesNewQuestionSetHash(t *testing.T) {
	first, err := questionSetHash(fixtureSet())
	if err != nil {
		t.Fatal(err)
	}
	changed := fixtureSet()
	intent := changed.Questions["intent"]
	intent.Options = map[string]string{"products": "Products", "portfolio": "Portfolio of work"}
	changed.Questions["intent"] = intent
	second, err := questionSetHash(changed)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("changed criteria kept the QuestionSet hash")
	}
}

func TestReplayRejectsUnpinnedModelAdapterAndVerifier(t *testing.T) {
	bundle := fixtureBundle(t, nil)
	cases := []struct {
		name   string
		mutate func(*DecisionBundle)
		code   string
	}{
		{"substituted model", func(b *DecisionBundle) { b.DecisionSet.ResolvedModel = "jev-1.13.1" }, verify.CodeUnresolvedModel},
		{"model alias", func(b *DecisionBundle) { b.DecisionSet.ResolvedModel = "latest" }, verify.CodeUnresolvedModel},
		{"foreign adapter version", func(b *DecisionBundle) { b.DecisionSet.AdapterVersion = "9.9.9" }, verify.CodeUnpinnedAdapter},
		{"unknown adapter", func(b *DecisionBundle) { b.DecisionSet.AdapterID = "other-adapter" }, verify.CodeUnknownAdapter},
		{"foreign verifier", func(b *DecisionBundle) { b.VerifierVersion = "9.9.9" }, verify.CodeUnpinnedVerifier},
		{"decision identity", func(b *DecisionBundle) { b.Decision.Version = "2" }, verify.CodeDecisionChecksumMismatch},
		{"question set hash", func(b *DecisionBundle) { b.QuestionSet.Version = "2" }, verify.CodeUnpinnedQuestionSet},
		{"state hash", func(b *DecisionBundle) { b.DecisionSet.StateHash = strings.Repeat("0", 64) }, verify.CodeBindingMismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expectReport(t, reseal(t, bundle, tc.mutate), "invalid", tc.code)
		})
	}
}

func TestContextBindingIsVerifiedByContextRebuild(t *testing.T) {
	record := frozenContext(t, "project-test")
	bundle := fixtureBundle(t, func(input *DecisionBundleInput) { input.Context = &record })
	expectReport(t, bundle, "valid")

	foreign := frozenContext(t, "project-other")
	expectReport(t, fixtureBundle(t, func(input *DecisionBundleInput) { input.Context = &foreign }), "invalid", verify.CodeProjectBinding)

	expectReport(t, reseal(t, bundle, func(b *DecisionBundle) {
		b.Context.Snapshot = json.RawMessage(strings.Replace(string(b.Context.Snapshot), `"id":"`, `"note":"x","id":"`, 1))
	}), "invalid", verify.CodePackShape)

	expectReport(t, reseal(t, bundle, func(b *DecisionBundle) {
		b.Context.Snapshot = json.RawMessage(strings.Replace(string(b.Context.Snapshot), "The account is locked.", "The account is open.", 1))
	}), "invalid", verify.CodeSnapshotIdentity)

	expectReport(t, reseal(t, bundle, func(b *DecisionBundle) {
		b.Context.PackRequest = json.RawMessage(strings.Replace(string(b.Context.PackRequest), `"query":"account"`, `"query":"locked"`, 1))
	}), "invalid", verify.CodePackRebuild)

	expectReport(t, reseal(t, bundle, func(b *DecisionBundle) {
		b.Context.PackHash = strings.Repeat("0", 64)
	}), "invalid", verify.CodeBindingMismatch)
}

func TestReplayLeavesTheSavedBundleUnchanged(t *testing.T) {
	record := frozenContext(t, "project-test")
	bundle := fixtureBundle(t, func(input *DecisionBundleInput) { input.Context = &record })
	saved := append([]byte(nil), bundle...)
	expectReport(t, bundle, "valid")
	if string(saved) != string(bundle) {
		t.Fatal("replay rewrote the caller bundle")
	}
}

func TestReplayDoesNotUseNetwork(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("replay dialed the network")
		return nil, errors.New("network disabled")
	})
	defer func() { http.DefaultTransport = previous }()
	record := frozenContext(t, "project-test")
	expectReport(t, fixtureBundle(t, func(input *DecisionBundleInput) { input.Context = &record }), "valid")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func TestCanceledReplayIsNotReportedAsPackMismatch(t *testing.T) {
	record := frozenContext(t, "project-test")
	bundle := fixtureBundle(t, func(input *DecisionBundleInput) { input.Context = &record })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fixtureVerifier().ReplayContext(ctx, bundle); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDecisionBundleTypeMirrorsSchema(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("schema", "decision-bundle.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := value.(map[string]any)
	properties := schema["properties"].(map[string]any)
	definitions := schema["$defs"].(map[string]any)

	expectMirror(t, "decision bundle", reflect.TypeOf(DecisionBundle{}), keys(properties))
	expectMirror(t, "decision identity", reflect.TypeOf(DecisionIdentityRecord{}), keys(properties["decision"].(map[string]any)["properties"].(map[string]any)))
	expectMirror(t, "decision state", reflect.TypeOf(DecisionStateRecord{}), keys(properties["state"].(map[string]any)["properties"].(map[string]any)))
	expectMirror(t, "question set", reflect.TypeOf(QuestionSetRecord{}), keys(definitions["questionSet"].(map[string]any)["properties"].(map[string]any)))
	expectMirror(t, "decision set", reflect.TypeOf(DecisionRecord{}), keys(properties["decision_set"].(map[string]any)["properties"].(map[string]any)))
}

func expectMirror(t *testing.T, name string, typ reflect.Type, want []string) {
	t.Helper()
	got := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		got = append(got, strings.Split(typ.Field(i).Tag.Get("json"), ",")[0])
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: type fields %v, schema properties %v", name, got, want)
	}
}

func keys[V any](object map[string]V) []string {
	out := make([]string, 0, len(object))
	for key := range object {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
