// Package evidence is the LeX side of the Context evidence plane. It verifies a
// caller-frozen state by asking Context to rebuild it. Retrieval, selection,
// budgeting, and rejection belong to Context; this package never recomputes them.
package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	"github.com/fastygo/lex/internal/canonical"
)

// MaxSourceBytes bounds the source text Context may rebuild for one frozen state.
const MaxSourceBytes = 256 << 10

// Frozen is an immutable Context state: the pack plus the snapshot and pack
// request Context needs to reproduce it.
type Frozen struct {
	Pack        json.RawMessage
	Snapshot    contextmemory.Snapshot
	PackRequest contextmemory.PackRequest
}

var (
	// ErrSnapshotShape means the snapshot is not the pinned Context manifest.
	ErrSnapshotShape = errors.New("snapshot is not a Context manifest")
	// ErrRequestShape means the pack request is not the pinned Context request.
	ErrRequestShape = errors.New("pack request is not a Context pack request")
	// ErrSnapshotMismatch means Context does not reproduce the frozen snapshot.
	ErrSnapshotMismatch = errors.New("Context does not reproduce the frozen snapshot")
	// ErrPackMismatch means Context does not reproduce the frozen pack.
	ErrPackMismatch = errors.New("Context does not reproduce the frozen pack")
)

// Decode strictly decodes a wire snapshot and pack request. Unknown fields are
// refused so a bundle cannot smuggle material the verifier does not read.
func Decode(pack, snapshot, packRequest json.RawMessage) (Frozen, error) {
	frozen := Frozen{Pack: pack}
	if err := strict(snapshot, &frozen.Snapshot); err != nil {
		return Frozen{}, fmt.Errorf("%w: %v", ErrSnapshotShape, err)
	}
	if err := strict(packRequest, &frozen.PackRequest); err != nil {
		return Frozen{}, fmt.Errorf("%w: %v", ErrRequestShape, err)
	}
	return frozen, nil
}

func strict(raw json.RawMessage, dest any) error {
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

// Verify asks Context to rebuild the frozen state from its own snapshot sources
// and pack request, then compares. Cancellation is returned as is; every
// other failure is a mismatch, never a rewrite of the saved state.
func Verify(ctx context.Context, frozen Frozen) error {
	runtime, err := contextmemory.New(ctx, contextmemory.Config{
		ProjectID: frozen.Snapshot.ProjectID,
		Sources:   frozen.Snapshot.Sources,
		MaxBytes:  MaxSourceBytes,
	})
	if err != nil {
		return stopOr(ctx, err, ErrSnapshotMismatch)
	}
	if !snapshotEqual(runtime.Snapshot(), frozen.Snapshot) {
		return ErrSnapshotMismatch
	}
	result, err := runtime.ContextPack(ctx, frozen.PackRequest)
	if err != nil {
		return stopOr(ctx, err, ErrPackMismatch)
	}
	saved, err := canonical.HashJSON(frozen.Pack)
	if err != nil {
		return ErrPackMismatch
	}
	rebuilt, err := canonical.HashJSON(result.ContextPack)
	if err != nil || saved != rebuilt {
		return ErrPackMismatch
	}
	return nil
}

func stopOr(ctx context.Context, err, mismatch error) error {
	if ctx.Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return mismatch
}

func snapshotEqual(a, b contextmemory.Snapshot) bool {
	return a.ID == b.ID && a.ProjectID == b.ProjectID && a.RuntimeVersion == b.RuntimeVersion && slices.Equal(a.Sources, b.Sources)
}
