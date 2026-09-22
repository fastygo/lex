package httpapi

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fastygo/lex/internal/adapters/openrouter"
	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/profile"
	"github.com/fastygo/lex/internal/wire"
)

const (
	defaultRequestTimeout = 20 * time.Second
	defaultMaxBodyBytes   = 2 << 20
	defaultMaxInFlight    = 4
	maxJSONDepth          = 32
	responseReserve       = 64 << 10
)

// Config contains deployment-owned settings for the HTTP boundary.
type Config struct {
	BearerTokens        map[string][]string
	RequestTimeout      time.Duration
	MaxBodyBytes        int64
	MaxInFlight         int
	Decider             Decider
	HostedResolvedModel string
	// Profiles are compatibility-only claim evaluation profiles. Generic
	// decisions use caller-owned QuestionSets and never inspect this registry.
	Profiles profile.Registry
}

// LoadConfig reads the API boundary configuration without exposing secrets.
func LoadConfig() (Config, error) {
	config := Config{
		HostedResolvedModel: os.Getenv("LEX_HOSTED_RESOLVED_MODEL"),
		RequestTimeout:      defaultRequestTimeout,
		MaxBodyBytes:        defaultMaxBodyBytes,
	}

	rawTokens := strings.TrimSpace(os.Getenv("LEX_BEARER_TOKENS"))
	if rawTokens == "" {
		return Config{}, fmt.Errorf("LEX_BEARER_TOKENS is required")
	}
	tokens, err := parseBearerTokens(rawTokens)
	if err != nil {
		return Config{}, err
	}
	config.BearerTokens = tokens
	if err := config.validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c *Config) validate() error {
	if c.HostedResolvedModel != "" && !openrouter.ValidResolvedPin(c.HostedResolvedModel) {
		return fmt.Errorf("LEX_HOSTED_RESOLVED_MODEL must be an exact resolved identity")
	}
	if c.Decider != nil && c.Decider.AdapterID() == openrouter.AdapterID && c.HostedResolvedModel == "" {
		return fmt.Errorf("hosted adapter requires a resolved model pin")
	}
	if c.MaxInFlight == 0 {
		c.MaxInFlight = defaultMaxInFlight
	}
	if c.MaxInFlight < 1 || c.MaxInFlight > defaultMaxInFlight {
		return fmt.Errorf("in-flight limit must be between 1 and %d", defaultMaxInFlight)
	}
	if len(c.BearerTokens) == 0 {
		return fmt.Errorf("at least one bearer token is required")
	}
	for token, projects := range c.BearerTokens {
		if strings.TrimSpace(token) == "" {
			return fmt.Errorf("bearer token must not be empty")
		}
		if len(projects) == 0 {
			return fmt.Errorf("bearer token must allow at least one project")
		}
		seen := make(map[string]struct{}, len(projects))
		for _, projectID := range projects {
			if strings.TrimSpace(projectID) == "" {
				return fmt.Errorf("allowed project must not be empty")
			}
			if _, exists := seen[projectID]; exists {
				return fmt.Errorf("allowed project must not be repeated")
			}
			seen[projectID] = struct{}{}
		}
	}
	if c.RequestTimeout <= 0 || c.RequestTimeout > defaultRequestTimeout {
		return fmt.Errorf("request timeout must be between zero and %s", defaultRequestTimeout)
	}
	if c.MaxBodyBytes < 1 || c.MaxBodyBytes > defaultMaxBodyBytes {
		return fmt.Errorf("body limit must be between one and %d bytes", defaultMaxBodyBytes)
	}
	return nil
}

func parseBearerTokens(raw string) (map[string][]string, error) {
	value, err := canonical.DecodeJSON([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("LEX_BEARER_TOKENS must be one JSON object")
	}
	object, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("LEX_BEARER_TOKENS must be one JSON object")
	}
	tokens := make(map[string][]string, len(object))
	for token, rawProjects := range object {
		projects, ok := rawProjects.([]any)
		if !ok {
			return nil, fmt.Errorf("LEX_BEARER_TOKENS values must be project arrays")
		}
		allowed := make([]string, 0, len(projects))
		for _, rawProject := range projects {
			projectID, ok := rawProject.(string)
			if !ok {
				return nil, fmt.Errorf("LEX_BEARER_TOKENS project ids must be strings")
			}
			allowed = append(allowed, projectID)
		}
		tokens[token] = allowed
	}
	return tokens, nil
}

func (c Config) verifier() wire.Verifier {
	return wire.NewVerifier(c.legacyProfiles(), c.adapterPins())
}

func (c Config) decisionVerifier() wire.DecisionVerifier {
	return wire.NewDecisionVerifier(c.adapterPins())
}

func (c Config) adapterPins() map[string]wire.AdapterPin {
	pins := map[string]wire.AdapterPin{typesafe.AdapterID: {Version: typesafe.AdapterVersion, Model: typesafe.Model}}
	if c.HostedResolvedModel != "" {
		pins[openrouter.AdapterID] = wire.AdapterPin{Version: openrouter.AdapterVersion, Model: c.HostedResolvedModel}
	}
	return pins
}

// profile is the profile live evaluations use.
func (c Config) profile() profile.Profile {
	prof, _ := c.legacyProfiles().Default()
	return prof
}
