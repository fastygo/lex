package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Bundle is the typed form of the published replay-bundle schema. Every field
// mirrors schema/replay-bundle.schema.json; TestBundleTypeMirrorsSchema keeps
// them aligned. The schema is validated first, so decoding never widens it.
type Bundle struct {
	ProtocolVersion string            `json:"protocol_version"`
	VerifierVersion string            `json:"verifier_version"`
	Entity          EntityRecord      `json:"entity"`
	Context         ContextRecord     `json:"context"`
	QuestionSet     QuestionSetRecord `json:"question_set"`
	Policy          PolicyRecord      `json:"policy"`
	DecisionSet     DecisionRecord    `json:"decision_set"`
	BundleHash      string            `json:"bundle_hash,omitempty"`
}

// EntityRecord is the sealed entity identity and its checksum.
type EntityRecord struct {
	ID            string `json:"id"`
	ProjectID     string `json:"project_id"`
	Type          string `json:"type"`
	SchemaVersion string `json:"schema_version"`
	Version       string `json:"version"`
	Checksum      string `json:"checksum"`
}

// ContextRecord is the frozen evidence state. Pack, snapshot, and pack request
// keep Context's own JSON; the verifier decodes them through the evidence plane.
type ContextRecord struct {
	Pack        json.RawMessage `json:"pack"`
	Snapshot    json.RawMessage `json:"snapshot"`
	PackRequest json.RawMessage `json:"pack_request"`
	PackHash    string          `json:"pack_hash"`
}

// QuestionSetRecord is the sealed question document reference and body.
type QuestionSetRecord struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	Hash      string          `json:"hash"`
	Questions json.RawMessage `json:"questions"`
}

// PolicyRecord is the sealed policy reference.
type PolicyRecord struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Hash    string `json:"hash"`
}

// DecisionRecord is either a provider decision or an explicit skip. The schema
// oneOf guarantees the two shapes never mix; the verifier re-checks it.
type DecisionRecord struct {
	ContextPackHash string          `json:"context_pack_hash"`
	QuestionSetHash string          `json:"question_set_hash"`
	PolicyHash      string          `json:"policy_hash"`
	AdapterID       string          `json:"adapter_id,omitempty"`
	AdapterVersion  string          `json:"adapter_version,omitempty"`
	ResolvedModel   string          `json:"resolved_model,omitempty"`
	Answers         json.RawMessage `json:"answers,omitempty"`
	Skipped         bool            `json:"skipped,omitempty"`
}

// DecodeBundle validates raw against the schema and self-hash, then decodes it.
func DecodeBundle(raw []byte) (Bundle, error) {
	if err := VerifyReplayBundleHash(raw); err != nil {
		return Bundle{}, err
	}
	var bundle Bundle
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return Bundle{}, fmt.Errorf("decode replay bundle: %w", err)
	}
	return bundle, nil
}
