package wire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
)

// evidenceItem is the part of a Context evidence item LeX reads. The pack is
// Context's document; unknown fields are Context's business.
type evidenceItem struct {
	Class      string `json:"class"`
	TrustLevel string `json:"trust_level"`
	Surface    string `json:"surface"`
	SourceRef  struct {
		SourceID  string `json:"source_id"`
		ProjectID string `json:"project_id"`
		Checksum  string `json:"checksum"`
		Span      *struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"span"`
	} `json:"source_ref"`
}

// CheckContext applies the LeX evidence controls to one frozen state and asks
// Context to reproduce it. It is the same check for a live evaluation that
// accepts a caller-frozen state and for a replay. It never recomputes
// retrieval, selection, or rejection; Context does.
func CheckContext(ctx context.Context, prof profile.Profile, projectID string, record ContextRecord) ([]verify.Finding, error) {
	items, ok := evidenceItems(record.Pack)
	if !ok {
		return []verify.Finding{{Code: verify.CodePackShape, Verdict: verify.VerdictError, Detail: "frozen pack has no evidence item list"}}, nil
	}
	frozen, err := evidence.Decode(record.Pack, record.Snapshot, record.PackRequest)
	if err != nil {
		if errors.Is(err, evidence.ErrRequestShape) {
			return []verify.Finding{{Code: verify.CodeUnpinnedFocus, Verdict: verify.VerdictError, Detail: "frozen pack request is not a Context pack request"}}, nil
		}
		return []verify.Finding{{Code: verify.CodeSnapshotIdentity, Verdict: verify.VerdictError, Detail: "frozen snapshot is not a Context manifest"}}, nil
	}
	admissible, inference := 0, 0
	findings := make([]verify.Finding, 0)
	for _, item := range items {
		switch item.Class {
		case "instruction", "policy":
			findings = append(findings, verify.Finding{Code: verify.CodeInstructionInEvidence, Verdict: verify.VerdictError, Detail: "instructions and policy are not evidence"})
		case "model_inference":
			inference++
		case "source_text":
			if item.TrustLevel != "project" {
				break
			}
			if finding, failed := provenanceFinding(projectID, frozen.Snapshot, item); failed {
				findings = append(findings, finding)
				break
			}
			admissible++
		}
	}
	if len(findings) > 0 {
		return findings, nil
	}
	if admissible == 0 && inference > 0 {
		return []verify.Finding{{Code: verify.CodeInferenceOnly, Verdict: verify.VerdictInsufficient, Detail: "model inference cannot establish a factual claim"}}, nil
	}
	if frozen.Snapshot.ProjectID != projectID || frozen.Snapshot.RuntimeVersion != contextmemory.Version {
		return []verify.Finding{{Code: verify.CodeProjectBinding, Verdict: verify.VerdictError, Detail: "frozen snapshot is not bound to the entity project and pinned runtime"}}, nil
	}
	if !focusPinned(prof, projectID, frozen.PackRequest) {
		return []verify.Finding{{Code: verify.CodeUnpinnedFocus, Verdict: verify.VerdictError, Detail: "frozen pack request does not use the profile focus"}}, nil
	}
	switch err := evidence.Verify(ctx, frozen); {
	case err == nil:
	case errors.Is(err, evidence.ErrSnapshotMismatch):
		return []verify.Finding{{Code: verify.CodeSnapshotIdentity, Verdict: verify.VerdictError, Detail: "Context does not reproduce the frozen snapshot"}}, nil
	case errors.Is(err, evidence.ErrPackMismatch):
		return []verify.Finding{{Code: verify.CodePackRebuild, Verdict: verify.VerdictError, Detail: "frozen pack does not match the pack rebuilt from the snapshot and request"}}, nil
	default:
		return nil, err
	}
	if admissible == 0 {
		return []verify.Finding{{Code: verify.CodeNoEligibleEvidence, Verdict: verify.VerdictInsufficient, Detail: "the frozen pack contains no admissible source text"}}, nil
	}
	return nil, nil
}

func evidenceItems(pack json.RawMessage) ([]evidenceItem, bool) {
	var view struct {
		EvidenceItems json.RawMessage `json:"evidence_items"`
	}
	if err := json.Unmarshal(pack, &view); err != nil {
		return nil, false
	}
	if len(view.EvidenceItems) == 0 || string(view.EvidenceItems) == "null" {
		return nil, true
	}
	var items []evidenceItem
	if err := json.Unmarshal(view.EvidenceItems, &items); err != nil {
		return nil, false
	}
	return items, true
}

// focusPinned checks that the pack request carries exactly the profile focus
// for the entity project and no caller controls.
func focusPinned(prof profile.Profile, projectID string, req contextmemory.PackRequest) bool {
	if req.ProjectID != projectID || req.Query == "" || req.TaskID != "" || len(req.Instructions) > 0 || len(req.PolicyRefs) > 0 || len(req.VerificationRequirements) > 0 {
		return false
	}
	return req.Focus == prof.Focus()
}

// provenanceFinding checks one admissible item against its frozen source:
// complete identity, project binding, byte checksum, and full-source surface.
func provenanceFinding(projectID string, snapshot contextmemory.Snapshot, item evidenceItem) (verify.Finding, bool) {
	ref := item.SourceRef
	if item.Surface == "" || ref.SourceID == "" || ref.Checksum == "" || ref.ProjectID == "" || ref.ProjectID != projectID {
		return verify.Finding{Code: verify.CodeMissingProvenance, Verdict: verify.VerdictError, Detail: "admissible evidence has no complete source identity"}, true
	}
	sum := sha256.Sum256([]byte(item.Surface))
	if hex.EncodeToString(sum[:]) != ref.Checksum {
		return verify.Finding{Code: verify.CodeChecksumMismatch, Verdict: verify.VerdictError, Detail: "evidence surface does not match its source checksum"}, true
	}
	for _, source := range snapshot.Sources {
		if source.SourceID != ref.SourceID {
			continue
		}
		if source.Version == "" || source.TrustLevel != "project" || source.EvidenceClass != "source_text" {
			return verify.Finding{Code: verify.CodeSourceAdmission, Verdict: verify.VerdictError, Detail: "frozen source is not admissible project text"}, true
		}
		excerpt := source.Text
		if ref.Span != nil {
			if ref.Span.Start < 0 || ref.Span.End < ref.Span.Start || ref.Span.End > int64(len(source.Text)) {
				return verify.Finding{Code: verify.CodeChecksumMismatch, Verdict: verify.VerdictError, Detail: "evidence span does not fit the frozen source"}, true
			}
			excerpt = source.Text[ref.Span.Start:ref.Span.End]
		}
		if excerpt != item.Surface {
			return verify.Finding{Code: verify.CodeChecksumMismatch, Verdict: verify.VerdictError, Detail: "evidence surface does not match the frozen source"}, true
		}
		if excerpt != source.Text {
			return verify.Finding{Code: verify.CodePartialSurface, Verdict: verify.VerdictError, Detail: "evidence surface must be the full frozen source"}, true
		}
		return verify.Finding{}, false
	}
	return verify.Finding{Code: verify.CodeMissingProvenance, Verdict: verify.VerdictError, Detail: "admissible evidence does not resolve to a frozen source"}, true
}
