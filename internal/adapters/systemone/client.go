// Package systemone calls one allowlisted System One HTTP endpoint.
package systemone

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fastygo/lex/internal/canonical"
)

const maxResponseBytes = 2 << 20

// Call is one bounded provider request. It never retries or changes model.
type Call struct {
	APIKey           string
	Endpoint         string
	OfficialEndpoint string
	Model            string
	AllowLoopback    bool
	HTTP             *http.Client
}

// Decision preserves the raw provider payload.
type Decision struct {
	ResolvedModel string
	Answers       json.RawMessage
	Usage         json.RawMessage
}

// Evaluate posts one frozen state and question map.
func Evaluate(ctx context.Context, call Call, state any, questions map[string]any) (Decision, error) {
	if strings.TrimSpace(call.APIKey) == "" {
		return Decision{}, fmt.Errorf("decision credential is not configured")
	}
	if call.Endpoint != call.OfficialEndpoint && !(call.AllowLoopback && strings.HasPrefix(call.Endpoint, "http://127.0.0.1:")) {
		return Decision{}, fmt.Errorf("decision endpoint is not allowlisted")
	}
	payload, err := json.Marshal(map[string]any{"model": call.Model, "state": state, "questions": questions})
	if err != nil {
		return Decision{}, fmt.Errorf("encode decision request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, call.Endpoint, bytes.NewReader(payload))
	if err != nil {
		return Decision{}, err
	}
	request.Header.Set("Authorization", "Bearer "+call.APIKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	client := call.HTTP
	if client == nil {
		client = &http.Client{Timeout: 12 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return Decision{}, fmt.Errorf("decision transport: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		return Decision{}, fmt.Errorf("read decision response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return Decision{}, fmt.Errorf("decision response exceeds limit")
	}
	if response.StatusCode != http.StatusOK {
		return Decision{}, fmt.Errorf("decision provider status %d", response.StatusCode)
	}
	if _, err := canonical.DecodeJSON(body); err != nil {
		return Decision{}, fmt.Errorf("decode decision response: %w", err)
	}
	var decoded struct {
		Model   string          `json:"model"`
		Answers json.RawMessage `json:"answers"`
		Usage   json.RawMessage `json:"usage"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Decision{}, fmt.Errorf("decode decision response: %w", err)
	}
	if decoded.Model == "" || strings.Contains(decoded.Model, "latest") || len(decoded.Answers) == 0 {
		return Decision{}, fmt.Errorf("decision response lacks a resolved model or answers")
	}
	return Decision{ResolvedModel: decoded.Model, Answers: decoded.Answers, Usage: decoded.Usage}, nil
}
