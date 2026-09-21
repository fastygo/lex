// Package httpapi provides the synchronous HTTP boundary for LeX.
package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	contextmemory "github.com/fastygo/context/pkg/contextkit/runtime"
	frameworkapp "github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/security"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/verify"
	"github.com/fastygo/lex/internal/wire"
)

const problemMediaType = "application/problem+json"

// NewHandler builds the Framework-composed API handler.
func NewHandler(config Config) (http.Handler, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	verifier := config.verifier()
	builder := frameworkapp.New(frameworkapp.Config{}).
		DisableStatic().
		WithSecurity(security.Config{Enabled: false}).
		WithHealthEndpoints("/healthz", "")
	mux := builder.Mux()
	mux.Handle("/v1/capabilities", recoverProblem(authenticated(config, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		capabilities(w, request, config)
	}))))
	mux.Handle("/v1/evaluations", recoverProblem(authenticated(config, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		evaluate(w, request, config.Decider, config.MaxBodyBytes, verifier)
	}))))
	mux.Handle("/v1/replays", recoverProblem(authenticated(config, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { replay(w, r, verifier) }))))

	return withKnownRoutes(withRequestLimits(builder.Build().Handler(), config, newAdmission(config.MaxInFlight))), nil
}

func recoverProblem(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		defer func() {
			if recover() != nil {
				slog.Error("http.panic")
				writeProblem(w, http.StatusInternalServerError, reasonInternalError, "the request could not be completed")
			}
		}()
		next.ServeHTTP(w, request)
	})
}

func withKnownRoutes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/healthz", "/v1/capabilities", "/v1/evaluations", "/v1/replays":
			next.ServeHTTP(w, request)
		default:
			writeProblem(w, http.StatusNotFound, reasonNotFound, "the route is not provided")
		}
	})
}

func withRequestLimits(next http.Handler, config Config, gate *admission) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/healthz" {
			token, ok := bearerToken(request.Header.Get("Authorization"))
			if !ok {
				writeProblem(w, http.StatusUnauthorized, reasonAuthenticationRequired, "a bearer token is required")
				return
			}
			projects, ok := authorizedProjects(token, config.BearerTokens)
			if !ok {
				writeProblem(w, http.StatusForbidden, reasonAuthenticationDenied, "the bearer token is not authorized")
				return
			}
			request = request.WithContext(context.WithValue(request.Context(), projectsKey{}, projects))
			if !gate.acquire() {
				writeProblem(w, http.StatusServiceUnavailable, reasonAdmissionLimited, "too many requests are already running in this process")
				return
			}
			defer gate.release()
		}
		if request.ContentLength > config.MaxBodyBytes {
			writeProblem(w, http.StatusRequestEntityTooLarge, reasonBodyTooLarge, "request body exceeds the configured limit")
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, config.MaxBodyBytes)
		if request.Body != nil && request.ContentLength != 0 {
			raw, err := io.ReadAll(request.Body)
			if err != nil {
				writeProblem(w, http.StatusRequestEntityTooLarge, reasonBodyTooLarge, "request body exceeds the configured limit")
				return
			}
			if !utf8.Valid(raw) {
				writeProblem(w, http.StatusBadRequest, reasonInvalidJSON, "request body must be valid UTF-8")
				return
			}
			if jsonDepth(raw) > maxJSONDepth {
				writeProblem(w, http.StatusBadRequest, reasonJSONTooDeep, "JSON nesting exceeds the configured limit")
				return
			}
			request.Body = io.NopCloser(strings.NewReader(string(raw)))
		}
		ctx, cancel := context.WithTimeout(request.Context(), config.RequestTimeout)
		defer cancel()
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

type admission struct{ slots chan struct{} }

func newAdmission(limit int) *admission {
	return &admission{slots: make(chan struct{}, limit)}
}

func (gate *admission) acquire() bool {
	select {
	case gate.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

func (gate *admission) release() { <-gate.slots }

func jsonDepth(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	depth := 0
	maxDepth := 0
	for {
		token, err := decoder.Token()
		if err != nil {
			return maxDepth
		}
		switch token {
		case json.Delim('{'), json.Delim('['):
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case json.Delim('}'), json.Delim(']'):
			if depth > 0 {
				depth--
			}
		}
	}
}

func authenticated(config Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		token, ok := bearerToken(request.Header.Get("Authorization"))
		if !ok {
			writeProblem(w, http.StatusUnauthorized, reasonAuthenticationRequired, "a bearer token is required")
			return
		}
		projects, ok := authorizedProjects(token, config.BearerTokens)
		if !ok {
			writeProblem(w, http.StatusForbidden, reasonAuthenticationDenied, "the bearer token is not authorized")
			return
		}
		next.ServeHTTP(w, request.WithContext(context.WithValue(request.Context(), projectsKey{}, projects)))
	})
}

func bearerToken(header string) (string, bool) {
	const scheme = "Bearer "
	if !strings.HasPrefix(header, scheme) {
		return "", false
	}
	token := strings.TrimPrefix(header, scheme)
	return token, token != "" && !strings.ContainsAny(token, " \t\r\n")
}

type projectsKey struct{}

func acceptsJSON(header string) bool {
	if strings.TrimSpace(header) == "" {
		return true
	}
	bestSpecificity := 0
	bestQuality := 0.0
	for _, part := range strings.Split(header, ",") {
		media, params, err := mime.ParseMediaType(strings.TrimSpace(part))
		if err != nil {
			return false
		}
		quality := 1.0
		if raw, ok := params["q"]; ok {
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil || parsed < 0 || parsed > 1 {
				return false
			}
			quality = parsed
		}
		specificity := jsonMediaSpecificity(media)
		if specificity > bestSpecificity || (specificity == bestSpecificity && quality > bestQuality) {
			bestSpecificity = specificity
			bestQuality = quality
		}
	}
	return bestSpecificity > 0 && bestQuality > 0
}

func jsonMediaSpecificity(media string) int {
	switch media {
	case "application/json":
		return 3
	case "application/*":
		return 2
	case "*/*":
		return 1
	default:
		return 0
	}
}

func authorizedProjects(token string, tokens map[string][]string) ([]string, bool) {
	var projects []string
	matched := 0
	for expected, allowed := range tokens {
		if subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1 {
			matched++
			projects = append([]string{}, allowed...)
		}
	}
	return projects, matched == 1
}

func capabilities(w http.ResponseWriter, request *http.Request, config Config) {
	if request.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeProblem(w, http.StatusMethodNotAllowed, reasonMethodNotAllowed, "only GET is supported")
		return
	}
	if !acceptsJSON(request.Header.Get("Accept")) {
		writeProblem(w, http.StatusNotAcceptable, reasonNotAcceptable, "Accept must allow application/json")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"protocol_status": "working_draft",
		"context": map[string]string{
			"capability": "memory-exact-v1",
			"retrieval":  "exact_phrase",
		},
		"operations": map[string]bool{
			"evaluation": config.Decider != nil,
			"replay":     true,
			"execution":  false,
		},
		"policy": policyDisclosure(),
		"limits": map[string]int64{
			"max_sources":        int64(contextmemory.MaxSources),
			"max_body_bytes":     config.MaxBodyBytes,
			"focus_max_items":    int64(profile.FocusMaxItems),
			"focus_max_chars":    int64(profile.FocusMaxChars),
			"request_timeout_ms": config.RequestTimeout.Milliseconds(),
			"process_admission":  int64(config.MaxInFlight),
		},
		"retention": retentionDisclosure(),
	})
}

func replay(w http.ResponseWriter, request *http.Request, verifier wire.Verifier) {
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProblem(w, http.StatusMethodNotAllowed, reasonMethodNotAllowed, "only POST is supported")
		return
	}
	if !acceptsJSON(request.Header.Get("Accept")) {
		writeProblem(w, http.StatusNotAcceptable, reasonNotAcceptable, "Accept must allow application/json")
		return
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeProblem(w, http.StatusUnsupportedMediaType, reasonUnsupportedMediaType, "Content-Type must be application/json")
		return
	}
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			writeProblem(w, http.StatusBadRequest, reasonInvalidBody, "request body is unavailable")
			return
		}
		writeProblem(w, http.StatusRequestEntityTooLarge, reasonBodyTooLarge, "request body exceeds the configured limit")
		return
	}
	if writeRequestStop(w, request.Context().Err(), replayTraceStatus("failed")) {
		return
	}
	projects, _ := request.Context().Value(projectsKey{}).([]string)
	if projectID, ok := declaredReplayProject(raw); ok && !containsProject(projects, projectID) {
		writeProblem(w, http.StatusForbidden, reasonProjectForbidden, "the authenticated principal cannot access this project")
		return
	}
	report, err := verifier.ReplayContext(request.Context(), raw)
	if err != nil {
		if writeRequestStop(w, err, replayTraceStatus("failed")) {
			return
		}
		if errors.Is(err, verify.ErrUnusablePolicy) {
			status, reason, detail := replayFailure(err)
			tracedProblem(w, status, reason, detail, replayTraceStatus("failed"))
			return
		}
		writeProblem(w, http.StatusUnprocessableEntity, reasonInvalidReplayBundle, "replay bundle failed deterministic structural validation")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"replay_status":      "verdict_reproduced",
		"verdict":            report.Verdict,
		"findings":           report.Findings,
		"policy_calibration": profile.Calibration,
		"retention":          retentionDisclosure(),
		"trace":              replayTrace(),
	})
}

type retentionView struct {
	ServerHistory bool   `json:"server_history"`
	Replay        string `json:"replay"`
	Idempotency   string `json:"idempotency"`
}

func declaredReplayProject(raw []byte) (string, bool) {
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return "", false
	}
	object, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	entity, ok := object["entity"].(map[string]any)
	if !ok {
		return "", false
	}
	projectID, ok := entity["project_id"].(string)
	if !ok || projectID == "" {
		return "", false
	}
	return projectID, true
}

func retentionDisclosure() retentionView {
	return retentionView{ServerHistory: false, Replay: "caller_owned", Idempotency: "none"}
}

func writeProblem(w http.ResponseWriter, status int, reason, detail string) {
	writeProblemBody(w, status, reason, detail, nil)
}

func writeVerdictProblem(w http.ResponseWriter, status int, reason, detail string, report wire.Report, bundle []byte, meta map[string]any, maxBodyBytes int64) {
	body := map[string]any{
		"verdict":       report.Verdict,
		"findings":      report.Findings,
		"trace":         trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "completed"),
		"replay_bundle": json.RawMessage(bundle),
	}
	if meta != nil {
		body["adapter_metadata"] = meta
	}
	encoded, err := encodeProblem(status, reason, detail, body)
	if err != nil || int64(len(encoded)) > maxBodyBytes {
		tracedProblem(w, http.StatusUnprocessableEntity, reasonResponseBudget, "the evaluation response does not fit the response budget", trace("receive", "completed", "pack", "completed", "decide", "completed", "verify", "failed"))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", problemMediaType)
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func writeProblemBody(w http.ResponseWriter, status int, reason, detail string, extra map[string]any) {
	encoded, err := encodeProblem(status, reason, detail, extra)
	if err != nil {
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", problemMediaType)
	w.WriteHeader(status)
	_, _ = w.Write(encoded)
}

func encodeProblem(status int, reason, detail string, extra map[string]any) ([]byte, error) {
	title := http.StatusText(status)
	if title == "" {
		title = "Client Closed Request"
	}
	body := map[string]any{
		"type":   "https://lex.fastygo.dev/problems/" + reason,
		"title":  title,
		"status": status,
		"detail": detail,
		"reason": reason,
	}
	for key, value := range extra {
		body[key] = value
	}
	return json.Marshal(body)
}
