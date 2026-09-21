package systemone

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
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
	if err == nil || followed.Load() != 0 {
		t.Fatalf("followed = %d err = %v", followed.Load(), err)
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
