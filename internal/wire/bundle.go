package wire

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

const protocolVersion = "0.1-draft"
const pinnedRuntime = "memory-exact-v1"

// Entity is the caller-supplied object bound into a replay bundle.
type Entity struct {
	ID            string
	ProjectID     string
	Type          string
	SchemaVersion string
	Version       string
}

// BundleInput is the frozen material required to return a replayable evaluation.
type BundleInput struct {
	Entity         Entity
	Pack           json.RawMessage
	Snapshot       any
	PackRequest    any
	AdapterID      string
	AdapterVersion string
	ResolvedModel  string
	Answers        json.RawMessage
}

// Report is the deterministic verifier result.
type Report struct {
	Verdict  verify.Verdict
	Findings []verify.Finding
}

type bundleDocument struct {
	ProtocolVersion string          `json:"protocol_version"`
	VerifierVersion string          `json:"verifier_version"`
	Entity          entityDocument  `json:"entity"`
	Context         contextDocument `json:"context"`
	QuestionSet     questionSetWire `json:"question_set"`
	Policy          policyWire      `json:"policy"`
	DecisionSet     decisionWire    `json:"decision_set"`
}

type entityDocument struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	Type          string `json:"type"`
	SchemaVersion string `json:"schema_version"`
	Version       string `json:"version"`
	Checksum      string `json:"checksum"`
}

type contextDocument struct {
	Pack        json.RawMessage `json:"pack"`
	Snapshot    any             `json:"snapshot"`
	PackRequest any             `json:"pack_request"`
	PackHash    string          `json:"pack_hash"`
}

type questionSetWire struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	Hash      string          `json:"hash"`
	Questions json.RawMessage `json:"questions"`
}

type policyWire struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

type decisionWire struct {
	ContextPackHash string          `json:"context_pack_hash"`
	QuestionSetHash string          `json:"question_set_hash"`
	PolicyHash      string          `json:"policy_hash"`
	AdapterID       string          `json:"adapter_id"`
	AdapterVersion  string          `json:"adapter_version"`
	ResolvedModel   string          `json:"resolved_model"`
	Answers         json.RawMessage `json:"answers"`
}

// BuildBundle seals a replay bundle for the embedded profile.
func BuildBundle(input BundleInput) ([]byte, error) {
	packHash, err := canonical.HashJSON(input.Pack)
	if err != nil {
		return nil, fmt.Errorf("hash context pack: %w", err)
	}
	checksum, err := canonical.HashValue(struct {
		ID            string `json:"id"`
		ProjectID     string `json:"project_id"`
		Type          string `json:"type"`
		SchemaVersion string `json:"schema_version"`
		Version       string `json:"version"`
	}{input.Entity.ID, input.Entity.ProjectID, input.Entity.Type, input.Entity.SchemaVersion, input.Entity.Version})
	if err != nil {
		return nil, fmt.Errorf("hash entity: %w", err)
	}
	questions, err := embeddedQuestions()
	if err != nil {
		return nil, err
	}
	snapshot, err := canonicalValue(input.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	packRequest, err := canonicalValue(input.PackRequest)
	if err != nil {
		return nil, fmt.Errorf("encode pack request: %w", err)
	}
	document := bundleDocument{
		ProtocolVersion: protocolVersion,
		VerifierVersion: profile.VerifierVersion,
		Entity: entityDocument{
			ID: input.Entity.ID, ProjectID: input.Entity.ProjectID, Type: input.Entity.Type,
			SchemaVersion: input.Entity.SchemaVersion, Version: input.Entity.Version, Checksum: checksum,
		},
		Context: contextDocument{Pack: input.Pack, Snapshot: snapshot, PackRequest: packRequest, PackHash: packHash},
		QuestionSet: questionSetWire{
			ID: profile.QuestionSetID, Version: profile.QuestionSetVersion, Hash: profile.QuestionSetHash(), Questions: questions,
		},
		Policy: policyWire{ID: profile.PolicyID, Version: profile.PolicyVersion, Hash: profile.PolicyHash()},
		DecisionSet: decisionWire{
			ContextPackHash: packHash,
			QuestionSetHash: profile.QuestionSetHash(),
			PolicyHash:      profile.PolicyHash(),
			AdapterID:       input.AdapterID,
			AdapterVersion:  input.AdapterVersion,
			ResolvedModel:   input.ResolvedModel,
			Answers:         input.Answers,
		},
	}
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode replay bundle: %w", err)
	}
	decoded, err := canonical.DecodeJSON(raw)
	if err != nil {
		return nil, err
	}
	hash, err := canonical.HashValue(decoded)
	if err != nil {
		return nil, err
	}
	bundle, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("replay bundle must be a JSON object")
	}
	bundle["bundle_hash"] = hash
	final, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	if err := VerifyReplayBundleHash(final); err != nil {
		return nil, err
	}
	return final, nil
}

// Replay reproduces the verdict from a sealed bundle. It does not retrieve evidence or call a provider.
func Replay(raw []byte) (Report, error) {
	if err := VerifyReplayBundleHash(raw); err != nil {
		return Report{}, err
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return Report{}, err
	}
	bundle, ok := value.(map[string]any)
	if !ok {
		return Report{}, fmt.Errorf("replay bundle must be a JSON object")
	}
	findings := bindingFindings(bundle)
	if len(findings) > 0 {
		verdict, err := verify.ResolveVerdict(findings)
		if err != nil {
			return Report{}, err
		}
		return Report{Verdict: verdict, Findings: findings}, nil
	}
	answers, parseFindings := parseAnswers(bundle)
	if len(parseFindings) > 0 {
		verdict, err := verify.ResolveVerdict(parseFindings)
		if err != nil {
			return Report{}, err
		}
		return Report{Verdict: verdict, Findings: parseFindings}, nil
	}
	verdict, findings, err := verify.Interpret(profile.TypedQuestions(), profile.Thresholds(), answers)
	if err != nil {
		return Report{}, err
	}
	findings = append(findings, evidenceFindings(bundle)...)
	verdict, err = verify.ResolveVerdict(findings)
	if err != nil {
		return Report{}, err
	}
	if findings == nil {
		findings = []verify.Finding{}
	}
	return Report{Verdict: verdict, Findings: findings}, nil
}

func bindingFindings(bundle map[string]any) []verify.Finding {
	findings := make([]verify.Finding, 0)
	questionSet, _ := bundle["question_set"].(map[string]any)
	policyRef, _ := bundle["policy"].(map[string]any)
	decision, _ := bundle["decision_set"].(map[string]any)
	contextBody, _ := bundle["context"].(map[string]any)
	if bundle["verifier_version"] != profile.VerifierVersion {
		findings = append(findings, errorFinding("unpinned_verifier"))
	}
	if questionSet["id"] != profile.QuestionSetID || questionSet["version"] != profile.QuestionSetVersion || questionSet["hash"] != profile.QuestionSetHash() || !questionContentPinned(questionSet) {
		findings = append(findings, errorFinding("unpinned_question_set"))
	}
	if policyRef["id"] != profile.PolicyID || policyRef["version"] != profile.PolicyVersion || policyRef["hash"] != profile.PolicyHash() {
		findings = append(findings, errorFinding("unpinned_policy"))
	}
	if decision["question_set_hash"] != questionSet["hash"] || decision["policy_hash"] != policyRef["hash"] || decision["context_pack_hash"] != contextBody["pack_hash"] || !packHashMatches(contextBody) {
		findings = append(findings, errorFinding("binding_mismatch"))
	}
	entity, _ := bundle["entity"].(map[string]any)
	entityType, _ := entity["type"].(string)
	schemaVersion, _ := entity["schema_version"].(string)
	if entityType != profile.EntityType || schemaVersion != profile.EntitySchemaVersion {
		findings = append(findings, errorFinding("unpinned_entity"))
	}
	if !entityChecksumMatches(entity) {
		findings = append(findings, errorFinding("entity_checksum_mismatch"))
	}
	model, _ := decision["resolved_model"].(string)
	if model == "" || strings.Contains(model, "latest") {
		findings = append(findings, errorFinding("unresolved_model"))
	}
	switch decision["adapter_id"] {
	case "direct-systemone", "hosted-systemone":
		if decision["adapter_version"] != profile.AdapterVersion {
			findings = append(findings, errorFinding("unpinned_adapter"))
		}
	default:
		findings = append(findings, errorFinding("unknown_adapter"))
	}
	return findings
}

func evidenceFindings(bundle map[string]any) []verify.Finding {
	contextBody, _ := bundle["context"].(map[string]any)
	pack, _ := contextBody["pack"].(map[string]any)
	rawItems, ok := pack["evidence_items"].([]any)
	if !ok {
		return []verify.Finding{{Code: "pack_shape", Verdict: verify.VerdictError, Detail: "frozen pack has no evidence item list"}}
	}
	admissible := 0
	inference := 0
	findings := make([]verify.Finding, 0)
	for _, raw := range rawItems {
		item, _ := raw.(map[string]any)
		class, _ := item["class"].(string)
		trust, _ := item["trust_level"].(string)
		switch class {
		case "instruction", "policy":
			findings = append(findings, verify.Finding{Code: "instruction_in_evidence", Verdict: verify.VerdictError, Detail: "instructions and policy are not evidence"})
		case "model_inference":
			inference++
		case "source_text":
			if trust != "project" {
				break
			}
			if finding, failed := provenanceFinding(bundle, item); failed {
				findings = append(findings, finding)
				break
			}
			admissible++
		}
	}
	if len(findings) > 0 {
		return findings
	}
	if admissible == 0 && inference > 0 {
		return []verify.Finding{{Code: "inference_only", Verdict: verify.VerdictInsufficient, Detail: "model inference cannot establish a factual claim"}}
	}
	if finding, failed := selectionFinding(bundle); failed {
		return []verify.Finding{finding}
	}
	if finding, failed := rejectionFinding(bundle); failed {
		return []verify.Finding{finding}
	}
	if finding, failed := packShapeFinding(bundle); failed {
		return []verify.Finding{finding}
	}
	if finding, failed := rebuiltPackFinding(bundle); failed {
		return []verify.Finding{finding}
	}
	if admissible == 0 {
		return []verify.Finding{{Code: "no_eligible_evidence", Verdict: verify.VerdictInsufficient, Detail: "the frozen pack contains no admissible source text"}}
	}
	return nil
}

func parseAnswers(bundle map[string]any) (map[string]verify.Answer, []verify.Finding) {
	decision, _ := bundle["decision_set"].(map[string]any)
	raw, err := json.Marshal(decision["answers"])
	if err != nil {
		return nil, []verify.Finding{errorFinding("invalid_answers")}
	}
	var decoded map[string]struct {
		Type          string             `json:"type"`
		Noul          *float64           `json:"noul"`
		Choice        string             `json:"choice"`
		Probabilities map[string]float64 `json:"probabilities"`
		Score         *float64           `json:"score"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, []verify.Finding{errorFinding("invalid_answers")}
	}
	answers := make(map[string]verify.Answer, len(decoded))
	for id, answer := range decoded {
		parsed := verify.Answer{
			Type: verify.QuestionType(answer.Type), Noul: answer.Noul, Choice: answer.Choice,
			Probabilities: answer.Probabilities, Score: answer.Score,
		}
		answers[id] = parsed
	}
	return answers, nil
}

func packHashMatches(contextBody map[string]any) bool {
	hash, err := canonical.HashValue(contextBody["pack"])
	declared, _ := contextBody["pack_hash"].(string)
	return err == nil && hash == declared
}

func entityChecksumMatches(entity map[string]any) bool {
	id, _ := entity["id"].(string)
	projectID, _ := entity["project_id"].(string)
	entityType, _ := entity["type"].(string)
	schemaVersion, _ := entity["schema_version"].(string)
	version, _ := entity["version"].(string)
	hash, err := canonical.HashValue(struct {
		ID            string `json:"id"`
		ProjectID     string `json:"project_id"`
		Type          string `json:"type"`
		SchemaVersion string `json:"schema_version"`
		Version       string `json:"version"`
	}{id, projectID, entityType, schemaVersion, version})
	declared, _ := entity["checksum"].(string)
	return err == nil && hash == declared
}

func questionContentPinned(questionSet map[string]any) bool {
	id, _ := questionSet["id"].(string)
	version, _ := questionSet["version"].(string)
	hash, err := canonical.HashValue(struct {
		ID        string `json:"id"`
		Version   string `json:"version"`
		Questions any    `json:"questions"`
	}{ID: id, Version: version, Questions: questionSet["questions"]})
	return err == nil && hash == profile.QuestionSetHash() && questionSet["hash"] == hash
}

func provenanceFinding(bundle map[string]any, item map[string]any) (verify.Finding, bool) {
	surface, _ := item["surface"].(string)
	sourceRef, _ := item["source_ref"].(map[string]any)
	checksum, _ := sourceRef["checksum"].(string)
	sourceID, _ := sourceRef["source_id"].(string)
	projectID, _ := sourceRef["project_id"].(string)
	entity, _ := bundle["entity"].(map[string]any)
	if surface == "" || sourceID == "" || checksum == "" || projectID == "" || projectID != entity["project_id"] {
		return verify.Finding{Code: "missing_provenance", Verdict: verify.VerdictError, Detail: "admissible evidence has no complete source identity"}, true
	}
	contextBody, _ := bundle["context"].(map[string]any)
	snapshot, _ := contextBody["snapshot"].(map[string]any)
	if !snapshotIDMatches(snapshot) {
		return verify.Finding{Code: "snapshot_identity", Verdict: verify.VerdictError, Detail: "frozen snapshot identity does not match its sources"}, true
	}
	if snapshot["project_id"] != entity["project_id"] || snapshot["runtime_version"] != pinnedRuntime {
		return verify.Finding{Code: "project_binding", Verdict: verify.VerdictError, Detail: "frozen snapshot is not bound to the entity project and pinned runtime"}, true
	}
	pack, _ := contextBody["pack"].(map[string]any)
	request, _ := contextBody["pack_request"].(map[string]any)
	if !packRequestIDMatches(snapshot, pack, request) {
		return verify.Finding{Code: "pack_request_identity", Verdict: verify.VerdictError, Detail: "frozen pack identity does not match the pack request"}, true
	}
	if !focusPinned(request, textField(entity["project_id"])) {
		return verify.Finding{Code: "unpinned_focus", Verdict: verify.VerdictError, Detail: "frozen pack request does not use the embedded focus"}, true
	}
	sum := sha256.Sum256([]byte(surface))
	if hex.EncodeToString(sum[:]) != checksum {
		return verify.Finding{Code: "checksum_mismatch", Verdict: verify.VerdictError, Detail: "evidence surface does not match its source checksum"}, true
	}
	sources, _ := snapshot["sources"].([]any)
	for _, raw := range sources {
		source, _ := raw.(map[string]any)
		if source["source_id"] != sourceID {
			continue
		}
		version, _ := source["version"].(string)
		text, _ := source["text"].(string)
		if version == "" || source["trust_level"] != "project" || source["evidence_class"] != "source_text" {
			return verify.Finding{Code: "source_admission", Verdict: verify.VerdictError, Detail: "frozen source is not admissible project text"}, true
		}
		excerpt := text
		if span, ok := sourceRef["span"].(map[string]any); ok {
			start, startOK := nonNegative(span["start"])
			end, endOK := nonNegative(span["end"])
			if !startOK || !endOK || start > end || end > uint64(len(text)) {
				return verify.Finding{Code: "checksum_mismatch", Verdict: verify.VerdictError, Detail: "evidence span does not fit the frozen source"}, true
			}
			excerpt = text[start:end]
		}
		if excerpt != surface {
			return verify.Finding{Code: "checksum_mismatch", Verdict: verify.VerdictError, Detail: "evidence surface does not match the frozen source"}, true
		}
		if excerpt != text {
			return verify.Finding{Code: "partial_surface", Verdict: verify.VerdictError, Detail: "evidence surface must be the full frozen source"}, true
		}
		return verify.Finding{}, false
	}
	return verify.Finding{Code: "missing_provenance", Verdict: verify.VerdictError, Detail: "admissible evidence does not resolve to a frozen source"}, true
}

func selectionFinding(bundle map[string]any) (verify.Finding, bool) {
	contextBody, _ := bundle["context"].(map[string]any)
	request, _ := contextBody["pack_request"].(map[string]any)
	req, ok := decodePackRequest(request)
	if !ok {
		return verify.Finding{Code: "unpinned_focus", Verdict: verify.VerdictError, Detail: "frozen pack request does not use the embedded focus"}, true
	}
	snapshot, _ := contextBody["snapshot"].(map[string]any)
	query := strings.TrimSpace(req.Query)
	expected, ok := matchingSourceIDs(snapshot, query)
	if !ok || query == "" {
		return verify.Finding{Code: "query_mismatch", Verdict: verify.VerdictError, Detail: "admissible evidence does not contain the frozen exact phrase"}, true
	}
	pack, _ := contextBody["pack"].(map[string]any)
	rawItems, _ := pack["evidence_items"].([]any)
	got := make([]string, 0, len(rawItems))
	for _, raw := range rawItems {
		item, _ := raw.(map[string]any)
		class, _ := item["class"].(string)
		trust, _ := item["trust_level"].(string)
		if class != "source_text" || trust != "project" {
			continue
		}
		sourceRef, _ := item["source_ref"].(map[string]any)
		got = append(got, textField(sourceRef["source_id"]))
	}
	if len(got) != len(expected) {
		return verify.Finding{Code: "query_mismatch", Verdict: verify.VerdictError, Detail: "admissible evidence is not the exact-phrase selection"}, true
	}
	for i := range got {
		if got[i] != expected[i] {
			return verify.Finding{Code: "query_mismatch", Verdict: verify.VerdictError, Detail: "admissible evidence is not the exact-phrase selection"}, true
		}
	}
	return verify.Finding{}, false
}

func matchingSourceIDs(snapshot map[string]any, query string) ([]string, bool) {
	rawSources, _ := snapshot["sources"].([]any)
	type candidate struct {
		id   string
		text string
	}
	candidates := make([]candidate, 0, len(rawSources))
	for _, raw := range rawSources {
		source, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		if source["trust_level"] != profile.FocusTrust || source["evidence_class"] != "source_text" {
			continue
		}
		text, _ := source["text"].(string)
		if query == "" || !strings.Contains(text, query) {
			continue
		}
		candidates = append(candidates, candidate{id: textField(source["source_id"]), text: text})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].id < candidates[j].id })
	ids := make([]string, 0, len(candidates))
	chars := 0
	for _, candidate := range candidates {
		if len(ids) >= profile.FocusMaxItems {
			continue
		}
		if chars+len(candidate.text) > profile.FocusMaxChars {
			continue
		}
		ids = append(ids, candidate.id)
		chars += len(candidate.text)
	}
	return ids, true
}

func rejectionFinding(bundle map[string]any) (verify.Finding, bool) {
	contextBody, _ := bundle["context"].(map[string]any)
	request, _ := contextBody["pack_request"].(map[string]any)
	req, ok := decodePackRequest(request)
	if !ok {
		return verify.Finding{Code: "unpinned_focus", Verdict: verify.VerdictError, Detail: "frozen pack request does not use the embedded focus"}, true
	}
	snapshot, _ := contextBody["snapshot"].(map[string]any)
	query := strings.TrimSpace(req.Query)
	expected, ok := expectedRejections(snapshot, query)
	if !ok || query == "" {
		return verify.Finding{Code: "rejection_mismatch", Verdict: verify.VerdictError, Detail: "frozen rejections do not match the exact-phrase scan"}, true
	}
	pack, _ := contextBody["pack"].(map[string]any)
	rawItems, _ := pack["rejected_items"].([]any)
	if len(rawItems) != len(expected) {
		return verify.Finding{Code: "rejection_mismatch", Verdict: verify.VerdictError, Detail: "frozen rejections do not match the exact-phrase scan"}, true
	}
	for i, raw := range rawItems {
		item, _ := raw.(map[string]any)
		sourceRef, _ := item["source_ref"].(map[string]any)
		if textField(sourceRef["source_id"]) != expected[i].sourceID || textField(item["rejection_reason"]) != expected[i].reason {
			return verify.Finding{Code: "rejection_mismatch", Verdict: verify.VerdictError, Detail: "frozen rejections do not match the exact-phrase scan"}, true
		}
	}
	return verify.Finding{}, false
}

func packShapeFinding(bundle map[string]any) (verify.Finding, bool) {
	contextBody, _ := bundle["context"].(map[string]any)
	pack, _ := contextBody["pack"].(map[string]any)
	snapshot, _ := contextBody["snapshot"].(map[string]any)
	packID := textField(pack["id"])
	switch {
	case !strings.HasPrefix(packID, "pack_") || pack["retrieval_plan_id"] != "plan_"+strings.TrimPrefix(packID, "pack_"):
		return packEnvelope("plan")
	case pack["project_id"] != snapshot["project_id"] || textField(pack["task_id"]) != "":
		return packEnvelope("project")
	case pack["purpose"] != profile.FocusObjective:
		return packEnvelope("purpose")
	case pack["budget_estimator_version"] != "chars-div4-v1" || !budgetPinned(pack["budget"]):
		return packEnvelope("budget")
	case !emptyList(pack["instructions"]) || !emptyList(pack["policy_refs"]) || !emptyList(pack["verification_requirements"]):
		return packEnvelope("controls")
	}
	chunks, ok := chunkIDs(snapshot)
	if !ok {
		return verify.Finding{Code: "chunk_identity", Verdict: verify.VerdictError, Detail: "frozen evidence chunk is not the exact-phrase chunk"}, true
	}
	for _, field := range []string{"evidence_items", "rejected_items"} {
		rawItems, _ := pack[field].([]any)
		for _, raw := range rawItems {
			item, _ := raw.(map[string]any)
			if !chunkPinned(item, chunks, textField(snapshot["id"]), textField(snapshot["project_id"])) {
				return verify.Finding{Code: "chunk_identity", Verdict: verify.VerdictError, Detail: "frozen evidence chunk is not the exact-phrase chunk"}, true
			}
		}
	}
	return verify.Finding{}, false
}

func rebuiltPackFinding(bundle map[string]any) (verify.Finding, bool) {
	mismatch := verify.Finding{Code: "pack_rebuild", Verdict: verify.VerdictError, Detail: "frozen pack does not match the pack rebuilt from the snapshot and request"}
	contextBody, _ := bundle["context"].(map[string]any)
	pack, _ := contextBody["pack"].(map[string]any)
	snapshot, _ := contextBody["snapshot"].(map[string]any)
	request, _ := contextBody["pack_request"].(map[string]any)
	sources, err := snapshotSources(snapshot)
	if err != nil {
		return mismatch, true
	}
	var req contextmemory.PackRequest
	if err = recode(request, &req); err != nil {
		return mismatch, true
	}
	result, err := evidence.BuildPack(context.Background(), textField(snapshot["project_id"]), sources, req)
	if err != nil {
		return mismatch, true
	}
	saved, err := canonical.HashValue(pack)
	if err != nil {
		return mismatch, true
	}
	fresh, err := canonical.DecodeJSON(result.ContextPack)
	if err != nil {
		return mismatch, true
	}
	rebuilt, err := canonical.HashValue(fresh)
	if err != nil || saved != rebuilt {
		return mismatch, true
	}
	return verify.Finding{}, false
}

func snapshotSources(snapshot map[string]any) ([]contextmemory.Source, error) {
	var sources []contextmemory.Source
	if err := recode(snapshot["sources"], &sources); err != nil {
		return nil, err
	}
	return sources, nil
}

func recode(value any, dest any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, dest)
}

func packEnvelope(part string) (verify.Finding, bool) {
	return verify.Finding{Code: "pack_envelope", Verdict: verify.VerdictError, Detail: "frozen pack envelope is not the embedded exact-phrase pack: " + part}, true
}

func emptyList(value any) bool {
	if value == nil {
		return true
	}
	items, ok := value.([]any)
	return ok && len(items) == 0
}

func budgetPinned(value any) bool {
	budget, ok := value.(map[string]any)
	if !ok {
		return false
	}
	return numberEqual(budget["max_items"], profile.FocusMaxItems) && numberEqual(budget["max_chars"], profile.FocusMaxChars) && numberEqual(budget["max_tokens_estimate"], 0) && budget["budget_estimator_version"] == "chars-div4-v1" && numberEqual(budget["reject_score_floor"], 0.3) && budget["allow_span_truncate"] != true && numberEqual(budget["reserve_for_instructions"], 0)
}

func numberEqual(value any, want float64) bool {
	if value == nil {
		return want == 0
	}
	number, ok := value.(json.Number)
	if !ok {
		return false
	}
	got, err := number.Float64()
	return err == nil && math.Abs(got-want) <= 1e-9
}

func chunkIDs(snapshot map[string]any) (map[string]string, bool) {
	rawSources, _ := snapshot["sources"].([]any)
	ids := make([]string, 0, len(rawSources))
	for _, raw := range rawSources {
		source, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		ids = append(ids, textField(source["source_id"]))
	}
	sort.Strings(ids)
	chunks := make(map[string]string, len(ids))
	for i, id := range ids {
		if id == "" {
			return nil, false
		}
		chunks[id] = fmt.Sprintf("chunk_%04d", i)
	}
	return chunks, true
}

func chunkPinned(item map[string]any, chunks map[string]string, snapshotID, projectID string) bool {
	sourceRef, _ := item["source_ref"].(map[string]any)
	sourceID := textField(sourceRef["source_id"])
	chunkID, ok := chunks[sourceID]
	candidate, _ := item["candidate"].(map[string]any)
	contributions, _ := candidate["contributions"].([]any)
	if !ok || item["id"] != chunkID || candidate["chunk_id"] != chunkID || len(contributions) != 1 || !numberEqual(candidate["merged_score"], 1) || candidate["trust_level"] != item["trust_level"] {
		return false
	}
	contribution, _ := contributions[0].(map[string]any)
	reasons, _ := contribution["reasons"].([]any)
	return contribution["retriever_id"] == "exact" && numberEqual(contribution["raw_score"], 1) && numberEqual(contribution["normalized_score"], 1) && numberEqual(contribution["weight"], 1) && len(reasons) == 1 && reasons[0] == "exact_phrase" && contribution["explanation"] == "exact phrase match in chunk text" && contribution["snapshot_id"] == snapshotID && contribution["project_id"] == projectID
}

type phraseRejection struct {
	sourceID string
	reason   string
}

func expectedRejections(snapshot map[string]any, query string) ([]phraseRejection, bool) {
	sources, ok := matchingPhraseSources(snapshot, query)
	if !ok {
		return nil, false
	}
	rejected := make([]phraseRejection, 0)
	admissible := make([]phraseSource, 0)
	for _, source := range sources {
		switch source.class {
		case "instruction", "policy":
			rejected = append(rejected, phraseRejection{source.id, "instruction_or_policy_not_evidence"})
			continue
		}
		if source.trust == "quarantined" {
			rejected = append(rejected, phraseRejection{source.id, "quarantined"})
			continue
		}
		if trustRank(source.trust) < trustRank(profile.FocusTrust) {
			rejected = append(rejected, phraseRejection{source.id, "trust_below_required"})
			continue
		}
		admissible = append(admissible, source)
	}
	chars := 0
	selected := 0
	for _, source := range admissible {
		if selected >= profile.FocusMaxItems || chars+len(source.text) > profile.FocusMaxChars {
			rejected = append(rejected, phraseRejection{source.id, "budget_trim"})
			continue
		}
		selected++
		chars += len(source.text)
	}
	return rejected, true
}

type phraseSource struct {
	id    string
	text  string
	trust string
	class string
}

func matchingPhraseSources(snapshot map[string]any, query string) ([]phraseSource, bool) {
	rawSources, _ := snapshot["sources"].([]any)
	sources := make([]phraseSource, 0, len(rawSources))
	for _, raw := range rawSources {
		source, ok := raw.(map[string]any)
		if !ok {
			return nil, false
		}
		text, _ := source["text"].(string)
		if query == "" || !strings.Contains(text, query) {
			continue
		}
		sources = append(sources, phraseSource{
			id: textField(source["source_id"]), text: text,
			trust: textField(source["trust_level"]), class: textField(source["evidence_class"]),
		})
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].id < sources[j].id })
	return sources, true
}

func trustRank(level string) int {
	switch level {
	case "trusted":
		return 4
	case "project":
		return 3
	case "external":
		return 2
	case "untrusted":
		return 1
	case "quarantined":
		return 0
	default:
		return -1
	}
}

func packRequestIDMatches(snapshot, pack, request map[string]any) bool {
	expected, ok := expectedPackID(snapshot, request)
	declared, _ := pack["id"].(string)
	return ok && declared == expected
}

func expectedPackID(snapshot, request map[string]any) (string, bool) {
	req, ok := decodePackRequest(request)
	if !ok {
		return "", false
	}
	raw, err := json.Marshal(struct {
		SnapshotID string            `json:"snapshot_id"`
		Request    frozenPackRequest `json:"request"`
	}{textField(snapshot["id"]), req})
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(append([]byte("context/memory-pack-request/v1\x00"), raw...))
	return "pack_" + hex.EncodeToString(sum[:]), true
}

func focusPinned(request map[string]any, projectID string) bool {
	req, ok := decodePackRequest(request)
	if !ok {
		return false
	}
	if req.ProjectID != projectID || req.Query == "" || req.TaskID != "" || len(req.Instructions) > 0 || len(req.PolicyRefs) > 0 || len(req.VerificationRequirements) > 0 {
		return false
	}
	if req.Focus.ID != profile.FocusID || req.Focus.Objective != profile.FocusObjective || req.Focus.RequiredTrustLevel != profile.FocusTrust {
		return false
	}
	return req.Focus.Budget.MaxItems == profile.FocusMaxItems && req.Focus.Budget.MaxChars == profile.FocusMaxChars && req.Focus.Budget.MaxTokensEstimate == 0
}

func decodePackRequest(request map[string]any) (frozenPackRequest, bool) {
	encoded, err := json.Marshal(request)
	if err != nil {
		return frozenPackRequest{}, false
	}
	var req frozenPackRequest
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		return frozenPackRequest{}, false
	}
	return req, true
}

type frozenPackRequest struct {
	ProjectID                string      `json:"project_id"`
	TaskID                   string      `json:"task_id,omitempty"`
	Query                    string      `json:"query"`
	Focus                    frozenFocus `json:"focus"`
	Instructions             []string    `json:"instructions,omitempty"`
	PolicyRefs               []string    `json:"policy_refs,omitempty"`
	VerificationRequirements []string    `json:"verification_requirements,omitempty"`
}

type frozenFocus struct {
	ID                 string       `json:"id"`
	Objective          string       `json:"objective"`
	RequiredTrustLevel string       `json:"required_trust_level"`
	Budget             frozenBudget `json:"context_budget"`
}

type frozenBudget struct {
	MaxItems          int `json:"max_items"`
	MaxChars          int `json:"max_chars"`
	MaxTokensEstimate int `json:"max_tokens_estimate,omitempty"`
}

func snapshotIDMatches(snapshot map[string]any) bool {
	expected, ok := expectedSnapshotID(snapshot)
	declared, _ := snapshot["id"].(string)
	return ok && declared == expected
}

func expectedSnapshotID(snapshot map[string]any) (string, bool) {
	for key := range snapshot {
		switch key {
		case "id", "project_id", "runtime_version", "sources":
		default:
			return "", false
		}
	}
	encoded, err := json.Marshal(snapshot["sources"])
	if err != nil {
		return "", false
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var sources []frozenSource
	if err = decoder.Decode(&sources); err != nil {
		return "", false
	}
	var trailing any
	if err = decoder.Decode(&trailing); err != io.EOF {
		return "", false
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].SourceID < sources[j].SourceID })
	seen := make(map[string]struct{}, len(sources))
	for _, source := range sources {
		if source.SourceID == "" {
			return "", false
		}
		if _, duplicate := seen[source.SourceID]; duplicate {
			return "", false
		}
		seen[source.SourceID] = struct{}{}
	}
	raw, err := json.Marshal(frozenSnapshot{
		ProjectID:      textField(snapshot["project_id"]),
		RuntimeVersion: textField(snapshot["runtime_version"]),
		Sources:        sources,
	})
	if err != nil {
		return "", false
	}
	sum := sha256.Sum256(append([]byte("context/memory-snapshot/v1\x00"), raw...))
	return "snapshot_" + hex.EncodeToString(sum[:]), true
}

func textField(value any) string {
	text, _ := value.(string)
	return text
}

type frozenSnapshot struct {
	ID             string         `json:"id"`
	ProjectID      string         `json:"project_id"`
	RuntimeVersion string         `json:"runtime_version"`
	Sources        []frozenSource `json:"sources"`
}

type frozenSource struct {
	SourceID      string `json:"source_id"`
	Version       string `json:"version"`
	Text          string `json:"text"`
	TrustLevel    string `json:"trust_level"`
	EvidenceClass string `json:"evidence_class"`
}

func nonNegative(value any) (uint64, bool) {
	switch number := value.(type) {
	case json.Number:
		parsed, err := number.Int64()
		if err != nil || parsed < 0 {
			return 0, false
		}
		return uint64(parsed), true
	case float64:
		if number < 0 || number != float64(uint64(number)) {
			return 0, false
		}
		return uint64(number), true
	default:
		return 0, false
	}
}

func errorFinding(code string) verify.Finding {
	return verify.Finding{Code: code, Verdict: verify.VerdictError, Detail: "replay binding failed " + code}
}

func embeddedQuestions() (json.RawMessage, error) {
	raw, err := json.Marshal(profile.QuestionSetBody())
	if err != nil {
		return nil, err
	}
	var document struct {
		Questions json.RawMessage `json:"questions"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil, err
	}
	return document.Questions, nil
}

func canonicalValue(value any) (any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return canonical.DecodeJSON(raw)
}
