// Package typesafe adapts the direct System One HTTP API to LeX.
package typesafe

import (
	"context"
	"net/http"
	"time"

	"github.com/fastygo/lex/internal/adapters/systemone"
	"github.com/fastygo/lex/internal/profile"
)

const (
	// Endpoint is the only production direct endpoint.
	Endpoint = "https://api.typesafe.ai/v1/systemone"
	// Model is the pinned request model. Aliases are rejected.
	Model          = profile.DirectModel
	AdapterID      = "direct-systemone"
	AdapterVersion = profile.AdapterVersion
)

// Client calls one allowlisted direct endpoint. The zero value is not usable.
type Client struct {
	APIKey        string
	endpoint      string
	allowLoopback bool
	HTTP          *http.Client
}

// Decision is the raw provider response retained for verification.
type Decision = systemone.Decision

// New returns the production direct adapter.
func New(apiKey string) Client {
	return Client{APIKey: apiKey, endpoint: Endpoint, HTTP: &http.Client{Timeout: 12 * time.Second}}
}

func newForTest(apiKey, endpoint string) Client {
	return Client{APIKey: apiKey, endpoint: endpoint, allowLoopback: true}
}

// Evaluate sends one frozen state and question map. It does not retry or fall back.
func (c Client) Evaluate(ctx context.Context, state any, questions map[string]any) (Decision, error) {
	return systemone.Evaluate(ctx, systemone.Call{
		APIKey:           c.APIKey,
		Endpoint:         c.endpoint,
		OfficialEndpoint: Endpoint,
		Model:            Model,
		ExpectedModel:    Model,
		AllowLoopback:    c.allowLoopback,
		HTTP:             c.HTTP,
	}, state, questions)
}
