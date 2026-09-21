package evidence

import (
	"context"
	"testing"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
)

func TestBuildPackUsesPinnedContextRuntime(t *testing.T) {
	result, err := BuildPack(
		context.Background(),
		"project-test",
		[]contextmemory.Source{{
			SourceID:      "source-1",
			Version:       "v1",
			Text:          "The account is locked.",
			TrustLevel:    "project",
			EvidenceClass: "source_text",
		}},
		contextmemory.PackRequest{
			ProjectID: "project-test",
			Query:     "account",
			Focus: contextmemory.Focus{
				ID:                 "account-check-v1",
				Objective:          "Find account evidence.",
				RequiredTrustLevel: "project",
				Budget: contextmemory.Budget{
					MaxItems: 1,
					MaxChars: 4096,
				},
			},
		},
	)
	if err != nil {
		t.Fatalf("BuildPack() error = %v", err)
	}
	if result.Snapshot.RuntimeVersion != contextmemory.Version {
		t.Fatalf("runtime version = %q, want %q", result.Snapshot.RuntimeVersion, contextmemory.Version)
	}
	if len(result.Snapshot.Sources) != 1 {
		t.Fatalf("source count = %d, want 1", len(result.Snapshot.Sources))
	}
	if len(result.ContextPack) == 0 {
		t.Fatal("expected frozen Context pack")
	}
}

func TestBuildPackRejectsCrossProjectRequest(t *testing.T) {
	_, err := BuildPack(
		context.Background(),
		"project-a",
		nil,
		contextmemory.PackRequest{ProjectID: "project-b"},
	)
	if err == nil {
		t.Fatal("expected project mismatch error")
	}
}
