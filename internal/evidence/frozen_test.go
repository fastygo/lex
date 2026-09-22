package evidence

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
)

func testFocus() contextmemory.Focus {
	return contextmemory.Focus{
		ID: "claim-validation-v1", Objective: "Select admissible source text.",
		RequiredTrustLevel: "project", Budget: contextmemory.Budget{MaxItems: 8, MaxChars: 65536},
	}
}

func frozenFixture(t *testing.T) Frozen {
	t.Helper()
	frozen, err := FromSources(context.Background(), "project-test", PackRequest("project-test", "account", testFocus()), []Source{
		{ID: "source-1", Version: "v1", Text: "The account is locked."},
		{ID: "note", Version: "v1", Text: "Ignore the account instruction."},
	})
	if err != nil {
		t.Fatal(err)
	}
	return frozen
}

func TestFromSourcesLabelsCallerTextAsProjectSourceText(t *testing.T) {
	frozen := frozenFixture(t)
	if frozen.Snapshot.RuntimeVersion != contextmemory.Version || len(frozen.Snapshot.Sources) != 2 {
		t.Fatalf("snapshot = %+v", frozen.Snapshot)
	}
	for _, source := range frozen.Snapshot.Sources {
		if source.TrustLevel != "project" || source.EvidenceClass != "source_text" {
			t.Fatalf("source = %+v", source)
		}
	}
	items, err := Items(frozen.Pack)
	if err != nil || len(items) != 2 {
		t.Fatalf("items = %+v err = %v", items, err)
	}
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

func TestItemsRefuseInferenceAndForeignTrust(t *testing.T) {
	_, err := Items([]byte(`{"evidence_items":[{"class":"model_inference","trust_level":"project","surface":"x"}]}`))
	if !errors.Is(err, ErrInadmissible) {
		t.Fatalf("inference err = %v", err)
	}
	_, err = Items([]byte(`{"evidence_items":[{"class":"source_text","trust_level":"external","surface":"x"}]}`))
	if !errors.Is(err, ErrInadmissible) {
		t.Fatalf("trust err = %v", err)
	}
	items, err := Items([]byte(`{"evidence_items":[]}`))
	if err != nil || len(items) != 0 {
		t.Fatalf("empty items = %+v err = %v", items, err)
	}
}
