package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const validAnswers = `{"intent":{"type":"choice","choice":"portfolio","probabilities":{"products":0.1,"portfolio":0.8,"cases":0.1}},"has_transactions":{"type":"noul","noul":0.9},"fit":{"type":"score","score":1.5}}`

const decisionBody = `{
  "project_id":"project-test",
  "decision":{"id":"agent-step-1","version":"1"},
  "state":{"request":"Create a storefront","requirements":{"transactions":true}},
  "question_set":{
    "id":"agent.intent","version":"1",
    "questions":{
      "intent":{"type":"choice","instructions":"Which storefront type is best supported?","options":{"products":"Product catalogue","portfolio":"Portfolio of work","cases":"Client case studies"}},
      "has_transactions":{"type":"noul","instructions":"Does the supplied state require transactions?"},
      "fit":{"type":"score","instructions":"How strongly does the stack fit?","levels":["weak","adequate","strong"]}
    }
  },
  "metadata":{"client_ref":"agent-step-1"}
}`

func TestCapabilitiesRequireBearerToken(t *testing.T) {
	recorder := serve(mustHandler(t), http.MethodGet, "/v1/capabilities", "", "")
	if recorder.Code != http.StatusUnauthorized || recorder.Header().Get("Content-Type") != problemMediaType {
		t.Fatalf("status = %d type = %q", recorder.Code, recorder.Header().Get("Content-Type"))
	}
}

func TestCapabilitiesDiscloseTheDecisionContract(t *testing.T) {
	recorder := serve(mustHandler(t), http.MethodGet, "/v1/capabilities", "test-token", "")
	if recorder.Code != http.StatusOK || recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d cache = %q", recorder.Code, recorder.Header().Get("Cache-Control"))
	}
	body := recorder.Body.String()
	for _, want := range []string{
		`"protocol_status":"canary"`, `"protocol_version":"0.2"`, `"decision":false`, `"replay":true`, `"execution":false`,
		`"primitives":["noul","choice","score"]`, `"adapter_contract":"0.1.0"`, `"semantic_verdict":false`,
		`"binding":"optional"`, `"runtime":"memory-exact-v1"`, `"server_history":false`, `"idempotency":"none"`, `"process_admission":4`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body lacks %s: %s", want, body)
		}
	}
	for _, legacy := range []string{"evaluation", `"policy"`, "focus", `"verdict"`} {
		if strings.Contains(body, legacy) {
			t.Fatalf("capabilities disclose removed %q: %s", legacy, body)
		}
	}
}

func TestHealthDoesNotRequireBearerToken(t *testing.T) {
	if recorder := serve(mustHandler(t), http.MethodGet, "/healthz", "", ""); recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestUnauthorizedBearerTokenIsForbidden(t *testing.T) {
	const token = "test-token-must-not-appear"
	recorder := serve(mustHandler(t), http.MethodGet, "/v1/capabilities", token, "")
	if recorder.Code != http.StatusForbidden || strings.Contains(recorder.Body.String(), token) {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}

func TestProblemReasonCatalogMatchesOpenAPI(t *testing.T) {
	document := openAPIDocument(t)
	schemas := document["components"].(map[string]any)["schemas"].(map[string]any)
	published := schemas["Problem"].(map[string]any)["properties"].(map[string]any)["reason"].(map[string]any)["enum"].([]any)
	if len(published) != len(problemReasons()) {
		t.Fatalf("schema lists %d reasons, code lists %d", len(published), len(problemReasons()))
	}
	seen := map[string]bool{}
	for _, item := range published {
		seen[item.(string)] = true
	}
	for _, code := range problemReasons() {
		if !seen[code] {
			t.Fatalf("schema omits %s", code)
		}
	}
}

func TestLiveResponsesMatchOpenAPI(t *testing.T) {
	handler := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(validAnswers)})
	for _, h := range []http.Handler{handler, mustHandler(t)} {
		capabilities := serve(h, http.MethodGet, "/v1/capabilities", "test-token", "")
		validateSchema(t, "CapabilitiesResponse", capabilities)
	}

	decided := postJSON(t, handler, "/v1/decisions", decisionBody)
	if decided.Code != http.StatusOK {
		t.Fatalf("decision status = %d body = %s", decided.Code, decided.Body)
	}
	validateSchema(t, "DecisionResponse", decided)
	assertNoTimestamps(t, strictJSON(t, decided.Body.Bytes()))

	replayed := postJSON(t, handler, "/v1/replays", string(replayBundle(t, decided)))
	if replayed.Code != http.StatusOK {
		t.Fatalf("replay status = %d body = %s", replayed.Code, replayed.Body)
	}
	validateSchema(t, "ReplayResponse", replayed)

	invalid := postJSON(t, mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(`{"intent":{"type":"noul","noul":0.9}}`)}), "/v1/decisions", decisionBody)
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("invalid status = %d body = %s", invalid.Code, invalid.Body)
	}
	validateSchema(t, "Problem", invalid)
	validateSchema(t, "Problem", postJSON(t, handler, "/v1/replays", `{}`))
}

func TestMalformedAnswersRemainReplayableStructuralFailures(t *testing.T) {
	cases := map[string]string{
		"nonwinning-choice":  strings.Replace(validAnswers, `"products":0.1,"portfolio":0.8`, `"products":0.8,"portfolio":0.1`, 1),
		"case-folded-noul":   strings.ReplaceAll(validAnswers, `"noul":`, `"NOUL":`),
		"missing-answer":     strings.Replace(validAnswers, `"has_transactions":{"type":"noul","noul":0.9},`, "", 1),
		"score-out-of-range": strings.Replace(validAnswers, `"score":1.5`, `"score":2.5`, 1),
	}
	for name, answers := range cases {
		t.Run(name, func(t *testing.T) {
			handler := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(answers)})
			recorder := postJSON(t, handler, "/v1/decisions", decisionBody)
			if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"reason":"decision_error"`) || !strings.Contains(recorder.Body.String(), `"structural_status":"invalid"`) {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
			}
			bundle := replayBundle(t, recorder)
			var sealed struct {
				DecisionSet struct {
					Answers json.RawMessage `json:"answers"`
				} `json:"decision_set"`
			}
			if err := json.Unmarshal(bundle, &sealed); err != nil {
				t.Fatal(err)
			}
			want, _ := canonical.HashJSON([]byte(answers))
			got, _ := canonical.HashJSON(sealed.DecisionSet.Answers)
			if want == "" || want != got {
				t.Fatal("raw answer values were changed")
			}
			replayed := postJSON(t, handler, "/v1/replays", string(bundle))
			if replayed.Code != http.StatusOK || !strings.Contains(replayed.Body.String(), `"structural_status":"invalid"`) {
				t.Fatalf("replay status = %d body = %s", replayed.Code, replayed.Body)
			}
		})
	}
}

func TestAdapterMetadataStaysOutOfTheReplayBundle(t *testing.T) {
	elapsed := 12.5
	handler := mustHandlerWithDecider(t, &scriptedDecider{
		answers: []byte(validAnswers), requestID: "provider-request-1", evaluationTime: &elapsed, usage: json.RawMessage(`{"total_tokens":42}`),
	})
	recorder := postJSON(t, handler, "/v1/decisions", decisionBody)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"adapter_metadata":{"evaluation_time_ms":12.5,"request_id":"provider-request-1","usage":{"total_tokens":42}}`) {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	if bundle := replayBundle(t, recorder); strings.Contains(string(bundle), "provider-request-1") || strings.Contains(string(bundle), "total_tokens") {
		t.Fatalf("provider metadata entered the bundle: %s", bundle)
	}
	if strings.Contains(postJSON(t, mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(validAnswers)}), "/v1/decisions", decisionBody).Body.String(), "adapter_metadata") {
		t.Fatal("omitted provider metadata was fabricated")
	}
}

func TestProviderFailuresAreClassifiedByStage(t *testing.T) {
	const secret = "sk-test-secret-must-not-leak"
	cases := []struct {
		name    string
		decider Decider
		status  int
		reason  string
	}{
		{"adapter failure", failingDecider{err: errors.New("provider rejected credential " + secret)}, http.StatusBadGateway, reasonDecisionError},
		{"retryable provider", failingDecider{err: retryableProviderError{}}, http.StatusServiceUnavailable, reasonProviderUnavailable},
		{"provider body over budget", failingDecider{err: responseBudgetError{}}, http.StatusUnprocessableEntity, reasonResponseBudget},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := postJSON(t, mustHandlerWithDecider(t, tc.decider), "/v1/decisions", decisionBody)
			body := recorder.Body.String()
			if recorder.Code != tc.status || !strings.Contains(body, `"reason":"`+tc.reason+`"`) || !strings.Contains(body, `"name":"decide","status":"failed"`) || strings.Contains(body, secret) || strings.Contains(body, "replay_bundle") {
				t.Fatalf("status = %d body = %s", recorder.Code, body)
			}
		})
	}
	missing := postJSON(t, mustHandler(t), "/v1/decisions", decisionBody)
	if missing.Code != http.StatusServiceUnavailable || !strings.Contains(missing.Body.String(), `"reason":"decision_provider_unavailable"`) {
		t.Fatalf("missing adapter status = %d body = %s", missing.Code, missing.Body)
	}
}

func TestMalformedBodiesNeverReachTheProvider(t *testing.T) {
	cases := []struct {
		name   string
		body   string
		status int
		reason string
	}{
		{"duplicate key", strings.Replace(decisionBody, `"project_id":"project-test"`, `"project_id":"project-test","project_id":"project-test"`, 1), http.StatusBadRequest, reasonInvalidJSON},
		{"lone surrogate", strings.Replace(decisionBody, "Create a storefront", `Create \ud800`, 1), http.StatusBadRequest, reasonInvalidJSON},
		{"invalid utf-8", strings.Replace(decisionBody, "Create a storefront", "Create "+string([]byte{0xff}), 1), http.StatusBadRequest, reasonInvalidJSON},
		{"caller policy", strings.Replace(decisionBody, `"metadata"`, `"policy":{"threshold":0},"metadata"`, 1), http.StatusUnprocessableEntity, reasonQuestionError},
		{"caller approval", strings.Replace(decisionBody, `"metadata"`, `"approved":true,"metadata"`, 1), http.StatusUnprocessableEntity, reasonQuestionError},
		{"another project", strings.Replace(decisionBody, `"project_id":"project-test"`, `"project_id":"other-project"`, 1), http.StatusForbidden, reasonProjectForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			decider := &scriptedDecider{answers: []byte(validAnswers)}
			recorder := postJSON(t, mustHandlerWithDecider(t, decider), "/v1/decisions", tc.body)
			if recorder.Code != tc.status || decider.calls != 0 || !strings.Contains(recorder.Body.String(), `"reason":"`+tc.reason+`"`) {
				t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
			}
		})
	}
}

func TestOversizedRequestIsRejectedBeforeDecision(t *testing.T) {
	decider := &scriptedDecider{answers: []byte(validAnswers)}
	request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(decisionBody))
	request.ContentLength = defaultMaxBodyBytes + 1
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	mustHandlerWithDecider(t, decider).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge || decider.calls != 0 {
		t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
	}
}

func TestJSONDepthLimit(t *testing.T) {
	body := strings.Repeat(`{"nested":`, maxJSONDepth+1) + `1` + strings.Repeat(`}`, maxJSONDepth+1)
	recorder := postJSON(t, mustHandler(t), "/v1/replays", body)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), `"reason":"json_too_deep"`) {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}

func TestConcurrentProjectsStayIsolated(t *testing.T) {
	handler, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"token-alpha": {"project-alpha"}, "token-beta": {"project-beta"}},
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   defaultMaxBodyBytes,
		Decider:        &scriptedDecider{answers: []byte(validAnswers)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	failures := make(chan string, 8)
	run := func(token, projectID, phrase, foreign string) {
		defer wg.Done()
		body := strings.Replace(strings.Replace(decisionBody, `"project-test"`, `"`+projectID+`"`, 1), "Create a storefront", phrase, 1)
		recorder := serve(handler, http.MethodPost, "/v1/decisions", token, body)
		if recorder.Code != http.StatusOK || strings.Contains(recorder.Body.String(), foreign) {
			failures <- recorder.Body.String()
		}
	}
	for range 2 {
		wg.Add(2)
		go run("token-alpha", "project-alpha", "alpha-state", "beta-state")
		go run("token-beta", "project-beta", "beta-state", "alpha-state")
	}
	wg.Wait()
	close(failures)
	for body := range failures {
		t.Fatalf("project isolation failed: %s", body)
	}
}

func TestAdmissionRejectsOverflowAndKeepsHealthOpen(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	handler, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"test-token": {"project-test"}},
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   defaultMaxBodyBytes,
		MaxInFlight:    1,
		Decider:        holdDecider{entered: entered, release: release, answers: []byte(validAnswers)},
	})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() { done <- postJSON(t, handler, "/v1/decisions", decisionBody).Code }()
	<-entered

	for _, token := range []string{"", "wrong-token"} {
		probe := &readProbe{}
		request := httptest.NewRequest(http.MethodPost, "/v1/decisions", nil)
		request.Body = probe
		request.ContentLength = 32
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if (recorder.Code != http.StatusUnauthorized && recorder.Code != http.StatusForbidden) || probe.reads != 0 {
			t.Fatalf("token %q status = %d reads = %d", token, recorder.Code, probe.reads)
		}
	}
	overflow := postJSON(t, handler, "/v1/decisions", decisionBody)
	if overflow.Code != http.StatusServiceUnavailable || !strings.Contains(overflow.Body.String(), `"reason":"admission_limited"`) {
		t.Fatalf("overflow status = %d body = %s", overflow.Code, overflow.Body)
	}
	if health := serve(handler, http.MethodGet, "/healthz", "", ""); health.Code != http.StatusOK {
		t.Fatalf("health status = %d", health.Code)
	}
	close(release)
	if code := <-done; code != http.StatusOK {
		t.Fatalf("held decision status = %d", code)
	}
}

func TestResponseBudgetIsCheckedBeforeAndAfterTheProvider(t *testing.T) {
	small := &scriptedDecider{answers: []byte(validAnswers)}
	handler, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"test-token": {"project-test"}},
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   responseReserve + 1024,
		MaxInFlight:    1,
		Decider:        small,
	})
	if err != nil {
		t.Fatal(err)
	}
	large := strings.Replace(decisionBody, "Create a storefront", strings.Repeat("storefront ", 3000), 1)
	before := postJSON(t, handler, "/v1/decisions", large)
	if before.Code != http.StatusUnprocessableEntity || small.calls != 0 || !strings.Contains(before.Body.String(), `"reason":"response_budget"`) || !strings.Contains(before.Body.String(), `"name":"decide","status":"failed"`) {
		t.Fatalf("status = %d calls = %d body = %s", before.Code, small.calls, before.Body)
	}

	oversized := &scriptedDecider{answers: []byte(strings.Replace(validAnswers, `"noul":0.9`, `"noul":0.9,"note":"`+strings.Repeat("OVERSIZED", 300000)+`"`, 1))}
	after := postJSON(t, mustHandlerWithDecider(t, oversized), "/v1/decisions", decisionBody)
	if after.Code != http.StatusUnprocessableEntity || oversized.calls != 1 || !strings.Contains(after.Body.String(), `"reason":"response_budget"`) || strings.Contains(after.Body.String(), "OVERSIZED") {
		t.Fatalf("status = %d calls = %d", after.Code, oversized.calls)
	}
}

func TestFullAnswerBudgetStillFitsTheResponse(t *testing.T) {
	probe := &budgetProbe{answers: []byte(validAnswers)}
	if recorder := postJSON(t, mustHandlerWithDecider(t, probe), "/v1/decisions", decisionBody); recorder.Code != http.StatusOK || probe.budget < len(validAnswers) {
		t.Fatalf("status = %d budget = %d", recorder.Code, probe.budget)
	}
	prefix := `{"fit":{"score":1.5,"type":"score"},"has_transactions":{"note":"`
	suffix := `","noul":0.9,"type":"noul"},"intent":{"choice":"portfolio","probabilities":{"cases":0.1,"portfolio":0.8,"products":0.1},"type":"choice"}}`
	padded := prefix + strings.Repeat("x", probe.budget-len(prefix)-len(suffix)) + suffix
	filler := &scriptedDecider{answers: []byte(padded)}
	recorder := postJSON(t, mustHandlerWithDecider(t, filler), "/v1/decisions", decisionBody)
	if recorder.Code != http.StatusOK || recorder.Body.Len() < probe.budget || recorder.Body.Len() > defaultMaxBodyBytes {
		t.Fatalf("status = %d bytes = %d budget = %d", recorder.Code, recorder.Body.Len(), probe.budget)
	}
}

func TestDeadlineAndDisconnectStopDecideVerifyAndReplay(t *testing.T) {
	t.Run("deadline before decide", func(t *testing.T) {
		decider := &scriptedDecider{answers: []byte(validAnswers)}
		recorder := serveStopped(t, decider, expiredContext(), "/v1/decisions", decisionBody)
		if recorder.Code != http.StatusGatewayTimeout || decider.calls != 0 || !strings.Contains(recorder.Body.String(), `"name":"decide","status":"failed"`) {
			t.Fatalf("status = %d calls = %d body = %s", recorder.Code, decider.calls, recorder.Body)
		}
	})
	t.Run("disconnect during decide", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		recorder := serveStopped(t, cancelDuringDecide{cancel: cancel}, ctx, "/v1/decisions", decisionBody)
		if recorder.Code != statusClientClosedRequest || !strings.Contains(recorder.Body.String(), `"name":"decide","status":"failed"`) || strings.Contains(recorder.Body.String(), `"reason":"decision_error"`) {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
		}
	})
	t.Run("deadline before verify", func(t *testing.T) {
		handler, err := NewHandler(Config{
			BearerTokens:   map[string][]string{"test-token": {"project-test"}},
			RequestTimeout: 40 * time.Millisecond,
			MaxBodyBytes:   defaultMaxBodyBytes,
			MaxInFlight:    1,
			Decider:        finishAfterDeadline{},
		})
		if err != nil {
			t.Fatal(err)
		}
		recorder := postJSON(t, handler, "/v1/decisions", decisionBody)
		if recorder.Code != http.StatusGatewayTimeout || !strings.Contains(recorder.Body.String(), `"name":"verify","status":"failed"`) || strings.Contains(recorder.Body.String(), `"structural_status":"valid"`) {
			t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
		}
	})
	for name, ctx := range map[string]context.Context{"disconnect before replay": canceledContext(), "deadline before replay": expiredContext()} {
		t.Run(name, func(t *testing.T) {
			recorder := serveStopped(t, nil, ctx, "/v1/replays", `{}`)
			if (recorder.Code != statusClientClosedRequest && recorder.Code != http.StatusGatewayTimeout) || !strings.Contains(recorder.Body.String(), `"name":"replay","status":"failed"`) {
				t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
			}
		})
	}
}

func TestReplayRefusesAnotherProject(t *testing.T) {
	owner := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(validAnswers)})
	bundle := replayBundle(t, postJSON(t, owner, "/v1/decisions", decisionBody))
	foreign, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"other-token": {"other-project"}},
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   defaultMaxBodyBytes,
	})
	if err != nil {
		t.Fatal(err)
	}
	denied := serve(foreign, http.MethodPost, "/v1/replays", "other-token", string(bundle))
	if denied.Code != http.StatusForbidden || !strings.Contains(denied.Body.String(), `"reason":"project_forbidden"`) || strings.Contains(denied.Body.String(), "storefront") {
		t.Fatalf("status = %d body = %s", denied.Code, denied.Body)
	}
	declared := postJSON(t, owner, "/v1/replays", `{"project_id":"other-project"}`)
	if declared.Code != http.StatusForbidden {
		t.Fatalf("declared status = %d body = %s", declared.Code, declared.Body)
	}
	if allowed := postJSON(t, owner, "/v1/replays", string(bundle)); allowed.Code != http.StatusOK || !strings.Contains(allowed.Body.String(), `"structural_status":"valid"`) {
		t.Fatalf("owner status = %d body = %s", allowed.Code, allowed.Body)
	}
}

func TestReplayRejectsBundlesOutsideTheDecisionSchema(t *testing.T) {
	handler := mustHandler(t)
	for _, body := range []string{`{}`, `{"protocol_version":"0.1","entity":{"project_id":"project-test"}}`, `[]`} {
		recorder := postJSON(t, handler, "/v1/replays", body)
		if recorder.Code != http.StatusUnprocessableEntity || !strings.Contains(recorder.Body.String(), `"reason":"invalid_replay_bundle"`) {
			t.Fatalf("body %s status = %d response = %s", body, recorder.Code, recorder.Body)
		}
	}
}

func TestPanicBecomesProblemJSON(t *testing.T) {
	recorder := postJSON(t, mustHandlerWithDecider(t, panicDecider{}), "/v1/decisions", decisionBody)
	if recorder.Code != http.StatusInternalServerError || recorder.Header().Get("Content-Type") != problemMediaType || !strings.Contains(recorder.Body.String(), `"reason":"internal_error"`) || strings.Contains(recorder.Body.String(), "SECRET-PANIC") {
		t.Fatalf("status = %d type = %q body = %s", recorder.Code, recorder.Header().Get("Content-Type"), recorder.Body)
	}
}

func TestDecisionDoesNotWriteRelativeFiles(t *testing.T) {
	directory := t.TempDir()
	t.Chdir(directory)
	if recorder := postJSON(t, mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(validAnswers)}), "/v1/decisions", decisionBody); recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("decision wrote %d files", len(entries))
	}
}

func TestNegotiationRejectsMethodMediaAndAccept(t *testing.T) {
	handler := mustHandler(t)
	method := serve(handler, http.MethodPost, "/v1/capabilities", "test-token", "")
	if method.Code != http.StatusMethodNotAllowed || method.Header().Get("Allow") != http.MethodGet || method.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("status = %d allow = %q", method.Code, method.Header().Get("Allow"))
	}
	for _, path := range []string{"/v1/decisions", "/v1/replays"} {
		get := serve(handler, http.MethodGet, path, "test-token", "")
		if get.Code != http.StatusMethodNotAllowed || get.Header().Get("Allow") != http.MethodPost {
			t.Fatalf("%s status = %d allow = %q", path, get.Code, get.Header().Get("Allow"))
		}
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Authorization", "Bearer test-token")
		request.Header.Set("Content-Type", "text/plain")
		media := httptest.NewRecorder()
		handler.ServeHTTP(media, request)
		if media.Code != http.StatusUnsupportedMediaType {
			t.Fatalf("%s media status = %d", path, media.Code)
		}
	}
	for _, header := range []string{
		"text/html, application/json;q=0",
		"*/*;q=0.8, application/json;q=0",
		"application/json;q=0, application/json;q=0",
		"application/*;q=1, application/json;q=0",
		"application/problem+json",
		"application/json;q=0, */*;q=0.1",
	} {
		request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
		request.Header.Set("Authorization", "Bearer test-token")
		request.Header.Set("Accept", header)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusNotAcceptable || !strings.Contains(recorder.Body.String(), `"reason":"not_acceptable"`) {
			t.Fatalf("header %q status = %d body = %s", header, recorder.Code, recorder.Body)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Accept", "*/*;q=0, application/json;q=0, application/json;q=0.2")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body)
	}
}

func TestFreshInstancesIgnoreIdempotencyKey(t *testing.T) {
	first := &scriptedDecider{answers: []byte(validAnswers)}
	second := &scriptedDecider{answers: []byte(validAnswers)}
	responses := make([]string, 0, 2)
	for _, decider := range []*scriptedDecider{first, second} {
		request := httptest.NewRequest(http.MethodPost, "/v1/decisions", strings.NewReader(decisionBody))
		request.Header.Set("Authorization", "Bearer test-token")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Idempotency-Key", "same-request-key")
		recorder := httptest.NewRecorder()
		mustHandlerWithDecider(t, decider).ServeHTTP(recorder, request)
		responses = append(responses, recorder.Body.String())
	}
	if first.calls != 1 || second.calls != 1 || responses[0] != responses[1] || !strings.Contains(responses[0], `"structural_status":"valid"`) {
		t.Fatalf("calls = %d %d", first.calls, second.calls)
	}
}

func TestSecurityProfileRejectsCookiesAndCORS(t *testing.T) {
	handler := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(validAnswers)})
	cookie := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	cookie.Header.Set("Cookie", "session=admin")
	cookieRecorder := httptest.NewRecorder()
	handler.ServeHTTP(cookieRecorder, cookie)
	if cookieRecorder.Code != http.StatusUnauthorized || cookieRecorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("status = %d cors = %q", cookieRecorder.Code, cookieRecorder.Header().Get("Access-Control-Allow-Origin"))
	}
	preflight := httptest.NewRequest(http.MethodOptions, "/v1/decisions", nil)
	preflight.Header.Set("Origin", "https://example.invalid")
	preflight.Header.Set("Access-Control-Request-Method", "POST")
	preflightRecorder := httptest.NewRecorder()
	handler.ServeHTTP(preflightRecorder, preflight)
	if preflightRecorder.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("preflight granted %q", preflightRecorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestNoExecutionOrRemovedRoutes(t *testing.T) {
	handler := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(validAnswers)})
	for _, path := range []string{"/v1/executions", "/v1/evaluations"} {
		recorder := postJSON(t, handler, path, `{"approved":true}`)
		if recorder.Code != http.StatusNotFound || !strings.Contains(recorder.Body.String(), `"reason":"not_found"`) || recorder.Header().Get("Content-Type") != problemMediaType {
			t.Fatalf("%s returned status %d body %s", path, recorder.Code, recorder.Body)
		}
	}
}

func serve(handler http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func postJSON(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	return serve(handler, http.MethodPost, path, "test-token", body)
}

func serveStopped(t *testing.T, decider Decider, ctx context.Context, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	handler, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"test-token": {"project-test"}},
		RequestTimeout: time.Second,
		MaxBodyBytes:   defaultMaxBodyBytes,
		MaxInFlight:    1,
		Decider:        decider,
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body)).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}

func expiredContext() context.Context {
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	cancel()
	return ctx
}

func replayBundle(t *testing.T, recorder *httptest.ResponseRecorder) json.RawMessage {
	t.Helper()
	var response struct {
		ReplayBundle json.RawMessage `json:"replay_bundle"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil || len(response.ReplayBundle) == 0 {
		t.Fatalf("no replay bundle: status = %d body = %s", recorder.Code, recorder.Body)
	}
	return response.ReplayBundle
}

func openAPIDocument(t *testing.T) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "wire", "schema", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value.(map[string]any)
}

func openAPISchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("https://lex.fastygo.dev/openapi.json", openAPIDocument(t)); err != nil {
		t.Fatal(err)
	}
	checkID := "https://lex.fastygo.dev/schema/check-" + name
	if err := compiler.AddResource(checkID, map[string]any{
		"$ref": "https://lex.fastygo.dev/openapi.json#/components/schemas/" + name,
	}); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(checkID)
	if err != nil {
		t.Fatal(err)
	}
	return schema
}

func validateSchema(t *testing.T, name string, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if err := openAPISchema(t, name).Validate(strictJSON(t, recorder.Body.Bytes())); err != nil {
		t.Fatalf("%s: %v\n%s", name, err, recorder.Body)
	}
}

func strictJSON(t *testing.T, raw []byte) any {
	t.Helper()
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func assertNoTimestamps(t *testing.T, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == "timestamp" || strings.HasSuffix(key, "_at") {
				t.Fatalf("response contains timestamp field %s", key)
			}
			assertNoTimestamps(t, child)
		}
	case []any:
		for _, child := range typed {
			assertNoTimestamps(t, child)
		}
	}
}

func mustHandlerWithDecider(t *testing.T, decider Decider) http.Handler {
	t.Helper()
	handler, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"test-token": {"project-test"}},
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   defaultMaxBodyBytes,
		Decider:        decider,
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}

func mustHandler(t *testing.T) http.Handler {
	t.Helper()
	return mustHandlerWithDecider(t, nil)
}

const testAdapterID = typesafe.AdapterID

type scriptedDecider struct {
	mu             sync.Mutex
	answers        []byte
	calls          int
	requestID      string
	evaluationTime *float64
	usage          json.RawMessage
}

func (*scriptedDecider) AdapterID() string      { return testAdapterID }
func (*scriptedDecider) AdapterVersion() string { return typesafe.AdapterVersion }

func (decider *scriptedDecider) Evaluate(context.Context, any, map[string]any) (Decision, error) {
	decider.mu.Lock()
	decider.calls++
	decider.mu.Unlock()
	return Decision{
		ResolvedModel: typesafe.Model, Answers: decider.answers,
		RequestID: decider.requestID, EvaluationTimeMS: decider.evaluationTime, Usage: decider.usage,
	}, nil
}

type capturingDecider struct {
	mu        sync.Mutex
	answers   []byte
	state     any
	questions map[string]any
}

func (*capturingDecider) AdapterID() string      { return testAdapterID }
func (*capturingDecider) AdapterVersion() string { return typesafe.AdapterVersion }

func (decider *capturingDecider) Evaluate(_ context.Context, state any, questions map[string]any) (Decision, error) {
	decider.mu.Lock()
	decider.state = state
	decider.questions = questions
	decider.mu.Unlock()
	return Decision{ResolvedModel: typesafe.Model, Answers: decider.answers}, nil
}

type failingDecider struct{ err error }

func (failingDecider) AdapterID() string      { return testAdapterID }
func (failingDecider) AdapterVersion() string { return typesafe.AdapterVersion }
func (decider failingDecider) Evaluate(context.Context, any, map[string]any) (Decision, error) {
	return Decision{}, decider.err
}

type retryableProviderError struct{}

func (retryableProviderError) Error() string           { return "decision provider is temporarily unavailable" }
func (retryableProviderError) ProviderRetryable() bool { return true }

type responseBudgetError struct{}

func (responseBudgetError) Error() string               { return "decision response exceeds the response budget" }
func (responseBudgetError) ExceedsResponseBudget() bool { return true }

type holdDecider struct {
	entered chan struct{}
	release chan struct{}
	answers []byte
}

func (holdDecider) AdapterID() string      { return testAdapterID }
func (holdDecider) AdapterVersion() string { return typesafe.AdapterVersion }
func (decider holdDecider) Evaluate(context.Context, any, map[string]any) (Decision, error) {
	decider.entered <- struct{}{}
	<-decider.release
	return Decision{ResolvedModel: typesafe.Model, Answers: decider.answers}, nil
}

type readProbe struct{ reads int }

func (probe *readProbe) Read([]byte) (int, error) {
	probe.reads++
	return 0, errors.New("body was read")
}

func (*readProbe) Close() error { return nil }

type budgetProbe struct {
	answers []byte
	budget  int
}

func (*budgetProbe) AdapterID() string      { return testAdapterID }
func (*budgetProbe) AdapterVersion() string { return typesafe.AdapterVersion }
func (decider *budgetProbe) Evaluate(ctx context.Context, _ any, _ map[string]any) (Decision, error) {
	if n, ok := AnswerBudget(ctx); ok {
		decider.budget = int(n)
	}
	return Decision{ResolvedModel: typesafe.Model, Answers: decider.answers}, nil
}

type cancelDuringDecide struct{ cancel context.CancelFunc }

func (cancelDuringDecide) AdapterID() string      { return testAdapterID }
func (cancelDuringDecide) AdapterVersion() string { return typesafe.AdapterVersion }
func (decider cancelDuringDecide) Evaluate(ctx context.Context, _ any, _ map[string]any) (Decision, error) {
	decider.cancel()
	<-ctx.Done()
	return Decision{}, ctx.Err()
}

type finishAfterDeadline struct{}

func (finishAfterDeadline) AdapterID() string      { return testAdapterID }
func (finishAfterDeadline) AdapterVersion() string { return typesafe.AdapterVersion }
func (finishAfterDeadline) Evaluate(ctx context.Context, _ any, _ map[string]any) (Decision, error) {
	<-ctx.Done()
	return Decision{ResolvedModel: typesafe.Model, Answers: []byte(validAnswers)}, nil
}

type panicDecider struct{}

func (panicDecider) AdapterID() string      { return testAdapterID }
func (panicDecider) AdapterVersion() string { return typesafe.AdapterVersion }
func (panicDecider) Evaluate(context.Context, any, map[string]any) (Decision, error) {
	panic("SECRET-PANIC")
}
