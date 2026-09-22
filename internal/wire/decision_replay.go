package wire

import (
	"bytes"
	"context"
	"crypto/subtle"
	_ "embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/verify"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const decisionBundleSchemaID = "https://lex.fastygo.dev/schema/v0.2/decision-bundle"

//go:embed schema/decision-bundle.schema.json
var decisionBundleSchema []byte

var (
	decisionBundleSchemaOnce sync.Once
	compiledDecisionBundle   *jsonschema.Schema
	decisionBundleSchemaErr  error
)

// DecisionReport is a structural verification report. It intentionally has no
// semantic verdict: question meaning and downstream action belong to the caller.
type DecisionReport struct {
	StructuralStatus string           `json:"structural_status"`
	Findings         []verify.Finding `json:"findings"`
}

// AdapterPin is deployment-owned authority for an adapter and resolved model.
type AdapterPin struct {
	Version string
	Model   string
}

// DecisionVerifier replays decision bundles against immutable adapter pins.
// It never retrieves evidence or calls a provider.
type DecisionVerifier struct {
	adapters map[string]AdapterPin
}

// NewDecisionVerifier freezes trusted adapter and model pins.
func NewDecisionVerifier(adapters map[string]AdapterPin) DecisionVerifier {
	pins := make(map[string]AdapterPin, len(adapters))
	for id, pin := range adapters {
		pins[id] = pin
	}
	return DecisionVerifier{adapters: pins}
}

// ValidateDecisionBundle checks the decision-bundle schema.
func ValidateDecisionBundle(raw []byte) error {
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return fmt.Errorf("decode decision bundle: %w", err)
	}
	schema, err := compiledBundleSchema()
	if err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate decision bundle: %w", err)
	}
	return nil
}

// VerifyDecisionBundleHash validates the self-hash of a decision bundle.
func VerifyDecisionBundleHash(raw []byte) error {
	if err := ValidateDecisionBundle(raw); err != nil {
		return err
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return fmt.Errorf("decode decision bundle: %w", err)
	}
	bundle, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("decision bundle must be an object")
	}
	declared, ok := bundle["bundle_hash"].(string)
	if !ok {
		return fmt.Errorf("decision bundle hash must be a string")
	}
	delete(bundle, "bundle_hash")
	computed, err := canonical.HashValue(bundle)
	if err != nil {
		return fmt.Errorf("hash decision bundle: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(declared), []byte(computed)) != 1 {
		return fmt.Errorf("decision bundle hash does not match")
	}
	return nil
}

// DecodeDecisionBundle decodes only a schema-validated and self-hashed bundle.
func DecodeDecisionBundle(raw []byte) (DecisionBundle, error) {
	if err := VerifyDecisionBundleHash(raw); err != nil {
		return DecisionBundle{}, err
	}
	var bundle DecisionBundle
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&bundle); err != nil {
		return DecisionBundle{}, fmt.Errorf("decode decision bundle: %w", err)
	}
	return bundle, nil
}

// Replay reproduces structural validation without retrieval or a provider call.
func (v DecisionVerifier) Replay(raw []byte) (DecisionReport, error) {
	return v.ReplayContext(context.Background(), raw)
}

// ReplayContext is Replay bound to cancellation and deadline.
func (v DecisionVerifier) ReplayContext(ctx context.Context, raw []byte) (DecisionReport, error) {
	if err := ctx.Err(); err != nil {
		return DecisionReport{}, err
	}
	bundle, err := DecodeDecisionBundle(raw)
	if err != nil {
		return DecisionReport{}, err
	}
	findings, set := v.bindingFindings(bundle)
	if len(findings) > 0 {
		return decisionReport(findings), nil
	}
	answers, parseFindings := parseAnswers(bundle.DecisionSet.Answers)
	if len(parseFindings) > 0 {
		return decisionReport(parseFindings), nil
	}
	findings = verify.ValidateAnswers(set.VerifierQuestions(), answers)
	if bundle.Context != nil {
		extra, err := CheckContext(ctx, bundle.ProjectID, *bundle.Context)
		if err != nil {
			return DecisionReport{}, err
		}
		findings = append(findings, extra...)
	}
	return decisionReport(findings), nil
}

func (v DecisionVerifier) bindingFindings(bundle DecisionBundle) ([]verify.Finding, QuestionSet) {
	findings := make([]verify.Finding, 0)
	if bundle.BundleKind != DecisionBundleKind || bundle.ProtocolVersion != DecisionProtocolVersion || bundle.VerifierVersion != DecisionVerifierVersion {
		findings = append(findings, errorFinding(verify.CodeUnpinnedVerifier))
	}
	if !wireID(bundle.ProjectID) || !wireID(bundle.Decision.ID) || !wireVersion(bundle.Decision.Version) {
		findings = append(findings, errorFinding(verify.CodeBindingMismatch))
	}
	checksum, err := decisionChecksum(bundle.ProjectID, DecisionIdentity{ID: bundle.Decision.ID, Version: bundle.Decision.Version})
	if err != nil || checksum != bundle.Decision.Checksum {
		findings = append(findings, errorFinding(verify.CodeDecisionChecksumMismatch))
	}
	stateHash, err := canonical.HashJSON(bundle.State.Value)
	if err != nil || stateHash != bundle.State.Hash || bundle.DecisionSet.StateHash != bundle.State.Hash {
		findings = append(findings, errorFinding(verify.CodeBindingMismatch))
	}
	var questions map[string]Question
	if err := strictJSON(bundle.QuestionSet.Questions, &questions); err != nil {
		findings = append(findings, errorFinding(verify.CodeUnpinnedQuestionSet))
		return findings, QuestionSet{}
	}
	set := QuestionSet{ID: bundle.QuestionSet.ID, Version: bundle.QuestionSet.Version, Questions: questions}
	hash, err := questionSetHash(set)
	if err != nil || set.Validate() != nil || hash != bundle.QuestionSet.Hash || bundle.DecisionSet.QuestionSetHash != bundle.QuestionSet.Hash {
		findings = append(findings, errorFinding(verify.CodeUnpinnedQuestionSet))
	}
	if bundle.Context == nil {
		if bundle.DecisionSet.ContextPackHash != "" {
			findings = append(findings, errorFinding(verify.CodeBindingMismatch))
		}
	} else {
		packHash, err := canonical.HashJSON(bundle.Context.Pack)
		if err != nil || packHash != bundle.Context.PackHash || bundle.DecisionSet.ContextPackHash != bundle.Context.PackHash {
			findings = append(findings, errorFinding(verify.CodeBindingMismatch))
		}
	}
	pin, ok := v.adapters[bundle.DecisionSet.AdapterID]
	switch {
	case !ok:
		findings = append(findings, errorFinding(verify.CodeUnknownAdapter))
	default:
		if pin.Version == "" || pin.Version != bundle.DecisionSet.AdapterVersion {
			findings = append(findings, errorFinding(verify.CodeUnpinnedAdapter))
		}
		if pin.Model == "" || pin.Model != bundle.DecisionSet.ResolvedModel {
			findings = append(findings, errorFinding(verify.CodeUnresolvedModel))
		}
	}
	return findings, set
}

func decisionReport(findings []verify.Finding) DecisionReport {
	if findings == nil {
		findings = []verify.Finding{}
	}
	status := "valid"
	if len(findings) > 0 {
		status = "invalid"
	}
	return DecisionReport{StructuralStatus: status, Findings: findings}
}

func errorFinding(code string) verify.Finding {
	return verify.Finding{Code: code, Detail: "deterministic check failed " + code}
}

func strictJSON(raw json.RawMessage, dest any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("trailing data")
	}
	return nil
}

func compiledBundleSchema() (*jsonschema.Schema, error) {
	decisionBundleSchemaOnce.Do(func() {
		value, err := canonical.DecodeJSON(decisionBundleSchema)
		if err != nil {
			decisionBundleSchemaErr = fmt.Errorf("decode decision bundle schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		if err := compiler.AddResource(decisionBundleSchemaID, value); err != nil {
			decisionBundleSchemaErr = fmt.Errorf("register decision bundle schema: %w", err)
			return
		}
		compiledDecisionBundle, decisionBundleSchemaErr = compiler.Compile(decisionBundleSchemaID)
	})
	return compiledDecisionBundle, decisionBundleSchemaErr
}
