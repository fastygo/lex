package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/wire"
)

type exampleDecider struct {
	answers []byte
	calls   int
	state   any
}

func (exampleDecider) AdapterID() string      { return "direct-systemone" }
func (exampleDecider) AdapterVersion() string { return "0.1.0" }

func (decider *exampleDecider) Evaluate(_ context.Context, state any, _ map[string]any) (Decision, error) {
	decider.calls++
	decider.state = state
	return Decision{ResolvedModel: typesafe.Model, Answers: decider.answers}, nil
}

func TestExampleRequestsRunThroughTheProtocol(t *testing.T) {
	dir := filepath.Join("..", "..", ".project", ".jev", "test-vercel", "requests")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 18 {
		t.Fatalf("example requests = %d, want 18", len(entries))
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			t.Fatalf("unexpected entry %s", entry.Name())
		}
		t.Run(entry.Name(), func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if err := wire.ValidateEvaluationRequest(raw); err != nil {
				t.Fatal(err)
			}
			decider := &exampleDecider{answers: []byte(passingAnswers)}
			handler, err := NewHandler(Config{
				BearerTokens:   map[string][]string{"test-token": {"example-project"}},
				RequestTimeout: defaultRequestTimeout,
				MaxBodyBytes:   defaultMaxBodyBytes,
				MaxInFlight:    1,
				Decider:        decider,
			})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/v1/evaluations", strings.NewReader(string(raw)))
			request.Header.Set("Authorization", "Bearer test-token")
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("status = %d cache = %q body = %s", recorder.Code, recorder.Header().Get("Cache-Control"), recorder.Body)
			}
			var response struct {
				Verdict      string          `json:"verdict"`
				ReplayBundle json.RawMessage `json:"replay_bundle"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Verdict != "validated" || len(response.ReplayBundle) == 0 || decider.calls != 1 {
				t.Fatalf("verdict = %s calls = %d bundle = %d", response.Verdict, decider.calls, len(response.ReplayBundle))
			}
			state, err := json.Marshal(decider.state)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(state), `"client_ref"`) || strings.Contains(string(response.ReplayBundle), `"client_ref"`) {
				t.Fatal("caller metadata entered the decision state or the replay bundle")
			}
			replay := httptest.NewRequest(http.MethodPost, "/v1/replays", strings.NewReader(string(response.ReplayBundle)))
			replay.Header.Set("Authorization", "Bearer test-token")
			replay.Header.Set("Content-Type", "application/json")
			replayed := httptest.NewRecorder()
			handler.ServeHTTP(replayed, replay)
			if replayed.Code != http.StatusOK || !strings.Contains(replayed.Body.String(), `"verdict":"validated"`) || !strings.Contains(replayed.Body.String(), `"replay_status":"verdict_reproduced"`) || decider.calls != 1 {
				t.Fatalf("replay status = %d calls = %d body = %s", replayed.Code, decider.calls, replayed.Body)
			}
		})
	}
}

func TestResearchMapsAreNotEvaluationRequests(t *testing.T) {
	files := []string{
		"playground/questions-jev.json",
		"playground/response-jev.json",
		"context/questions-context.json",
		"context/response-context.json",
		"llm/questions-llm.json",
		"llm/response-llm.json",
		"chaos/questions-chaos.json",
		"chaos/response-chaos.json",
	}
	decider := &scriptedDecider{answers: []byte(passingAnswers)}
	handler := mustHandlerWithDecider(t, decider)
	root := filepath.Join("..", "..", ".project", ".jev", "examples")
	for _, relative := range files {
		t.Run(relative, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join(root, relative))
			if err != nil {
				t.Fatal(err)
			}
			if err := wire.ValidateEvaluationRequest(raw); err == nil {
				t.Fatal("research map passed the evaluation schema")
			}
			before := decider.calls
			request := httptest.NewRequest(http.MethodPost, "/v1/evaluations", strings.NewReader(string(raw)))
			request.Header.Set("Authorization", "Bearer test-token")
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"reason":"invalid_json"`) || decider.calls != before {
				t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
			}
		})
	}
}
