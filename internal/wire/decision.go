// Package wire owns the LeX wire contract: request and bundle schemas, typed
// records, canonical hashes, and the structural replay verifier.
package wire

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/verify"
)

const (
	// DecisionProtocolVersion is the caller-defined decision envelope.
	DecisionProtocolVersion = "0.2"
	// DecisionBundleKind names the sealed typed-decision bundle.
	DecisionBundleKind = "typed_decision"
	// DecisionVerifierVersion identifies the structural verifier.
	DecisionVerifierVersion = "0.3.0"
)

// DecisionIdentity is caller-owned identity for one decision request.
type DecisionIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// Question is one caller-owned typed question. Options and Levels are
// deliberately distinct so an unordered Choice cannot be confused with Score.
type Question struct {
	Type         verify.QuestionType `json:"type"`
	Instructions string              `json:"instructions"`
	Options      map[string]string   `json:"options,omitempty"`
	Levels       []string            `json:"levels,omitempty"`
}

// QuestionSet is a versioned caller-owned question document.
type QuestionSet struct {
	ID        string              `json:"id"`
	Version   string              `json:"version"`
	Questions map[string]Question `json:"questions"`
}

// Validate checks question semantics after the JSON schema checks shape.
func (set QuestionSet) Validate() error {
	if !wireID(set.ID) || !wireVersion(set.Version) || len(set.Questions) < 1 || len(set.Questions) > 64 {
		return fmt.Errorf("question set id, version, and question count are invalid")
	}
	for id, question := range set.Questions {
		if !questionID(id) || question.Instructions == "" || len(question.Instructions) > 16384 {
			return fmt.Errorf("question %q is invalid", id)
		}
		switch question.Type {
		case verify.QuestionNoul:
			if len(question.Options) != 0 || len(question.Levels) != 0 {
				return fmt.Errorf("noul question %q has an answer domain", id)
			}
		case verify.QuestionChoice:
			if len(question.Options) < 2 || len(question.Options) > 255 || len(question.Levels) != 0 {
				return fmt.Errorf("choice question %q has an invalid option domain", id)
			}
			for option, description := range question.Options {
				if !questionID(option) || description == "" || len(description) > 16384 {
					return fmt.Errorf("choice question %q has an invalid option", id)
				}
			}
		case verify.QuestionScore:
			if len(question.Options) != 0 || len(question.Levels) < 2 || len(question.Levels) > 10 {
				return fmt.Errorf("score question %q has an invalid ordered domain", id)
			}
			seen := map[string]struct{}{}
			for _, level := range question.Levels {
				if level == "" || len(level) > 16384 {
					return fmt.Errorf("score question %q has an invalid level", id)
				}
				if _, duplicate := seen[level]; duplicate {
					return fmt.Errorf("score question %q repeats a level", id)
				}
				seen[level] = struct{}{}
			}
		default:
			return fmt.Errorf("question %q has unsupported type", id)
		}
	}
	return nil
}

// VerifierQuestions returns the strict answer-domain view of the QuestionSet.
func (set QuestionSet) VerifierQuestions() map[string]verify.Question {
	questions := make(map[string]verify.Question, len(set.Questions))
	for id, question := range set.Questions {
		typed := verify.Question{Type: question.Type}
		switch question.Type {
		case verify.QuestionChoice:
			typed.Choices = sortedKeys(question.Options)
		case verify.QuestionScore:
			typed.Levels = len(question.Levels)
		}
		questions[id] = typed
	}
	return questions
}

// ProviderQuestions maps the wire primitives to the provider-neutral
// adapter shape without adding domain semantics.
func (set QuestionSet) ProviderQuestions() map[string]any {
	questions := make(map[string]any, len(set.Questions))
	for id, question := range set.Questions {
		provider := map[string]any{"type": string(question.Type), "instructions": question.Instructions}
		switch question.Type {
		case verify.QuestionChoice:
			provider["criteria"] = question.Options
		case verify.QuestionScore:
			provider["criteria"] = question.Levels
		}
		questions[id] = provider
	}
	return questions
}

func questionSetHash(set QuestionSet) (string, error) {
	return canonical.HashValue(struct {
		ID        string              `json:"id"`
		Version   string              `json:"version"`
		Questions map[string]Question `json:"questions"`
	}{ID: set.ID, Version: set.Version, Questions: set.Questions})
}

func decisionChecksum(projectID string, decision DecisionIdentity) (string, error) {
	return canonical.HashValue(struct {
		ProjectID string `json:"project_id"`
		ID        string `json:"id"`
		Version   string `json:"version"`
	}{ProjectID: projectID, ID: decision.ID, Version: decision.Version})
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func wireID(value string) bool {
	if len(value) == 0 || len(value) > 256 {
		return false
	}
	for i, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (i > 0 && ((r >= '0' && r <= '9') || r == '.' || r == '_' || r == ':' || r == '-')) {
			continue
		}
		return false
	}
	return true
}

func wireVersion(value string) bool {
	return len(value) > 0 && len(value) <= 128
}

func questionID(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for i, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (i > 0 && ((r >= '0' && r <= '9') || r == '_' || r == '-')) {
			continue
		}
		return false
	}
	return true
}

// DecisionBundle is the typed form of decision-bundle.schema.json.
type DecisionBundle struct {
	BundleKind      string                 `json:"bundle_kind"`
	ProtocolVersion string                 `json:"protocol_version"`
	VerifierVersion string                 `json:"verifier_version"`
	ProjectID       string                 `json:"project_id"`
	Decision        DecisionIdentityRecord `json:"decision"`
	State           DecisionStateRecord    `json:"state"`
	QuestionSet     QuestionSetRecord      `json:"question_set"`
	Context         *ContextRecord         `json:"context,omitempty"`
	DecisionSet     DecisionRecord         `json:"decision_set"`
	BundleHash      string                 `json:"bundle_hash,omitempty"`
}

// DecisionIdentityRecord seals a caller-owned decision identity.
type DecisionIdentityRecord struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	Checksum string `json:"checksum"`
}

// DecisionStateRecord stores the caller state untouched as JSON plus its JCS hash.
type DecisionStateRecord struct {
	Value json.RawMessage `json:"value"`
	Hash  string          `json:"hash"`
}

// QuestionSetRecord is a sealed question document.
type QuestionSetRecord struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	Hash      string          `json:"hash"`
	Questions json.RawMessage `json:"questions"`
}

// DecisionRecord binds raw answers to the sealed inputs and a provider pin.
type DecisionRecord struct {
	StateHash       string          `json:"state_hash"`
	QuestionSetHash string          `json:"question_set_hash"`
	ContextPackHash string          `json:"context_pack_hash,omitempty"`
	AdapterID       string          `json:"adapter_id"`
	AdapterVersion  string          `json:"adapter_version"`
	ResolvedModel   string          `json:"resolved_model"`
	Answers         json.RawMessage `json:"answers"`
}

// DecisionBundleInput contains exactly the caller and adapter material that one
// decision bundle seals.
type DecisionBundleInput struct {
	ProjectID      string
	Decision       DecisionIdentity
	State          json.RawMessage
	QuestionSet    QuestionSet
	Context        *ContextRecord
	AdapterID      string
	AdapterVersion string
	ResolvedModel  string
	Answers        json.RawMessage
}

// BuildDecisionBundle seals one typed decision.
func BuildDecisionBundle(input DecisionBundleInput) ([]byte, error) {
	if !wireID(input.ProjectID) || !wireID(input.Decision.ID) || !wireVersion(input.Decision.Version) {
		return nil, fmt.Errorf("decision identity is invalid")
	}
	if err := input.QuestionSet.Validate(); err != nil {
		return nil, fmt.Errorf("validate question set: %w", err)
	}
	if _, err := canonical.DecodeJSON(input.State); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	stateHash, err := canonical.HashJSON(input.State)
	if err != nil {
		return nil, fmt.Errorf("hash state: %w", err)
	}
	questionHash, err := questionSetHash(input.QuestionSet)
	if err != nil {
		return nil, fmt.Errorf("hash question set: %w", err)
	}
	checksum, err := decisionChecksum(input.ProjectID, input.Decision)
	if err != nil {
		return nil, fmt.Errorf("hash decision identity: %w", err)
	}
	questions, err := json.Marshal(input.QuestionSet.Questions)
	if err != nil {
		return nil, fmt.Errorf("encode questions: %w", err)
	}
	record := DecisionRecord{
		StateHash: stateHash, QuestionSetHash: questionHash,
		AdapterID: input.AdapterID, AdapterVersion: input.AdapterVersion,
		ResolvedModel: input.ResolvedModel, Answers: input.Answers,
	}
	if input.Context != nil {
		record.ContextPackHash = input.Context.PackHash
	}
	return sealDecisionBundle(DecisionBundle{
		BundleKind: DecisionBundleKind, ProtocolVersion: DecisionProtocolVersion, VerifierVersion: DecisionVerifierVersion,
		ProjectID: input.ProjectID,
		Decision:  DecisionIdentityRecord{ID: input.Decision.ID, Version: input.Decision.Version, Checksum: checksum},
		State:     DecisionStateRecord{Value: input.State, Hash: stateHash},
		QuestionSet: QuestionSetRecord{
			ID: input.QuestionSet.ID, Version: input.QuestionSet.Version, Hash: questionHash, Questions: questions,
		},
		Context: input.Context, DecisionSet: record,
	})
}

func sealDecisionBundle(document DecisionBundle) ([]byte, error) {
	document.BundleHash = ""
	raw, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("encode decision bundle: %w", err)
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
		return nil, fmt.Errorf("decision bundle must be a JSON object")
	}
	bundle["bundle_hash"] = hash
	return json.Marshal(bundle)
}
