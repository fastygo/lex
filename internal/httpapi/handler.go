// Package httpapi provides the synchronous HTTP boundary for LeX.
package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	frameworkapp "github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/security"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/wire"
)

const problemMediaType = "application/problem+json"

// NewHandler builds the Framework-composed API handler.
func NewHandler(config Config) (http.Handler, error) {
	if err := config.validate(); err != nil {
		return nil, err
	}

	builder := frameworkapp.New(frameworkapp.Config{}).
		DisableStatic().
		WithSecurity(security.Config{Enabled: false}).
		WithHealthEndpoints("/healthz", "")
	mux := builder.Mux()
	mux.Handle("/v1/capabilities", authenticated(config, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		capabilities(w, request, config.Decider != nil)
	})))
	mux.Handle("/v1/evaluations", authenticated(config, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		evaluate(w, request, config.Decider)
	})))
	mux.Handle("/v1/replays", authenticated(config, http.HandlerFunc(replay)))

	return withRequestLimits(builder.Build().Handler(), config), nil
}

func withRequestLimits(next http.Handler, config Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.ContentLength > config.MaxBodyBytes {
			writeProblem(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds the configured limit")
			return
		}
		request.Body = http.MaxBytesReader(w, request.Body, config.MaxBodyBytes)
		ctx, cancel := context.WithTimeout(request.Context(), config.RequestTimeout)
		defer cancel()
		next.ServeHTTP(w, request.WithContext(ctx))
	})
}

func authenticated(config Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		token, ok := bearerToken(request.Header.Get("Authorization"))
		if !ok {
			writeProblem(w, http.StatusUnauthorized, "authentication_required", "a bearer token is required")
			return
		}
		projects, ok := authorizedProjects(token, config.BearerTokens)
		if !ok {
			writeProblem(w, http.StatusForbidden, "authentication_denied", "the bearer token is not authorized")
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

func capabilities(w http.ResponseWriter, request *http.Request, evaluationEnabled bool) {
	if request.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeProblem(w, http.StatusMethodNotAllowed, "method_not_allowed", "only GET is supported")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"protocol_status": "working_draft",
		"context": map[string]string{
			"capability": "memory-exact-v1",
		},
		"operations": map[string]bool{
			"evaluation": evaluationEnabled,
			"replay":     true,
			"execution":  false,
		},
	})
}

func replay(w http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeProblem(w, http.StatusMethodNotAllowed, "method_not_allowed", "only POST is supported")
		return
	}
	contentType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		writeProblem(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return
	}
	raw, err := io.ReadAll(request.Body)
	if err != nil {
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			writeProblem(w, http.StatusBadRequest, "invalid_body", "request body is unavailable")
			return
		}
		writeProblem(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body exceeds the configured limit")
		return
	}
	report, err := wire.Replay(raw)
	if err != nil {
		writeProblem(w, http.StatusUnprocessableEntity, "invalid_replay_bundle", "replay bundle failed deterministic structural validation")
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"replay_status":      "verdict_reproduced",
		"verdict":            report.Verdict,
		"findings":           report.Findings,
		"policy_calibration": profile.Calibration,
	})
}

func writeProblem(w http.ResponseWriter, status int, reason, detail string) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", problemMediaType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   "https://lex.fastygo.dev/problems/" + reason,
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
		"reason": reason,
	})
}
