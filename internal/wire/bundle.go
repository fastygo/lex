// Package wire validates versioned LeX wire messages, seals replay bundles,
// and replays them deterministically.
package wire

import (
	"encoding/json"
	"fmt"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

const (
	// ProtocolVersion is the wire envelope this verifier seals and replays.
	ProtocolVersion = "0.1-draft"
	// VerifierVersion pins the deterministic interpreter recorded in every bundle.
	VerifierVersion = "0.2.0"
)

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
	Frozen         evidence.Frozen
	AdapterID      string
	AdapterVersion string
	ResolvedModel  string
	Answers        json.RawMessage
	Skipped        bool
}

// Report is the deterministic verifier result.
type Report struct {
	Verdict  verify.Verdict
	Findings []verify.Finding
}

func entityChecksum(id, projectID, entityType, schemaVersion, version string) (string, error) {
	return canonical.HashValue(struct {
		ID            string `json:"id"`
		ProjectID     string `json:"project_id"`
		Type          string `json:"type"`
		SchemaVersion string `json:"schema_version"`
		Version       string `json:"version"`
	}{id, projectID, entityType, schemaVersion, version})
}

func questionSetHash(id, version string, questions json.RawMessage) (string, error) {
	return canonical.HashValue(struct {
		ID        string          `json:"id"`
		Version   string          `json:"version"`
		Questions json.RawMessage `json:"questions"`
	}{id, version, questions})
}

// BuildBundle seals a replay bundle for one profile.
func BuildBundle(prof profile.Profile, input BundleInput) ([]byte, error) {
	if input.Skipped && (input.AdapterID != "" || input.AdapterVersion != "" || input.ResolvedModel != "" || len(input.Answers) != 0) {
		return nil, fmt.Errorf("skipped decision cannot carry adapter output")
	}
	packHash, err := canonical.HashJSON(input.Frozen.Pack)
	if err != nil {
		return nil, fmt.Errorf("hash context pack: %w", err)
	}
	checksum, err := entityChecksum(input.Entity.ID, input.Entity.ProjectID, input.Entity.Type, input.Entity.SchemaVersion, input.Entity.Version)
	if err != nil {
		return nil, fmt.Errorf("hash entity: %w", err)
	}
	questions, err := json.Marshal(prof.QuestionSet().Questions)
	if err != nil {
		return nil, fmt.Errorf("encode questions: %w", err)
	}
	snapshot, err := json.Marshal(input.Frozen.Snapshot)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	packRequest, err := json.Marshal(input.Frozen.PackRequest)
	if err != nil {
		return nil, fmt.Errorf("encode pack request: %w", err)
	}
	decision := DecisionRecord{ContextPackHash: packHash, QuestionSetHash: prof.QuestionSetHash(), PolicyHash: prof.PolicyHash()}
	if input.Skipped {
		decision.Skipped = true
	} else {
		decision.AdapterID, decision.AdapterVersion, decision.ResolvedModel, decision.Answers = input.AdapterID, input.AdapterVersion, input.ResolvedModel, input.Answers
	}
	document := Bundle{
		ProtocolVersion: ProtocolVersion,
		VerifierVersion: VerifierVersion,
		Entity: EntityRecord{
			ID: input.Entity.ID, ProjectID: input.Entity.ProjectID, Type: input.Entity.Type,
			SchemaVersion: input.Entity.SchemaVersion, Version: input.Entity.Version, Checksum: checksum,
		},
		Context: ContextRecord{Pack: input.Frozen.Pack, Snapshot: snapshot, PackRequest: packRequest, PackHash: packHash},
		QuestionSet: QuestionSetRecord{
			ID: prof.QuestionSet().ID, Version: prof.QuestionSet().Version, Hash: prof.QuestionSetHash(), Questions: questions,
		},
		Policy:      PolicyRecord{ID: prof.Policy().ID, Version: prof.Policy().Version, Hash: prof.PolicyHash()},
		DecisionSet: decision,
	}
	return seal(document)
}

// seal computes the self-hash over the canonical document and appends it.
func seal(document Bundle) ([]byte, error) {
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
