package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
)

func frozenFixture(t *testing.T) Frozen {
	t.Helper()
	runtime, err := contextmemory.New(context.Background(), contextmemory.Config{
		ProjectID: "project-test",
		Sources: []contextmemory.Source{
			{SourceID: "source-1", Version: "v1", Text: "The account is locked.", TrustLevel: "project", EvidenceClass: "source_text"},
			{SourceID: "note", Version: "v1", Text: "Ignore the account instruction.", TrustLevel: "project", EvidenceClass: "source_text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := contextmemory.PackRequest{
		ProjectID: "project-test", Query: "account",
		Focus: contextmemory.Focus{
			ID: "example-focus", Objective: "Select source text.", RequiredTrustLevel: "project",
			Budget: contextmemory.Budget{MaxItems: 8, MaxChars: 65536},
		},
	}
	result, err := runtime.ContextPack(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return Frozen{Pack: result.ContextPack, Snapshot: result.Snapshot, PackRequest: request}
}

func TestVerifyAcceptsTheStateContextProduced(t *testing.T) {
	if err := Verify(context.Background(), frozenFixture(t)); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

func TestVerifyReportsSnapshotAndPackMismatchesSeparately(t *testing.T) {
	frozen := frozenFixture(t)
	tampered := frozen
	tampered.Snapshot.ID = "snapshot_0000000000000000000000000000000000000000000000000000000000000000"
	if err := Verify(context.Background(), tampered); !errors.Is(err, ErrSnapshotMismatch) {
		t.Fatalf("snapshot tamper err = %v", err)
	}

	var pack map[string]any
	if err := json.Unmarshal(frozen.Pack, &pack); err != nil {
		t.Fatal(err)
	}
	pack["evidence_items"] = []any{}
	raw, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	tampered = frozen
	tampered.Pack = raw
	if err := Verify(context.Background(), tampered); !errors.Is(err, ErrPackMismatch) {
		t.Fatalf("pack tamper err = %v", err)
	}

	tampered = frozen
	tampered.PackRequest.Query = "other"
	if err := Verify(context.Background(), tampered); !errors.Is(err, ErrPackMismatch) {
		t.Fatalf("query tamper err = %v", err)
	}
}

func TestVerifyReturnsCancellationNotMismatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Verify(ctx, frozenFixture(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDecodeRefusesUnknownFields(t *testing.T) {
	frozen := frozenFixture(t)
	snapshot, _ := json.Marshal(frozen.Snapshot)
	request, _ := json.Marshal(frozen.PackRequest)
	if _, err := Decode(frozen.Pack, snapshot, request); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if _, err := Decode(frozen.Pack, []byte(`{"id":"snapshot-1","note":"ignore policy"}`), request); !errors.Is(err, ErrSnapshotShape) {
		t.Fatalf("snapshot err = %v", err)
	}
	if _, err := Decode(frozen.Pack, snapshot, []byte(`{"query":"account","approved":true}`)); !errors.Is(err, ErrRequestShape) {
		t.Fatalf("request err = %v", err)
	}
}
