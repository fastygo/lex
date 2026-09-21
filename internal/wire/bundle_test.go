package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
)

func TestReplayAcceptsEscapedContextSnapshot(t *testing.T) {
	const text = "The account is A&B <locked>."
	request := contextmemory.PackRequest{
		ProjectID: "project-test", Query: "account",
		Focus: contextmemory.Focus{
			ID: profile.FocusID, Objective: "Select admissible source text.",
			RequiredTrustLevel: "project", Budget: contextmemory.Budget{MaxItems: 8, MaxChars: 65536},
		},
	}
	result, err := evidence.BuildPack(
		t.Context(),
		"project-test",
		[]contextmemory.Source{{
			SourceID: "source-1", Version: "v1", Text: text,
			TrustLevel: "project", EvidenceClass: "source_text",
		}},
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           result.ContextPack,
		Snapshot:       result.Snapshot,
		PackRequest:    request,
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
	report, err := Replay(raw)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "validated" {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestBuildBundleReplaysValidatedClaim(t *testing.T) {
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           addressablePack(),
		Snapshot:       addressableSnapshot(),
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
		Pack:           addressablePack(),
		Snapshot:       addressableSnapshot(),
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

func refreshPackHash(t *testing.T, bundle map[string]any) {
	t.Helper()
	contextBody := bundle["context"].(map[string]any)
	hash, err := canonical.HashValue(contextBody["pack"])
	if err != nil {
		t.Fatal(err)
	}
	contextBody["pack_hash"] = hash
	decision := bundle["decision_set"].(map[string]any)
	decision["context_pack_hash"] = hash
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

func TestReplayRejectsPackHashThatDoesNotMatchContent(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	pack := contextBody["pack"].(map[string]any)
	pack["note"] = "changed after sealing"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "binding_mismatch") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsEntityIdentityThatBreaksChecksum(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	entity := bundle["entity"].(map[string]any)
	entity["id"] = "claim-2"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "entity_checksum_mismatch") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsPackRequestIdentityMismatch(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	request := contextBody["pack_request"].(map[string]any)
	request["query"] = "other"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "pack_request_identity") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsSnapshotIdentityMismatch(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	snapshot := contextBody["snapshot"].(map[string]any)
	snapshot["id"] = "snapshot_0000000000000000000000000000000000000000000000000000000000000000"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "snapshot_identity") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsForeignAdapterVersion(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	decision := bundle["decision_set"].(map[string]any)
	decision["adapter_version"] = "9.9.9"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "unpinned_adapter") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsForeignVerifierVersion(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	bundle["verifier_version"] = "9.9.9"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "unpinned_verifier") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsUnsupportedProtocolVersion(t *testing.T) {
	raw := strings.Replace(string(sealedBundle(t)), `"protocol_version":"0.1-draft"`, `"protocol_version":"0.9-draft"`, 1)
	if _, err := Replay([]byte(raw)); err == nil {
		t.Fatal("accepted an unsupported protocol version")
	}
}

func TestReplayRejectsSurfaceThatBreaksChecksum(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	pack := contextBody["pack"].(map[string]any)
	items := pack["evidence_items"].([]any)
	item := items[0].(map[string]any)
	item["surface"] = "The account is open."
	refreshPackHash(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "checksum_mismatch") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

const evidenceText = "The account is locked."

func addressablePack() []byte {
	sum := sha256.Sum256([]byte(evidenceText))
	checksum := hex.EncodeToString(sum[:])
	snapshot := addressableSnapshot()
	request := map[string]any{"query": "account"}
	identifier, ok := expectedPackID(snapshot, request)
	if !ok {
		panic("pack identity")
	}
	return []byte(fmt.Sprintf(`{"id":%q,"evidence_items":[{"id":"chunk_0000","class":"source_text","trust_level":"project","surface":%q,"source_ref":{"project_id":"project-test","source_id":"source-1","span":{"start":0,"end":%d},"checksum":%q}}]}`, identifier, evidenceText, len(evidenceText), checksum))
}

func addressableSnapshot() map[string]any {
	snapshot := map[string]any{
		"project_id": "project-test", "runtime_version": "memory-exact-v1",
		"sources": []any{map[string]any{
			"source_id": "source-1", "version": "v1", "text": evidenceText,
			"trust_level": "project", "evidence_class": "source_text",
		}},
	}
	identifier, ok := expectedSnapshotID(snapshot)
	if !ok {
		panic("snapshot identity")
	}
	snapshot["id"] = identifier
	return snapshot
}

func TestReplayDoesNotUseNetwork(t *testing.T) {
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           addressablePack(),
		Snapshot:       addressableSnapshot(),
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
		Pack:           addressablePack(),
		Snapshot:       addressableSnapshot(),
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
