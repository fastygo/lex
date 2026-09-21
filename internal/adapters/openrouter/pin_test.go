package openrouter

import (
	"context"
	"testing"
)

func TestHostedPinRejectsAliasesAndRequiresConfiguration(t *testing.T) {
	for _, suffix := range []string{"", "-stable", "-preview", "-LATEST", "-latest", "-\n20260901", "-2026/09", "-\u00a020260901"} {
		if ValidResolvedPin(Model + suffix) {
			t.Fatalf("accepted alias or invalid identity %q", suffix)
		}
	}
	if _, err := New("credential", "").Evaluate(context.Background(), nil, nil); err == nil {
		t.Fatal("unconfigured model pin must fail before dialing")
	}
}
