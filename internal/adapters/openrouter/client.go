// Package openrouter adapts the hosted System One HTTP API to LeX.
package openrouter

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fastygo/lex/internal/adapters/systemone"
	"github.com/fastygo/lex/internal/profile"
)

const (
	// Endpoint is the only production hosted endpoint.
	Endpoint = "https://openrouter.ai/api/v1/systemone"
	// Model is the pinned request selector. The response must be a resolved refinement.
	Model          = "typesafe/jev-1.13"
	AdapterID      = "hosted-systemone"
	AdapterVersion = profile.AdapterVersion
)

// Client calls one allowlisted hosted endpoint. The zero value is not usable.
type Client struct {
	APIKey        string
	resolvedModel string
	endpoint      string
	allowLoopback bool
	HTTP          *http.Client
}

// Decision is the raw provider response retained for verification.
type Decision = systemone.Decision

// New returns the production hosted adapter.
func New(apiKey, resolvedModel string) Client {
	return Client{APIKey: apiKey, resolvedModel: resolvedModel, endpoint: Endpoint, HTTP: &http.Client{Timeout: 12 * time.Second}}
}

func newForTest(apiKey, endpoint string) Client {
	return Client{APIKey: apiKey, resolvedModel: Model + "-20260901", endpoint: endpoint, allowLoopback: true}
}

// Evaluate sends one frozen state and question map. It does not retry or fall back.
func (c Client) Evaluate(ctx context.Context, state any, questions map[string]any) (Decision, error) {
	if !ValidResolvedPin(c.resolvedModel) {
		return Decision{}, fmt.Errorf("hosted resolved model pin is required")
	}
	decision, err := systemone.Evaluate(ctx, systemone.Call{
		APIKey:           c.APIKey,
		Endpoint:         c.endpoint,
		OfficialEndpoint: Endpoint,
		Model:            Model,
		ExpectedModel:    c.resolvedModel,
		AllowLoopback:    c.allowLoopback,
		HTTP:             c.HTTP,
	}, state, questions)
	if err != nil {
		return Decision{}, err
	}
	if decision.ResolvedModel != c.resolvedModel {
		return Decision{}, fmt.Errorf("decision response model is not a resolved identity")
	}
	return decision, nil
}

// ValidResolvedPin validates deployment configuration, not provider immutability.
// Operators must obtain an immutable identity from the provider; the response
// must then match that exact identity. A syntactic refinement alone is no proof.
func ValidResolvedPin(model string) bool {
	if !strings.HasPrefix(model, Model+"-") && !strings.HasPrefix(model, Model+".") {
		return false
	}
	suffix := model[len(Model)+1:]
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < 0x21 || r > 0x7e || r == '/' {
			return false
		}
	}
	lower := strings.ToLower(suffix)
	for _, alias := range []string{"latest", "stable", "preview", "auto"} {
		if strings.Contains(lower, alias) {
			return false
		}
	}
	return true
}
