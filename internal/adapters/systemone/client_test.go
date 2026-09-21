package systemone

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestEndpointAllowlistRejectsLoopbackDisguises(t *testing.T) {
	call := Call{
		APIKey:           "test-key",
		OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model:            "jev-1.13.0",
		AllowLoopback:    true,
	}
	for _, endpoint := range []string{
		"http://127.0.0.1:80@evil.example/systemone",
		"http://127.0.0.1.evil.example:80/systemone",
		"http://2130706433:80/systemone",
		"https://127.0.0.1:80/systemone",
		"http://127.0.0.1/systemone",
	} {
		call.Endpoint = endpoint
		if _, err := Evaluate(context.Background(), call, map[string]any{}, map[string]any{}); err == nil {
			t.Fatalf("accepted %s", endpoint)
		}
	}
	if !endpointAllowed(Call{Endpoint: "https://api.typesafe.ai/v1/systemone", OfficialEndpoint: "https://api.typesafe.ai/v1/systemone"}) {
		t.Fatal("rejected the official endpoint")
	}
}

func TestEvaluateDoesNotFollowRedirect(t *testing.T) {
	var followed atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		followed.Add(1)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, target.URL, http.StatusFound)
	}))
	defer source.Close()

	_, err := Evaluate(context.Background(), Call{
		APIKey:           "test-key",
		Endpoint:         source.URL,
		OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model:            "jev-1.13.0",
		AllowLoopback:    true,
		HTTP:             source.Client(),
	}, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	var classified CallError
	if !errors.As(err, &classified) || classified.Retryable || followed.Load() != 0 || strings.Contains(err.Error(), target.URL) {
		t.Fatalf("followed = %d err = %v", followed.Load(), err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Evaluate(canceled, Call{
		APIKey:           "test-key",
		Endpoint:         source.URL,
		OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model:            "jev-1.13.0",
		AllowLoopback:    true,
		HTTP:             source.Client(),
	}, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	if !errors.Is(err, context.Canceled) || strings.Contains(err.Error(), source.URL) {
		t.Fatalf("err = %v", err)
	}
}

func TestEvaluateClassifiesClientTimeoutAsProviderFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()
	client := server.Client()
	client.Timeout = 20 * time.Millisecond
	_, err := Evaluate(context.Background(), Call{
		APIKey: "test-key", Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model: "jev-1.13.0", AllowLoopback: true, HTTP: client,
	}, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	var classified CallError
	if !errors.As(err, &classified) || !classified.Retryable || errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), server.URL) {
		t.Fatalf("err = %v", err)
	}
}

func TestEvaluateDeclaresQuestionCapabilitiesBeforeDial(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls.Add(1)
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), `"type":"score"`) {
			t.Fatalf("request = %s", body)
		}
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"rubric":{"type":"score","score":1}}}`)
	}))
	defer server.Close()
	call := Call{
		APIKey: "test-key", Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model: "jev-1.13.0", AllowLoopback: true, HTTP: server.Client(),
	}
	if _, err := Evaluate(context.Background(), call, map[string]any{}, map[string]any{"rubric": map[string]any{"type": "score"}}); err != nil || calls.Load() != 1 {
		t.Fatalf("calls = %d err = %v", calls.Load(), err)
	}
	if _, err := Evaluate(context.Background(), call, map[string]any{}, map[string]any{"note": map[string]any{"type": "text"}}); err == nil || calls.Load() != 1 {
		t.Fatalf("calls = %d err = %v", calls.Load(), err)
	}
	questions := map[string]any{}
	for i := 0; i < maxProviderQuestions+1; i++ {
		questions[string(rune('a'+i))] = map[string]any{"type": "noul"}
	}
	if _, err := Evaluate(context.Background(), call, map[string]any{}, questions); err == nil || calls.Load() != 1 {
		t.Fatalf("calls = %d err = %v", calls.Load(), err)
	}
}

func TestEvaluateKeepsProviderMetadataOnlyWhenPresent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9}},"request_id":"req-1","evaluation_time_ms":12.5,"usage":{"input_tokens":3,"output_tokens":1}}`)
	}))
	defer server.Close()
	call := Call{
		APIKey: "test-key", Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model: "jev-1.13.0", AllowLoopback: true, HTTP: server.Client(),
	}
	decision, err := Evaluate(context.Background(), call, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	if err != nil {
		t.Fatal(err)
	}
	if decision.RequestID != "req-1" || decision.EvaluationTimeMS == nil || *decision.EvaluationTimeMS != 12.5 || !strings.Contains(string(decision.Usage), `"input_tokens":3`) {
		t.Fatalf("decision = %#v", decision)
	}
	secretServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9}},"request_id":"test-key"}`)
	}))
	defer secretServer.Close()
	call.Endpoint = secretServer.URL
	call.HTTP = secretServer.Client()
	_, err = Evaluate(context.Background(), call, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	if err == nil || strings.Contains(err.Error(), "test-key") {
		t.Fatalf("err = %v", err)
	}
	_ = json.Valid([]byte(decision.Usage))
}

func TestEvaluateDropsResponsesThatEchoTheCredential(t *testing.T) {
	const secret = "provider-credential-must-not-leak"
	bodies := []string{
		`{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9,"note":"` + secret + `"}}}`,
		`{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9,"note":"\u0070rovider-credential-must-not-leak"}}}`,
		`{"model":"` + secret + `","answers":{"support":{"type":"noul","noul":0.9}}}`,
	}
	for _, body := range bodies {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, body)
		}))
		decision, err := Evaluate(context.Background(), Call{
			APIKey: secret, Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
			Model: "jev-1.13.0", ExpectedModel: "jev-1.13.0", AllowLoopback: true, HTTP: server.Client(),
		}, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
		server.Close()
		if err == nil || strings.Contains(err.Error(), secret) || decision.ResolvedModel != "" || len(decision.Answers) != 0 {
			t.Fatalf("body %s err = %v decision = %#v", body, err, decision)
		}
	}
}

func TestEvaluateClassifiesRetryableProviderStatus(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "secret-provider-body", http.StatusTooManyRequests)
	}))
	defer server.Close()
	call := Call{
		APIKey: "test-key", Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model: "jev-1.13.0", AllowLoopback: true, HTTP: server.Client(),
	}
	_, err := Evaluate(context.Background(), call, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	var classified CallError
	if !errors.As(err, &classified) || !classified.Retryable || calls.Load() != 1 || strings.Contains(err.Error(), "secret-provider-body") || strings.Contains(err.Error(), "429") {
		t.Fatalf("calls = %d err = %v", calls.Load(), err)
	}
	rejected := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "secret-provider-body", http.StatusBadRequest)
	}))
	defer rejected.Close()
	call.Endpoint = rejected.URL
	call.HTTP = rejected.Client()
	_, err = Evaluate(context.Background(), call, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	if !errors.As(err, &classified) || classified.Retryable || strings.Contains(err.Error(), "secret-provider-body") {
		t.Fatalf("err = %v", err)
	}
}

func TestEvaluateStopsReadingAtTheResponseBudget(t *testing.T) {
	const marker = "BUDGET-MARKER-MUST-NOT-LEAK"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9}},"note":"`+marker+strings.Repeat("x", 64)+`"}`)
	}))
	defer server.Close()
	_, err := Evaluate(WithMaxResponseBytes(context.Background(), 32), Call{
		APIKey: "test-key", Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model: "jev-1.13.0", AllowLoopback: true, HTTP: server.Client(),
	}, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	var budget BudgetError
	if !errors.As(err, &budget) || strings.Contains(err.Error(), marker) {
		t.Fatalf("err = %v", err)
	}
}

func TestEvaluateRejectsDeepProviderJSON(t *testing.T) {
	const marker = "DEEP-MARKER-MUST-NOT-LEAK"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":`+strings.Repeat(`{"k":`, 40)+`{"type":"noul","noul":0.9,"note":"`+marker+`"}`+strings.Repeat(`}`, 40)+`}`)
	}))
	defer server.Close()
	_, err := Evaluate(context.Background(), Call{
		APIKey: "test-key", Endpoint: server.URL, OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model: "jev-1.13.0", AllowLoopback: true, HTTP: server.Client(),
	}, map[string]any{}, map[string]any{"support": map[string]any{"type": "noul"}})
	if err == nil || strings.Contains(err.Error(), marker) {
		t.Fatalf("err = %v", err)
	}
}
