package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerCapabilitiesRequiresBearerToken(t *testing.T) {
	handler := mustHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != problemMediaType {
		t.Fatalf("content type = %q, want %q", contentType, problemMediaType)
	}
}

func TestHandlerCapabilitiesReturnsNoStore(t *testing.T) {
	handler := mustHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", cacheControl)
	}
	if got := recorder.Body.String(); got == "" {
		t.Fatal("expected capability response body")
	}
}

func TestHandlerHealthDoesNotRequireBearerToken(t *testing.T) {
	handler := mustHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
}

func TestHandlerRejectsUnauthorizedBearerToken(t *testing.T) {
	handler := mustHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/capabilities", nil)
	request.Header.Set("Authorization", "Bearer wrong-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusForbidden)
	}
}

func TestReplayRejectsMalformedBundle(t *testing.T) {
	handler := mustHandler(t)
	request := httptest.NewRequest(http.MethodPost, "/v1/replays", strings.NewReader(`{"protocol_version":"0.1-draft"}`))
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusUnprocessableEntity)
	}
}

func mustHandler(t *testing.T) http.Handler {
	t.Helper()
	handler, err := NewHandler(Config{
		BearerTokens:   map[string][]string{"test-token": {"project-test"}},
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   defaultMaxBodyBytes,
	})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	return handler
}
