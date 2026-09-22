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

// Kind names one way evidence enters an evaluation. Both kinds end in the
// same Frozen state and the same verifier checks.
type Kind string

const (
	// KindSources means the caller sends text and LeX freezes it with the
	// embedded Context runtime.
	KindSources Kind = "sources"
	// KindFrozenContext means the caller sends a pack, snapshot, and pack
	// request it already obtained from Context; LeX only verifies the rebuild.
	KindFrozenContext Kind = "frozen_context"
)

// Kinds lists the evidence inputs this slice accepts, in disclosure order.
func Kinds() []Kind { return []Kind{KindSources, KindFrozenContext} }

// Frozen is the immutable evidence state shared by the decision plane, the
// replay bundle, and the verifier.
type Frozen struct {
	Pack        json.RawMessage
	Snapshot    contextmemory.Snapshot
	PackRequest contextmemory.PackRequest
}

// Source is caller text before LeX labels it. LeX is the authorized host, so
// every source it freezes is project-trusted source text.
type Source struct {
	ID      string
	Version string
	Text    string
}

// PackRequest is the only retrieval control LeX sends for one evaluation.
func PackRequest(projectID, query string, focus contextmemory.Focus) contextmemory.PackRequest {
	return contextmemory.PackRequest{ProjectID: projectID, Query: query, Focus: focus}
}

// FromSources freezes caller text with the embedded Context runtime.
func FromSources(ctx context.Context, projectID string, request contextmemory.PackRequest, sources []Source) (Frozen, error) {
	labelled := make([]contextmemory.Source, len(sources))
	for i, source := range sources {
		labelled[i] = contextmemory.Source{
			SourceID: source.ID, Version: source.Version, Text: source.Text,
			TrustLevel: "project", EvidenceClass: "source_text",
		}
	}
	result, err := BuildPack(ctx, projectID, labelled, request)
	if err != nil {
		return Frozen{}, err
	}
	return Frozen{Pack: result.ContextPack, Snapshot: result.Snapshot, PackRequest: request}, nil
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
	runtime, err := newRuntime(ctx, frozen.Snapshot.ProjectID, frozen.Snapshot.Sources)
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

// Item is one admissible evidence item in the shape sent to a decision adapter.
type Item struct {
	Class      string `json:"class"`
	TrustLevel string `json:"trust_level"`
	Surface    string `json:"surface"`
	SourceID   string `json:"source_id"`
}

// ErrInadmissible means an evidence item is not project-trusted source text.
var ErrInadmissible = errors.New("ineligible evidence")

type packView struct {
	EvidenceItems []struct {
		Class      string `json:"class"`
		TrustLevel string `json:"trust_level"`
		Surface    string `json:"surface"`
		SourceRef  struct {
			SourceID string `json:"source_id"`
		} `json:"source_ref"`
	} `json:"evidence_items"`
}

// Items lists the evidence items of a frozen pack for the decision plane.
// Any item that is not project source text makes the whole pack inadmissible.
func Items(pack json.RawMessage) ([]Item, error) {
	var view packView
	if err := json.Unmarshal(pack, &view); err != nil {
		return nil, err
	}
	items := make([]Item, 0, len(view.EvidenceItems))
	for _, item := range view.EvidenceItems {
		if item.Class != "source_text" || item.TrustLevel != "project" || item.Surface == "" {
			return nil, ErrInadmissible
		}
		items = append(items, Item{Class: item.Class, TrustLevel: item.TrustLevel, Surface: item.Surface, SourceID: item.SourceRef.SourceID})
	}
	return items, nil
}
