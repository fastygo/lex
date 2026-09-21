package wire

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

func TestBuildBundleReplaysValidatedClaim(t *testing.T) {
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           []byte(`{"evidence_items":[{"id":"chunk_0000","class":"source_text","trust_level":"project"}]}`),
		Snapshot:       map[string]any{"id": "snapshot-1"},
		PackRequest:    map[string]any{"query": "account"},
		AdapterID:      "direct-systemone",
		AdapterVersion: "0.1.0",
		ResolvedModel:  "fixture-v1",
		Answers: []byte(`{
			"support":{"type":"noul","noul":0.9},
			"established":{"type":"noul","noul":0.9},
			"conflict":{"type":"noul","noul":0.1},
			"safe_to_auto_act":{"type":"noul","noul":0.9},
			"action":{"type":"choice","choice":"proceed","probabilities":{"proceed":1,"reject":0,"manual_review":0,"other":0}}
		}`),
	})
	if err != nil {
		t.Fatalf("BuildBundle() error = %v", err)
	}
	report, err := Replay(raw)
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if report.Verdict != "validated" {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsCompoundQuestionEvenWhenSelfHashMatches(t *testing.T) {
	raw := sealedBundle(t)
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	questionSet := bundle["question_set"].(map[string]any)
	questions := questionSet["questions"].(map[string]any)
	action := questions["action"].(map[string]any)
	action["instructions"] = "Does evidence support the claim and authorize proceeding?"
	questions["support_and_safety"] = map[string]any{
		"type":         "noul",
		"instructions": "Does support overlap safety for the same claim?",
	}
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "unpinned_question_set") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func sealedBundle(t *testing.T) []byte {
	t.Helper()
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           []byte(`{"evidence_items":[{"id":"chunk_0000","class":"source_text","trust_level":"project"}]}`),
		Snapshot:       map[string]any{"id": "snapshot-1"},
		PackRequest:    map[string]any{"query": "account"},
		AdapterID:      "direct-systemone",
		AdapterVersion: "0.1.0",
		ResolvedModel:  "fixture-v1",
		Answers: []byte(`{
			"support":{"type":"noul","noul":0.9},
			"established":{"type":"noul","noul":0.9},
			"conflict":{"type":"noul","noul":0.1},
			"safe_to_auto_act":{"type":"noul","noul":0.9},
			"action":{"type":"choice","choice":"proceed","probabilities":{"proceed":1,"reject":0,"manual_review":0,"other":0}}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func reseal(t *testing.T, bundle map[string]any) []byte {
	t.Helper()
	delete(bundle, "bundle_hash")
	hash, err := canonical.HashValue(bundle)
	if err != nil {
		t.Fatal(err)
	}
	bundle["bundle_hash"] = hash
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func hasFinding(report Report, code string) bool {
	for _, finding := range report.Findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestReplayDoesNotUseNetwork(t *testing.T) {
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           []byte(`{"evidence_items":[{"id":"chunk_0000","class":"source_text","trust_level":"project"}]}`),
		Snapshot:       map[string]any{"id": "snapshot-1"},
		PackRequest:    map[string]any{"query": "account"},
		AdapterID:      "direct-systemone",
		AdapterVersion: "0.1.0",
		ResolvedModel:  "fixture-v1",
		Answers: []byte(`{
			"support":{"type":"noul","noul":0.9},
			"established":{"type":"noul","noul":0.9},
			"conflict":{"type":"noul","noul":0.1},
			"safe_to_auto_act":{"type":"noul","noul":0.9},
			"action":{"type":"choice","choice":"proceed","probabilities":{"proceed":1,"reject":0,"manual_review":0,"other":0}}
		}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripper(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network disabled")
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	report, err := Replay(raw)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "validated" {
		t.Fatalf("verdict = %s", report.Verdict)
	}
}

type roundTripper func(*http.Request) (*http.Response, error)

func (transport roundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestReplayRejectsUnpinnedPolicyWithoutProviderCall(t *testing.T) {
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           []byte(`{"evidence_items":[{"id":"chunk_0000","class":"source_text","trust_level":"project"}]}`),
		Snapshot:       map[string]any{"id": "snapshot-1"},
		PackRequest:    map[string]any{"query": "account"},
		AdapterID:      "hosted-systemone",
		AdapterVersion: "0.1.0",
		ResolvedModel:  "fixture-v1",
		Answers:        []byte(`{"support":{"type":"noul","noul":0.9}}`),
	})
	if err != nil {
		t.Fatalf("BuildBundle() error = %v", err)
	}
	tampered := strings.Replace(string(raw), `"policy":{"hash":`, `"policy":{"hash":"0000000000000000000000000000000000000000000000000000000000000000","ignored":`, 1)
	if _, err := Replay([]byte(tampered)); err == nil {
		t.Fatal("expected tampered bundle to fail structural validation")
	}
}

func TestReplayRefusesInferenceOnlyEvidence(t *testing.T) {
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           []byte(`{"evidence_items":[{"id":"chunk_0000","class":"model_inference","trust_level":"project","surface":"The model says the claim is true."}]}`),
		Snapshot:       map[string]any{"id": "snapshot-1"},
		PackRequest:    map[string]any{"query": "claim"},
		AdapterID:      "direct-systemone",
		AdapterVersion: "0.1.0",
		ResolvedModel:  "fixture-v1",
		Answers: []byte(`{
			"support":{"type":"noul","noul":0.99},
			"established":{"type":"noul","noul":0.99},
			"conflict":{"type":"noul","noul":0.01},
			"safe_to_auto_act":{"type":"noul","noul":0.99},
			"action":{"type":"choice","choice":"proceed","probabilities":{"proceed":1,"reject":0,"manual_review":0,"other":0}}
		}`),
	})
	if err != nil {
		t.Fatalf("BuildBundle() error = %v", err)
	}
	report, err := Replay(raw)
	if err != nil {
		t.Fatalf("Replay() error = %v", err)
	}
	if report.Verdict != "insufficient" {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
	found := false
	for _, finding := range report.Findings {
		if finding.Code == "inference_only" {
			found = true
		}
	}
	if !found {
		t.Fatalf("findings = %#v", report.Findings)
	}
}
