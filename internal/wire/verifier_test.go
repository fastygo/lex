package wire

import (
	"context"
	"testing"

	"github.com/fastygo/lex/internal/adapters/openrouter"
	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/profile"
)

// The hosted identity is synthetic test configuration, not a provider release claim.
func fixtureVerifier() Verifier {
	return NewVerifier(map[string]AdapterPin{
		typesafe.AdapterID:   {Version: typesafe.AdapterVersion, Model: typesafe.Model},
		openrouter.AdapterID: {Version: openrouter.AdapterVersion, Model: openrouter.Model + "-20260901"},
	})
}

func Replay(raw []byte) (Report, error) { return fixtureVerifier().Replay(raw) }
func ReplayContext(ctx context.Context, raw []byte) (Report, error) {
	return fixtureVerifier().ReplayContext(ctx, raw)
}

func TestVerifierRequiresExactDeploymentModelPin(t *testing.T) {
	raw := goldenBundle(t, goldenCase{adapter: openrouter.AdapterID, answers: goldenAnswers(.9, .9, .1, .9, "proceed")})
	pins := map[string]AdapterPin{openrouter.AdapterID: {Version: profile.AdapterVersion, Model: openrouter.Model + "-another-release"}}
	verifier := NewVerifier(pins)
	pins[openrouter.AdapterID] = AdapterPin{Version: profile.AdapterVersion, Model: openrouter.Model + "-20260901"}
	report, err := verifier.Replay(raw)
	if err != nil || !hasFinding(report, "unresolved_model") {
		t.Fatalf("unapproved model: %+v err=%v", report, err)
	}
	report, err = (Verifier{}).Replay(raw)
	if err != nil || !hasFinding(report, "unknown_adapter") {
		t.Fatalf("unconfigured adapter: %+v err=%v", report, err)
	}
}
