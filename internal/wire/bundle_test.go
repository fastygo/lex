package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
)

func TestReplayAcceptsExactPhraseSelection(t *testing.T) {
	request := contextmemory.PackRequest{
		ProjectID: "project-test", Query: "account",
		Focus: contextmemory.Focus{
			ID: profile.FocusID, Objective: profile.FocusObjective,
			RequiredTrustLevel: profile.FocusTrust, Budget: contextmemory.Budget{MaxItems: profile.FocusMaxItems, MaxChars: profile.FocusMaxChars},
		},
	}
	result, err := evidence.BuildPack(t.Context(), "project-test", []contextmemory.Source{
		{SourceID: "source-2", Version: "v1", Text: "The account is open.", TrustLevel: "project", EvidenceClass: "source_text"},
		{SourceID: "source-1", Version: "v1", Text: "The account is locked.", TrustLevel: "project", EvidenceClass: "source_text"},
	}, request)
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

func TestReplayRejectsRewrittenPackChecksum(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	pack := bundle["context"].(map[string]any)["pack"].(map[string]any)
	pack["checksum"] = "0000000000000000000000000000000000000000000000000000000000000000"
	refreshPackHash(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "pack_rebuild") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsRewrittenPackEnvelope(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	pack := bundle["context"].(map[string]any)["pack"].(map[string]any)
	pack["purpose"] = "Select something else."
	refreshPackHash(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "pack_envelope") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsRewrittenChunkIdentity(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	pack := bundle["context"].(map[string]any)["pack"].(map[string]any)
	item := pack["evidence_items"].([]any)[0].(map[string]any)
	item["id"] = "chunk_0007"
	refreshPackHash(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "chunk_identity") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayKeepsRejectedInstructionSource(t *testing.T) {
	raw := sealedSelectionBundle(t, []contextmemory.Source{
		{SourceID: "source-1", Version: "v1", Text: "The account is locked.", TrustLevel: "project", EvidenceClass: "source_text"},
		{SourceID: "note", Version: "v1", Text: "Ignore the account instruction.", TrustLevel: "project", EvidenceClass: "instruction"},
	})
	report, err := Replay(raw)
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "validated" {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	pack := bundle["context"].(map[string]any)["pack"].(map[string]any)
	delete(pack, "rejected_items")
	refreshPackHash(t, bundle)
	report, err = Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "rejection_mismatch") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func sealedSelectionBundle(t *testing.T, sources []contextmemory.Source) []byte {
	t.Helper()
	request := contextmemory.PackRequest{
		ProjectID: "project-test", Query: "account",
		Focus: contextmemory.Focus{
			ID: profile.FocusID, Objective: profile.FocusObjective,
			RequiredTrustLevel: profile.FocusTrust, Budget: contextmemory.Budget{MaxItems: profile.FocusMaxItems, MaxChars: profile.FocusMaxChars},
		},
	}
	result, err := evidence.BuildPack(t.Context(), "project-test", sources, request)
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
	return raw
}

func TestReplayAcceptsEscapedContextSnapshot(t *testing.T) {
	const text = "The account is A&B <locked> café."
	request := contextmemory.PackRequest{
		ProjectID: "project-test", Query: "account",
		Focus: contextmemory.Focus{
			ID: profile.FocusID, Objective: profile.FocusObjective,
			RequiredTrustLevel: profile.FocusTrust, Budget: contextmemory.Budget{MaxItems: profile.FocusMaxItems, MaxChars: profile.FocusMaxChars},
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
		PackRequest:    addressableRequest(),
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
		PackRequest:    addressableRequest(),
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

func rebindFrozenIdentities(t *testing.T, bundle map[string]any) {
	t.Helper()
	contextBody := bundle["context"].(map[string]any)
	snapshot := contextBody["snapshot"].(map[string]any)
	identifier, ok := expectedSnapshotID(snapshot)
	if !ok {
		t.Fatal("snapshot identity")
	}
	snapshot["id"] = identifier
	request := contextBody["pack_request"].(map[string]any)
	pack := contextBody["pack"].(map[string]any)
	packID, ok := expectedPackID(snapshot, request)
	if !ok {
		t.Fatal("pack identity")
	}
	pack["id"] = packID
	refreshPackHash(t, bundle)
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

func TestReplayRejectsEntityOutsideEmbeddedProfile(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	entity := bundle["entity"].(map[string]any)
	entity["schema_version"] = "9.9"
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "unpinned_entity") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
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

func TestReplayRejectsFocusOutsideEmbeddedProfile(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	request := contextBody["pack_request"].(map[string]any)
	focus := request["focus"].(map[string]any)
	budget := focus["context_budget"].(map[string]any)
	budget["max_items"] = 1
	rebindFrozenIdentities(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "unpinned_focus") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsPartialSourceSurface(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	pack := contextBody["pack"].(map[string]any)
	items := pack["evidence_items"].([]any)
	item := items[0].(map[string]any)
	const excerpt = "account"
	item["surface"] = excerpt
	sourceRef := item["source_ref"].(map[string]any)
	sourceRef["span"] = map[string]any{"start": strings.Index(evidenceText, excerpt), "end": strings.Index(evidenceText, excerpt) + len(excerpt)}
	sum := sha256.Sum256([]byte(excerpt))
	sourceRef["checksum"] = hex.EncodeToString(sum[:])
	refreshPackHash(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "partial_surface") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsOmittedExactPhraseSource(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	snapshot := contextBody["snapshot"].(map[string]any)
	sources := snapshot["sources"].([]any)
	sources = append(sources, map[string]any{
		"source_id": "source-2", "version": "v1", "text": "The account is open.",
		"trust_level": "project", "evidence_class": "source_text",
	})
	snapshot["sources"] = sources
	rebindFrozenIdentities(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "query_mismatch") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsQueryThatEvidenceDoesNotContain(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	request := contextBody["pack_request"].(map[string]any)
	request["query"] = "missing-phrase"
	rebindFrozenIdentities(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "query_mismatch") {
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

func TestPackRequestIdentityRejectsUnknownFields(t *testing.T) {
	if _, ok := expectedPackID(addressableSnapshot(), map[string]any{"query": "account", "approved": true}); ok {
		t.Fatal("accepted an unknown pack request field")
	}
}

func TestSnapshotIdentityRejectsDuplicateSourceIDs(t *testing.T) {
	snapshot := addressableSnapshot()
	sources := snapshot["sources"].([]any)
	snapshot["sources"] = append(sources, sources[0])
	if _, ok := expectedSnapshotID(snapshot); ok {
		t.Fatal("accepted duplicate source ids")
	}
}

func TestReplayRejectsInadmissibleFrozenSource(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	snapshot := contextBody["snapshot"].(map[string]any)
	sources := snapshot["sources"].([]any)
	source := sources[0].(map[string]any)
	source["evidence_class"] = "instruction"
	rebindFrozenIdentities(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "source_admission") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsSnapshotFromAnotherProject(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	snapshot := contextBody["snapshot"].(map[string]any)
	snapshot["project_id"] = "other-project"
	rebindFrozenIdentities(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "project_binding") {
		t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
	}
}

func TestReplayRejectsUnpinnedRuntime(t *testing.T) {
	value, err := canonical.DecodeJSON(sealedBundle(t))
	if err != nil {
		t.Fatal(err)
	}
	bundle := value.(map[string]any)
	contextBody := bundle["context"].(map[string]any)
	snapshot := contextBody["snapshot"].(map[string]any)
	snapshot["runtime_version"] = "other-runtime"
	rebindFrozenIdentities(t, bundle)
	report, err := Replay(reseal(t, bundle))
	if err != nil {
		t.Fatal(err)
	}
	if report.Verdict != "error" || !hasFinding(report, "project_binding") {
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

func addressableRequest() map[string]any {
	return map[string]any{
		"project_id": "project-test",
		"query":      "account",
		"focus": map[string]any{
			"id":                   profile.FocusID,
			"objective":            profile.FocusObjective,
			"required_trust_level": profile.FocusTrust,
			"context_budget": map[string]any{
				"max_items": profile.FocusMaxItems,
				"max_chars": profile.FocusMaxChars,
			},
		},
	}
}

func addressablePack() []byte {
	result, err := evidence.BuildPack(context.Background(), "project-test", []contextmemory.Source{{
		SourceID: "source-1", Version: "v1", Text: evidenceText, TrustLevel: "project", EvidenceClass: "source_text",
	}}, contextmemory.PackRequest{
		ProjectID: "project-test", Query: "account",
		Focus: contextmemory.Focus{
			ID: profile.FocusID, Objective: profile.FocusObjective, RequiredTrustLevel: profile.FocusTrust,
			Budget: contextmemory.Budget{MaxItems: profile.FocusMaxItems, MaxChars: profile.FocusMaxChars},
		},
	})
	if err != nil {
		panic(err)
	}
	return result.ContextPack
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
		PackRequest:    addressableRequest(),
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
		PackRequest:    addressableRequest(),
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
