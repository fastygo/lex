package httpapi

import (
	"strings"
	"testing"
)

func TestLoadConfigReadsProjectBinding(t *testing.T) {
	t.Setenv("LEX_BEARER_TOKENS", `{"test-token":["example-project"]}`)
	config, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(config.BearerTokens["test-token"]) != 1 || config.BearerTokens["test-token"][0] != "example-project" {
		t.Fatalf("projects = %#v", config.BearerTokens["test-token"])
	}
	if config.RequestTimeout != defaultRequestTimeout || config.MaxBodyBytes != defaultMaxBodyBytes || config.MaxInFlight != defaultMaxInFlight {
		t.Fatalf("timeout = %s body = %d inflight = %d", config.RequestTimeout, config.MaxBodyBytes, config.MaxInFlight)
	}
}

func TestLoadConfigRejectsAmbiguousTokensWithoutEcho(t *testing.T) {
	const token = "super-secret-token"
	for _, raw := range []string{
		`{"` + token + `":["project-a"],"` + token + `":["project-b"]}`,
		`{"` + token + `":["project-a"]} {"` + token + `":["project-b"]}`,
		`["` + token + `"]`,
		`{"` + token + `":"project-a"}`,
	} {
		t.Setenv("LEX_BEARER_TOKENS", raw)
		_, err := LoadConfig()
		if err == nil || strings.Contains(err.Error(), token) {
			t.Fatalf("accepted ambiguous bearer configuration or echoed the token: %v", err)
		}
	}
}
