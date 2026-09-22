package wire

import (
	"context"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

// AdapterPin is deployment-owned authority for an adapter and resolved model.
type AdapterPin struct {
	Version string
	Model   string
}

// Verifier replays sealed bundles against an immutable copy of the deployment's
// profiles and adapter pins. It never retrieves evidence or calls a provider.
type Verifier struct {
	profiles profile.Registry
	adapters map[string]AdapterPin
}

// NewVerifier freezes the profiles this deployment replays and the adapter pins it trusts.
func NewVerifier(profiles profile.Registry, adapters map[string]AdapterPin) Verifier {
	frozen := make(map[string]AdapterPin, len(adapters))
	for id, pin := range adapters {
		frozen[id] = pin
	}
	return Verifier{profiles: profiles, adapters: frozen}
}

// Replay reproduces the verdict from a sealed bundle.
func (v Verifier) Replay(raw []byte) (Report, error) {
	return v.ReplayContext(context.Background(), raw)
}

// ReplayContext is Replay bound to the caller's cancellation and deadline.
func (v Verifier) ReplayContext(ctx context.Context, raw []byte) (Report, error) {
	if err := ctx.Err(); err != nil {
		return Report{}, err
	}
	bundle, err := DecodeBundle(raw)
	if err != nil {
		return Report{}, err
	}
	prof, findings := v.bindingFindings(bundle)
	if len(findings) > 0 {
		return resolve(findings)
	}
	if bundle.DecisionSet.Skipped {
		return v.skippedReport(ctx, prof, bundle)
	}
	answers, parseFindings := parseAnswers(bundle.DecisionSet.Answers)
	if len(parseFindings) > 0 {
		return resolve(parseFindings)
	}
	_, findings, err = prof.Interpret(answers)
	if err != nil {
		return Report{}, err
	}
	extra, err := CheckContext(ctx, prof, bundle.Entity.ProjectID, bundle.Context)
	if err != nil {
		return Report{}, err
	}
	return resolve(append(findings, extra...))
}

// skippedReport confirms that a skipped decision really had nothing to decide.
func (v Verifier) skippedReport(ctx context.Context, prof profile.Profile, bundle Bundle) (Report, error) {
	findings, err := CheckContext(ctx, prof, bundle.Entity.ProjectID, bundle.Context)
	if err != nil {
		return Report{}, err
	}
	if len(findings) == 0 {
		findings = []verify.Finding{errorFinding(verify.CodeBindingMismatch)}
	}
	return resolve(findings)
}

func resolve(findings []verify.Finding) (Report, error) {
	verdict, err := verify.ResolveVerdict(findings)
	if err != nil {
		return Report{}, err
	}
	if findings == nil {
		findings = []verify.Finding{}
	}
	return Report{Verdict: verdict, Findings: findings}, nil
}

// bindingFindings checks that the bundle names a profile, adapter, and model
// this deployment pins and that every hash binds to the material beside it.
func (v Verifier) bindingFindings(bundle Bundle) (profile.Profile, []verify.Finding) {
	findings := make([]verify.Finding, 0)
	if bundle.VerifierVersion != VerifierVersion {
		findings = append(findings, errorFinding(verify.CodeUnpinnedVerifier))
	}
	prof, known := v.profiles.Lookup(profile.Ref{
		QuestionSetID: bundle.QuestionSet.ID, QuestionSetVersion: bundle.QuestionSet.Version,
		PolicyID: bundle.Policy.ID, PolicyVersion: bundle.Policy.Version,
	})
	switch {
	case !known:
		findings = append(findings, errorFinding(verify.CodeUnpinnedQuestionSet), errorFinding(verify.CodeUnpinnedPolicy))
	default:
		content, err := questionSetHash(bundle.QuestionSet.ID, bundle.QuestionSet.Version, bundle.QuestionSet.Questions)
		if err != nil || content != prof.QuestionSetHash() || bundle.QuestionSet.Hash != prof.QuestionSetHash() {
			findings = append(findings, errorFinding(verify.CodeUnpinnedQuestionSet))
		}
		if bundle.Policy.Hash != prof.PolicyHash() {
			findings = append(findings, errorFinding(verify.CodeUnpinnedPolicy))
		}
		if !prof.AcceptsEntity(bundle.Entity.Type, bundle.Entity.SchemaVersion) {
			findings = append(findings, errorFinding(verify.CodeUnpinnedEntity))
		}
	}
	decision := bundle.DecisionSet
	packHash, err := canonical.HashJSON(bundle.Context.Pack)
	if err != nil || packHash != bundle.Context.PackHash || decision.ContextPackHash != bundle.Context.PackHash || decision.QuestionSetHash != bundle.QuestionSet.Hash || decision.PolicyHash != bundle.Policy.Hash {
		findings = append(findings, errorFinding(verify.CodeBindingMismatch))
	}
	checksum, err := entityChecksum(bundle.Entity.ID, bundle.Entity.ProjectID, bundle.Entity.Type, bundle.Entity.SchemaVersion, bundle.Entity.Version)
	if err != nil || checksum != bundle.Entity.Checksum {
		findings = append(findings, errorFinding(verify.CodeEntityChecksumMismatch))
	}
	if decision.Skipped {
		if decision.AdapterID != "" || decision.AdapterVersion != "" || decision.ResolvedModel != "" || len(decision.Answers) != 0 {
			findings = append(findings, errorFinding(verify.CodeBindingMismatch))
		}
		return prof, findings
	}
	pin, ok := v.adapters[decision.AdapterID]
	switch {
	case !ok:
		findings = append(findings, errorFinding(verify.CodeUnknownAdapter))
	default:
		if pin.Version == "" || decision.AdapterVersion != pin.Version {
			findings = append(findings, errorFinding(verify.CodeUnpinnedAdapter))
		}
		if pin.Model == "" || decision.ResolvedModel != pin.Model {
			findings = append(findings, errorFinding(verify.CodeUnresolvedModel))
		}
	}
	return prof, findings
}

func errorFinding(code string) verify.Finding {
	return verify.Finding{Code: code, Verdict: verify.VerdictError, Detail: "replay binding failed " + code}
}
