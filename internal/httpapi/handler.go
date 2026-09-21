// Package httpapi provides the synchronous HTTP boundary for LeX.
package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	frameworkapp "github.com/fastygo/framework/pkg/app"
	"github.com/fastygo/framework/pkg/web/security"
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
	mux.Handle("/v1/capabilities", authenticated(config, http.HandlerFunc(capabilities)))

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
		if !authorizedToken(token, config.BearerTokens) {
			writeProblem(w, http.StatusForbidden, "authentication_denied", "the bearer token is not authorized")
			return
		}
		next.ServeHTTP(w, request)
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

func authorizedToken(token string, tokens map[string][]string) bool {
	authorized := 0
	for expected := range tokens {
		authorized |= subtle.ConstantTimeCompare([]byte(token), []byte(expected))
	}
	return authorized == 1
}

func capabilities(w http.ResponseWriter, request *http.Request) {
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
			"evaluation": false,
			"replay":     false,
			"execution":  false,
		},
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
