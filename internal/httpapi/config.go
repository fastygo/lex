package httpapi

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
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
	BearerTokens   map[string][]string
	RequestTimeout time.Duration
	MaxBodyBytes   int64
	MaxInFlight    int
	Decider        Decider
}

// LoadConfig reads the API boundary configuration without exposing secrets.
func LoadConfig() (Config, error) {
	config := Config{
		RequestTimeout: defaultRequestTimeout,
		MaxBodyBytes:   defaultMaxBodyBytes,
	}

	rawTokens := strings.TrimSpace(os.Getenv("LEX_BEARER_TOKENS"))
	if rawTokens == "" {
		return Config{}, fmt.Errorf("LEX_BEARER_TOKENS is required")
	}
	if err := json.Unmarshal([]byte(rawTokens), &config.BearerTokens); err != nil {
		return Config{}, fmt.Errorf("parse LEX_BEARER_TOKENS: %w", err)
	}
	if err := config.validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c *Config) validate() error {
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
