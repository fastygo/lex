// Package lifecycle is the request-scoped stage machine for one decision or replay.
// It records only the stages of the current request and keeps no server history.
package lifecycle

import "fmt"

// Kind selects the legal stage graph for one request.
type Kind string

const (
	// Decision is the typed-decision path: receive -> decide -> verify.
	Decision Kind = "decision"
	// Replay is the caller-supplied bundle path.
	Replay Kind = "replay"
)

// Event is one recorded stage outcome.
type Event struct {
	Name   string
	Status string
}

const (
	statusCompleted = "completed"
	statusFailed    = "failed"
)

// Run accepts a complete stage sequence or reports the first illegal transition.
func Run(kind Kind, events []Event) error {
	switch kind {
	case Decision:
		return runDecision(events)
	case Replay:
		return runReplay(events)
	default:
		return fmt.Errorf("unknown lifecycle %q", kind)
	}
}

func runDecision(events []Event) error {
	if len(events) < 2 || len(events) > 3 || events[0].Name != "receive" || events[0].Status != statusCompleted {
		return fmt.Errorf("illegal decision transition")
	}
	if events[1].Name != "decide" || (events[1].Status != statusCompleted && events[1].Status != statusFailed) {
		return fmt.Errorf("illegal decision transition")
	}
	if events[1].Status == statusFailed {
		if len(events) == 2 {
			return nil
		}
		return fmt.Errorf("illegal decision transition")
	}
	if len(events) == 3 && events[2].Name == "verify" && (events[2].Status == statusCompleted || events[2].Status == statusFailed) {
		return nil
	}
	return fmt.Errorf("illegal decision transition")
}

func runReplay(events []Event) error {
	if len(events) != 1 || events[0].Name != "replay" || (events[0].Status != statusCompleted && events[0].Status != statusFailed) {
		return fmt.Errorf("illegal replay transition")
	}
	return nil
}
