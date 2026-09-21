// Package conformance exercises both decision adapters against one fixture.
package conformance

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Result is the raw provider output an adapter must preserve.
type Result struct {
	ResolvedModel string
	Answers       []byte
	Err           error
}

// Evaluate is one adapter call against a test endpoint.
type Evaluate func(endpoint string, client *http.Client) Result

// Run checks model pinning, raw-answer preservation, and a single failed attempt.
// requestModel is what the adapter sends. responseModel is the resolved identity
// the fixture returns; for the direct adapter they are the same value.
func Run(t *testing.T, requestModel, responseModel string, evaluate Evaluate) {
	t.Helper()
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		calls++
		if request.Header.Get("Authorization") != "Bearer fixture-key" {
			t.Errorf("authorization = %q", request.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(request.Body)
		if !strings.Contains(string(body), `"model":"`+requestModel+`"`) {
			t.Errorf("request model missing %s in %s", requestModel, body)
		}
		if calls == 1 {
			_, _ = io.WriteString(w, `{"model":"`+responseModel+`","answers":{"support":{"type":"noul","noul":0.42}}}`)
			return
		}
		http.Error(w, "unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	ok := evaluate(server.URL, server.Client())
	if ok.Err != nil {
		t.Fatalf("successful fixture error = %v", ok.Err)
	}
	if ok.ResolvedModel != responseModel || !strings.Contains(string(ok.Answers), `"noul":0.42`) {
		t.Fatalf("result = %+v", ok)
	}
	failed := evaluate(server.URL, server.Client())
	if failed.Err == nil {
		t.Fatal("provider failure returned a decision")
	}
	if calls != 2 {
		t.Fatalf("provider calls = %d, want 2", calls)
	}
}
