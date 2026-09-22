package wire

import (
	"context"
	"encoding/json"
	"errors"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/evidence"
	"github.com/fastygo/lex/internal/verify"
)

// ContextRecord is an optional frozen Context state. Pack, snapshot, and pack
// request keep Context's own JSON; the verifier decodes them through the
// evidence plane.
type ContextRecord struct {
	Pack        json.RawMessage `json:"pack"`
	Snapshot    json.RawMessage `json:"snapshot"`
	PackRequest json.RawMessage `json:"pack_request"`
	PackHash    string          `json:"pack_hash"`
}

// CheckContext validates an optional Context binding without imposing a focus,
// evidence taxonomy, or domain policy on caller state. It never recomputes
// retrieval or selection: Context rebuilds the frozen state and LeX compares.
func CheckContext(ctx context.Context, projectID string, record ContextRecord) ([]verify.Finding, error) {
	frozen, err := evidence.Decode(record.Pack, record.Snapshot, record.PackRequest)
	if err != nil {
		return []verify.Finding{errorFinding(verify.CodePackShape)}, nil
	}
	if frozen.Snapshot.ProjectID != projectID || frozen.PackRequest.ProjectID != projectID || frozen.Snapshot.RuntimeVersion != contextmemory.Version {
		return []verify.Finding{errorFinding(verify.CodeProjectBinding)}, nil
	}
	switch err := evidence.Verify(ctx, frozen); {
	case err == nil:
		return nil, nil
	case errors.Is(err, evidence.ErrSnapshotMismatch):
		return []verify.Finding{errorFinding(verify.CodeSnapshotIdentity)}, nil
	case errors.Is(err, evidence.ErrPackMismatch):
		return []verify.Finding{errorFinding(verify.CodePackRebuild)}, nil
	default:
		return nil, err
	}
}
