// Package profile defines what one deployment pins for a question set:
// the typed questions, the policy gate, the entity kind, and the Context focus.
// It contains no profile of its own; concrete profiles live in subpackages and
// are composed into a Registry by the deployment.
package profile

import (
	"encoding/json"
	"fmt"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/verify"
)

// Question is one typed question as sent to a decision adapter and sealed in a bundle.
type Question struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions"`
	Criteria     map[string]string `json:"criteria,omitempty"`
}

// QuestionSet is the versioned question document. Its hash covers id, version, and questions.
type QuestionSet struct {
	ID        string              `json:"id"`
	Version   string              `json:"version"`
	Questions map[string]Question `json:"questions"`
}

// Policy names one gate and the document whose hash pins it.
type Policy struct {
	ID          string
	Version     string
	Calibration string
	// Document is the hashed policy body. It must marshal deterministically.
	Document any
	// Gate applies the policy to validated answers.
	Gate verify.Gate
}

// Entity pins the entity kind this profile evaluates.
type Entity struct {
	Type          string
	SchemaVersion string
}

// Profile is one immutable, deployment-owned evaluation contract.
// Build it with New; the zero value is not usable.
type Profile struct {
	questionSet     QuestionSet
	policy          Policy
	entity          Entity
	focus           contextmemory.Focus
	typed           map[string]verify.Question
	state           func(subject string, evidence any) map[string]any
	questionSetHash string
	policyHash      string
}

// Spec is the material New freezes into a Profile.
type Spec struct {
	QuestionSet QuestionSet
	Policy      Policy
	Entity      Entity
	// Focus is the only Context focus this profile sends and accepts.
	Focus contextmemory.Focus
	// Typed is the verifier view of QuestionSet.Questions.
	Typed map[string]verify.Question
	// State builds the frozen state sent to a decision adapter.
	State func(subject string, evidence any) map[string]any
}

// New validates and hashes a specification.
func New(spec Spec) (Profile, error) {
	if spec.QuestionSet.ID == "" || spec.QuestionSet.Version == "" || len(spec.QuestionSet.Questions) == 0 {
		return Profile{}, fmt.Errorf("profile question set must have id, version, and questions")
	}
	if spec.Policy.ID == "" || spec.Policy.Version == "" || spec.Policy.Document == nil || spec.Policy.Gate == nil {
		return Profile{}, fmt.Errorf("profile policy must have id, version, document, and gate")
	}
	if !spec.Policy.Gate.Usable() {
		return Profile{}, fmt.Errorf("profile policy gate is not usable")
	}
	if spec.Entity.Type == "" || spec.Entity.SchemaVersion == "" {
		return Profile{}, fmt.Errorf("profile entity must have type and schema version")
	}
	if spec.Focus.ID == "" || spec.Focus.Objective == "" || spec.Focus.RequiredTrustLevel == "" || spec.Focus.Budget.MaxItems < 1 || spec.Focus.Budget.MaxChars < 1 {
		return Profile{}, fmt.Errorf("profile focus must be complete")
	}
	if len(spec.Typed) != len(spec.QuestionSet.Questions) {
		return Profile{}, fmt.Errorf("typed questions must mirror the question set")
	}
	for id, question := range spec.QuestionSet.Questions {
		typed, ok := spec.Typed[id]
		if !ok || string(typed.Type) != question.Type {
			return Profile{}, fmt.Errorf("typed question %q does not mirror the question set", id)
		}
	}
	if spec.State == nil {
		return Profile{}, fmt.Errorf("profile must build a decision state")
	}
	questionSetHash, err := canonical.HashValue(spec.QuestionSet)
	if err != nil {
		return Profile{}, fmt.Errorf("hash question set: %w", err)
	}
	policyHash, err := canonical.HashValue(spec.Policy.Document)
	if err != nil {
		return Profile{}, fmt.Errorf("hash policy: %w", err)
	}
	typed := make(map[string]verify.Question, len(spec.Typed))
	for id, question := range spec.Typed {
		typed[id] = question
	}
	return Profile{
		questionSet: spec.QuestionSet, policy: spec.Policy, entity: spec.Entity, focus: spec.Focus,
		typed: typed, state: spec.State, questionSetHash: questionSetHash, policyHash: policyHash,
	}, nil
}

// MustNew is New for package-level profile constants.
func MustNew(spec Spec) Profile {
	prof, err := New(spec)
	if err != nil {
		panic(err)
	}
	return prof
}

// QuestionSet returns the sealed question document.
func (p Profile) QuestionSet() QuestionSet { return p.questionSet }

// QuestionSetHash is the canonical hash of the question document.
func (p Profile) QuestionSetHash() string { return p.questionSetHash }

// Policy returns the policy reference and gate.
func (p Profile) Policy() Policy { return p.policy }

// PolicyHash is the canonical hash of the policy document.
func (p Profile) PolicyHash() string { return p.policyHash }

// Entity returns the pinned entity kind.
func (p Profile) Entity() Entity { return p.entity }

// Focus returns the only Context focus this profile uses.
func (p Profile) Focus() contextmemory.Focus { return p.focus }

// AcceptsEntity reports whether the entity kind is the pinned one.
func (p Profile) AcceptsEntity(entityType, schemaVersion string) bool {
	return entityType == p.entity.Type && schemaVersion == p.entity.SchemaVersion
}

// TypedQuestions returns a copy of the verifier view of the questions.
func (p Profile) TypedQuestions() map[string]verify.Question {
	typed := make(map[string]verify.Question, len(p.typed))
	for id, question := range p.typed {
		typed[id] = question
	}
	return typed
}

// ProviderQuestions returns the question map sent to a decision adapter.
func (p Profile) ProviderQuestions() map[string]any {
	raw, err := json.Marshal(p.questionSet.Questions)
	if err != nil {
		panic(err)
	}
	var questions map[string]any
	if err := json.Unmarshal(raw, &questions); err != nil {
		panic(err)
	}
	return questions
}

// State builds the frozen state sent to a decision adapter.
func (p Profile) State(subject string, evidence any) map[string]any {
	return p.state(subject, evidence)
}

// Interpret applies answer validation and this profile's gate.
func (p Profile) Interpret(answers map[string]verify.Answer) (verify.Verdict, []verify.Finding, error) {
	return verify.Interpret(p.typed, p.policy.Gate, answers)
}

// Ref names a question set and policy as they appear in a replay bundle.
type Ref struct {
	QuestionSetID      string
	QuestionSetVersion string
	PolicyID           string
	PolicyVersion      string
}

// Ref returns the bundle reference of this profile.
func (p Profile) Ref() Ref {
	return Ref{
		QuestionSetID: p.questionSet.ID, QuestionSetVersion: p.questionSet.Version,
		PolicyID: p.policy.ID, PolicyVersion: p.policy.Version,
	}
}

// Registry is the immutable set of profiles one deployment evaluates and replays.
type Registry struct {
	byRef   map[Ref]Profile
	ordered []Profile
}

// NewRegistry freezes profiles. The first profile is the one live evaluations use.
func NewRegistry(profiles ...Profile) (Registry, error) {
	if len(profiles) == 0 {
		return Registry{}, fmt.Errorf("registry needs at least one profile")
	}
	byRef := make(map[Ref]Profile, len(profiles))
	ordered := make([]Profile, 0, len(profiles))
	for _, prof := range profiles {
		ref := prof.Ref()
		if ref.QuestionSetID == "" {
			return Registry{}, fmt.Errorf("registry received an unbuilt profile")
		}
		if _, duplicate := byRef[ref]; duplicate {
			return Registry{}, fmt.Errorf("registry received %s/%s twice", ref.QuestionSetID, ref.QuestionSetVersion)
		}
		byRef[ref] = prof
		ordered = append(ordered, prof)
	}
	return Registry{byRef: byRef, ordered: ordered}, nil
}

// MustRegistry is NewRegistry for composition roots.
func MustRegistry(profiles ...Profile) Registry {
	registry, err := NewRegistry(profiles...)
	if err != nil {
		panic(err)
	}
	return registry
}

// Lookup returns the profile a bundle names, if this deployment pins it.
func (r Registry) Lookup(ref Ref) (Profile, bool) {
	prof, ok := r.byRef[ref]
	return prof, ok
}

// Default is the profile used for live evaluations.
func (r Registry) Default() (Profile, bool) {
	if len(r.ordered) == 0 {
		return Profile{}, false
	}
	return r.ordered[0], true
}
