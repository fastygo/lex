package systemone

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestEndpointAllowlistRejectsLoopbackDisguises(t *testing.T) {
	call := Call{
		APIKey:           "test-key",
		OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model:            "jev-1.13.0",
		AllowLoopback:    true,
	}
	for _, endpoint := range []string{
		"http://127.0.0.1:80@evil.example/systemone",
		"http://127.0.0.1.evil.example:80/systemone",
		"http://2130706433:80/systemone",
		"https://127.0.0.1:80/systemone",
		"http://127.0.0.1/systemone",
	} {
		call.Endpoint = endpoint
		if _, err := Evaluate(context.Background(), call, map[string]any{}, map[string]any{}); err == nil {
			t.Fatalf("accepted %s", endpoint)
		}
	}
	if !endpointAllowed(Call{Endpoint: "https://api.typesafe.ai/v1/systemone", OfficialEndpoint: "https://api.typesafe.ai/v1/systemone"}) {
		t.Fatal("rejected the official endpoint")
	}
}

func TestEvaluateDoesNotFollowRedirect(t *testing.T) {
	var followed atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		followed.Add(1)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, target.URL, http.StatusFound)
	}))
	defer source.Close()

	_, err := Evaluate(context.Background(), Call{
		APIKey:           "test-key",
		Endpoint:         source.URL,
		OfficialEndpoint: "https://api.typesafe.ai/v1/systemone",
		Model:            "jev-1.13.0",
		AllowLoopback:    true,
		HTTP:             source.Client(),
	}, map[string]any{}, map[string]any{})
	if err == nil || followed.Load() != 0 {
		t.Fatalf("followed = %d err = %v", followed.Load(), err)
	}
}
