package typesafe

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEvaluatePreservesRawAnswers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatal("missing authorization")
		}
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), `"model":"jev-1.13.0"`) {
			t.Fatalf("request = %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9}},"usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer server.Close()

	client := newForTest("test-key", server.URL)
	client.HTTP = server.Client()
	decision, err := client.Evaluate(
		context.Background(), "frozen evidence", map[string]any{"support": map[string]any{"type": "noul", "instructions": "Is it supported?"}},
	)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.ResolvedModel != Model || !strings.Contains(string(decision.Answers), `"noul":0.9`) {
		t.Fatalf("decision = %#v", decision)
	}
}

func TestEvaluateRejectsDuplicateProviderKeys(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.1},"support":{"type":"noul","noul":0.9}}}`)
	}))
	defer server.Close()
	client := newForTest("test-key", server.URL)
	client.HTTP = server.Client()
	if _, err := client.Evaluate(context.Background(), "state", map[string]any{"support": map[string]any{"type": "noul"}}); err == nil {
		t.Fatal("accepted duplicate answer keys")
	}
}

func TestEvaluateRejectsOversizedProviderResponse(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		padding := strings.Repeat("x", 2<<20)
		_, _ = io.WriteString(w, `{"model":"jev-1.13.0","answers":{"support":{"type":"noul","noul":0.9,"note":"`+padding+`"}}}`)
	}))
	defer server.Close()
	client := newForTest("test-key", server.URL)
	client.HTTP = server.Client()
	_, err := client.Evaluate(context.Background(), "state", map[string]any{"support": map[string]any{"type": "noul"}})
	if err == nil || calls != 1 || strings.Contains(err.Error(), "xxxx") {
		t.Fatalf("calls = %d err = %v", calls, err)
	}
}

func TestEvaluateRejectsAliasAndRemoteEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"model":"jev-latest","answers":{}}`)
	}))
	defer server.Close()
	client := newForTest("test-key", server.URL)
	client.HTTP = server.Client()
	if _, err := client.Evaluate(context.Background(), "state", map[string]any{}); err == nil {
		t.Fatal("accepted alias response")
	}
	if _, err := newForTest("test-key", "https://example.invalid/systemone").Evaluate(context.Background(), "state", nil); err == nil {
		t.Fatal("accepted non-allowlisted endpoint")
	}
}
