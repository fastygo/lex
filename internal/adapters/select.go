// Package adapters selects one decision adapter without silent fallback.
package adapters

import "fmt"

// Kind identifies the configured decision path.
type Kind string

const (
	// KindNone means evaluation is not configured.
	KindNone Kind = ""
	// KindDirect is the pinned direct System One endpoint.
	KindDirect Kind = "direct"
	// KindHosted is the pinned hosted System One endpoint.
	KindHosted Kind = "hosted"
)

// Select chooses at most one adapter. Two configured credentials require an
// explicit selector. An empty result leaves evaluation unavailable.
func Select(selector string, directConfigured, hostedConfigured bool) (Kind, error) {
	switch selector {
	case string(KindDirect):
		if !directConfigured {
			return KindNone, fmt.Errorf("LEX_TYPESAFE_API_KEY is required")
		}
		return KindDirect, nil
	case string(KindHosted):
		if !hostedConfigured {
			return KindNone, fmt.Errorf("LEX_OPENROUTER_API_KEY is required")
		}
		return KindHosted, nil
	case "":
		switch {
		case directConfigured && hostedConfigured:
			return KindNone, fmt.Errorf("LEX_DECISION_ADAPTER is required when both decision credentials are configured")
		case directConfigured:
			return KindDirect, nil
		case hostedConfigured:
			return KindHosted, nil
		default:
			return KindNone, nil
		}
	default:
		return KindNone, fmt.Errorf("LEX_DECISION_ADAPTER must be direct or hosted")
	}
}
