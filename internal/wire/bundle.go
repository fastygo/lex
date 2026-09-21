package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

const protocolVersion = "0.1-draft"

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
	if !entityChecksumMatches(entity) {
		findings = append(findings, errorFinding("entity_checksum_mismatch"))
	}
	model, _ := decision["resolved_model"].(string)
	if model == "" || strings.Contains(model, "latest") {
		findings = append(findings, errorFinding("unresolved_model"))
	}
	switch decision["adapter_id"] {
	case "direct-systemone", "hosted-systemone":
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
	sum := sha256.Sum256([]byte(surface))
	if hex.EncodeToString(sum[:]) != checksum {
		return verify.Finding{Code: "checksum_mismatch", Verdict: verify.VerdictError, Detail: "evidence surface does not match its source checksum"}, true
	}
	contextBody, _ := bundle["context"].(map[string]any)
	snapshot, _ := contextBody["snapshot"].(map[string]any)
	sources, _ := snapshot["sources"].([]any)
	for _, raw := range sources {
		source, _ := raw.(map[string]any)
		if source["source_id"] != sourceID {
			continue
		}
		version, _ := source["version"].(string)
		text, _ := source["text"].(string)
		if version == "" {
			continue
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
		return verify.Finding{}, false
	}
	return verify.Finding{Code: "missing_provenance", Verdict: verify.VerdictError, Detail: "admissible evidence does not resolve to a frozen source"}, true
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
