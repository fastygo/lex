package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestLatencySampleAgreesOnVerdicts(t *testing.T) {
	handler := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(passingAnswers)})
	bundle := sealedBundle(t, handler)
	replaySamples := make([]time.Duration, 10000)
	for i := range replaySamples {
		start := time.Now()
		recorder := postReplay(handler, bundle)
		replaySamples[i] = time.Since(start)
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"verdict":"validated"`) {
			t.Fatalf("replay %d status = %d body = %s", i, recorder.Code, recorder.Body)
		}
	}
	evaluationSamples := make([]time.Duration, 1000)
	for i := range evaluationSamples {
		start := time.Now()
		request := httptest.NewRequest(http.MethodPost, "/v1/evaluations", strings.NewReader(evaluationBody))
		request.Header.Set("Authorization", "Bearer test-token")
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		evaluationSamples[i] = time.Since(start)
		if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"verdict":"validated"`) {
			t.Fatalf("evaluation %d status = %d body = %s", i, recorder.Code, recorder.Body)
		}
	}
	t.Logf("warm replay n=10000 p50=%s p95=%s p99=%s max=%s", percentile(replaySamples, 50), percentile(replaySamples, 95), percentile(replaySamples, 99), percentile(replaySamples, 100))
	t.Logf("stub evaluation n=1000 p50=%s p95=%s p99=%s max=%s", percentile(evaluationSamples, 50), percentile(evaluationSamples, 95), percentile(evaluationSamples, 99), percentile(evaluationSamples, 100))
}

func sealedBundle(t *testing.T, handler http.Handler) []byte {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/evaluations", strings.NewReader(evaluationBody))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("seal status = %d body = %s", recorder.Code, recorder.Body)
	}
	var response struct {
		ReplayBundle json.RawMessage `json:"replay_bundle"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	return response.ReplayBundle
}

func postReplay(handler http.Handler, bundle []byte) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/v1/replays", strings.NewReader(string(bundle)))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func percentile(samples []time.Duration, pct int) time.Duration {
	ordered := append([]time.Duration(nil), samples...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := len(ordered) * pct / 100
	if index >= len(ordered) {
		index = len(ordered) - 1
	}
	if index < 0 {
		index = 0
	}
	return ordered[index]
}
