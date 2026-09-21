package openrouter

import (
	"context"
	"net/http"
	"testing"

	"github.com/fastygo/lex/internal/adapters/conformance"
)

func TestHostedAdapterConformanceFixture(t *testing.T) {
	conformance.Run(t, Model, func(endpoint string, client *http.Client) conformance.Result {
		adapter := newForTest("fixture-key", endpoint)
		adapter.HTTP = client
		decision, err := adapter.Evaluate(context.Background(), map[string]any{"claim": "fixture"}, map[string]any{
			"support": map[string]any{"type": "noul", "instructions": "Does admissible frozen evidence directly support the claim?"},
		})
		return conformance.Result{ResolvedModel: decision.ResolvedModel, Answers: decision.Answers, Err: err}
	})
}
