// Package systemone calls one allowlisted System One HTTP endpoint.
package systemone

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fastygo/lex/internal/canonical"
)

const maxResponseBytes = 2 << 20

// maxProviderQuestions is the published question limit checked before a call.
const maxProviderQuestions = 64

// Call is one bounded provider request. It never retries or changes model.
type Call struct {
	APIKey           string
	Endpoint         string
	OfficialEndpoint string
	Model            string
	ExpectedModel    string
	AllowLoopback    bool
	HTTP             *http.Client
}

// Decision preserves the raw provider payload. Metadata fields stay empty when
// the provider omits them.
type Decision struct {
	ResolvedModel    string
	Answers          json.RawMessage
	RequestID        string
	EvaluationTimeMS *float64
	Usage            json.RawMessage
}

// Evaluate posts one frozen state and question map.
func Evaluate(ctx context.Context, call Call, state any, questions map[string]any) (Decision, error) {
	if strings.TrimSpace(call.APIKey) == "" {
		return Decision{}, fmt.Errorf("decision credential is not configured")
	}
	if !endpointAllowed(call) {
		return Decision{}, fmt.Errorf("decision endpoint is not allowlisted")
	}
	if !questionsSupported(questions) {
		return Decision{}, fmt.Errorf("decision questions are outside the declared capabilities")
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
	limited := *client
	limited.CheckRedirect = func(*http.Request, []*http.Request) error {
		return fmt.Errorf("decision endpoint redirect is not allowed")
	}
	response, err := limited.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Decision{}, err
		}
		return Decision{}, CallError{Retryable: true}
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Decision{}, err
		}
		return Decision{}, CallError{Retryable: true}
	}
	if len(body) > maxResponseBytes {
		return Decision{}, fmt.Errorf("decision response exceeds limit")
	}
	if response.StatusCode != http.StatusOK {
		return Decision{}, CallError{Retryable: retryableProviderStatus(response.StatusCode)}
	}
	if _, err := canonical.DecodeJSON(body); err != nil {
		return Decision{}, fmt.Errorf("decode decision response: %w", err)
	}
	var decoded struct {
		Model            string          `json:"model"`
		Answers          json.RawMessage `json:"answers"`
		Usage            json.RawMessage `json:"usage"`
		RequestID        string          `json:"request_id"`
		EvaluationTimeMS *float64        `json:"evaluation_time_ms"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Decision{}, fmt.Errorf("decode decision response: %w", err)
	}
	if decoded.Model == "" || strings.Contains(decoded.Model, "latest") || len(decoded.Answers) == 0 {
		return Decision{}, fmt.Errorf("decision response lacks a resolved model or answers")
	}
	if call.ExpectedModel != "" && decoded.Model != call.ExpectedModel {
		return Decision{}, fmt.Errorf("decision response model does not match the pinned request model")
	}
	usage, err := usableMetadata(call.APIKey, decoded.RequestID, decoded.EvaluationTimeMS, decoded.Usage)
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		ResolvedModel:    decoded.Model,
		Answers:          decoded.Answers,
		RequestID:        decoded.RequestID,
		EvaluationTimeMS: decoded.EvaluationTimeMS,
		Usage:            usage,
	}, nil
}

func retryableProviderStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout, 529:
		return true
	default:
		return false
	}
}

// CallError is a provider failure that carries no response body or status text.
type CallError struct {
	Retryable bool
}

func (err CallError) Error() string {
	if err.Retryable {
		return "decision provider is temporarily unavailable"
	}
	return "decision provider failed"
}

// ProviderRetryable reports whether the caller may try again later. This adapter does not retry.
func (err CallError) ProviderRetryable() bool { return err.Retryable }

func questionsSupported(questions map[string]any) bool {
	if len(questions) == 0 || len(questions) > maxProviderQuestions {
		return false
	}
	for _, raw := range questions {
		question, ok := raw.(map[string]any)
		if !ok {
			return false
		}
		switch question["type"] {
		case "noul", "choice", "score":
		default:
			return false
		}
	}
	return true
}

func usableMetadata(secret, requestID string, elapsed *float64, usage json.RawMessage) (json.RawMessage, error) {
	if requestID != "" && (len(requestID) > 256 || strings.Contains(requestID, secret) || strings.ContainsAny(requestID, "\r\n")) {
		return nil, fmt.Errorf("decision response metadata is not usable")
	}
	if elapsed != nil && (math.IsNaN(*elapsed) || math.IsInf(*elapsed, 0) || *elapsed < 0) {
		return nil, fmt.Errorf("decision response metadata is not usable")
	}
	if len(usage) == 0 || string(usage) == "null" {
		return nil, nil
	}
	if strings.Contains(string(usage), secret) {
		return nil, fmt.Errorf("decision response metadata is not usable")
	}
	var fields map[string]any
	if err := json.Unmarshal(usage, &fields); err != nil {
		return nil, fmt.Errorf("decision response metadata is not usable")
	}
	for _, value := range fields {
		number, ok := value.(float64)
		if !ok || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 {
			return nil, fmt.Errorf("decision response metadata is not usable")
		}
	}
	return usage, nil
}

func endpointAllowed(call Call) bool {
	if call.Endpoint == call.OfficialEndpoint {
		return true
	}
	if !call.AllowLoopback {
		return false
	}
	parsed, err := url.Parse(call.Endpoint)
	if err != nil || parsed.User != nil || parsed.Scheme != "http" || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "" {
		return false
	}
	return true
}
