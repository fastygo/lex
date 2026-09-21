package openrouter

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEvaluateRecordsResolvedHostedModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), `"model":"typesafe/jev-1.13"`) {
			t.Fatalf("request = %s", body)
		}
		_, _ = io.WriteString(w, `{"model":"typesafe/jev-1.13-20260901","answers":{"support":{"type":"noul","noul":0.9}}}`)
	}))
	defer server.Close()

	client := newForTest("test-key", server.URL)
	client.HTTP = server.Client()
	decision, err := client.Evaluate(context.Background(), "frozen evidence", map[string]any{"support": map[string]any{"type": "noul"}})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if decision.ResolvedModel != "typesafe/jev-1.13-20260901" {
		t.Fatalf("resolved model = %q", decision.ResolvedModel)
	}
}

func TestEvaluateRejectsUnresolvedHostedModel(t *testing.T) {
	for _, model := range []string{Model, "typesafe/jev-1.14-20260901", "typesafe/jev-latest"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"model":"`+model+`","answers":{"support":{"type":"noul","noul":0.9}}}`)
		}))
		client := newForTest("test-key", server.URL)
		client.HTTP = server.Client()
		_, err := client.Evaluate(context.Background(), "frozen evidence", map[string]any{"support": map[string]any{"type": "noul"}})
		server.Close()
		if err == nil || strings.Contains(err.Error(), model) {
			t.Fatalf("model %s error = %v", model, err)
		}
	}
}

func TestEvaluateRejectsForeignEndpoint(t *testing.T) {
	client := newForTest("test-key", "https://example.invalid/systemone")
	if _, err := client.Evaluate(context.Background(), "state", nil); err == nil {
		t.Fatal("accepted non-allowlisted endpoint")
	}
}
