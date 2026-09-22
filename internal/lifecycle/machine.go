// Package lifecycle is the request-scoped stage machine for one evaluation or replay.
// It records only the stages of the current request and keeps no server history.
package lifecycle

import "fmt"

// Kind selects the legal stage graph for one request.
type Kind string

const (
	// Evaluation is the live validation path.
	Evaluation Kind = "evaluation"
	// Decision is the generic typed-decision path.
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
	statusSkipped   = "skipped"
)

// Run accepts a complete stage sequence or reports the first illegal transition.
func Run(kind Kind, events []Event) error {
	switch kind {
	case Evaluation:
		return runEvaluation(events)
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

func runEvaluation(events []Event) error {
	if len(events) == 0 {
		return fmt.Errorf("illegal evaluation transition")
	}
	step := 0
	for _, event := range events {
		switch step {
		case 0:
			if event.Name != "receive" || event.Status != statusCompleted {
				return fmt.Errorf("illegal evaluation transition")
			}
			step = 1
		case 1:
			if event.Name != "pack" || (event.Status != statusCompleted && event.Status != statusFailed) {
				return fmt.Errorf("illegal evaluation transition")
			}
			if event.Status == statusFailed {
				step = 9
				continue
			}
			step = 2
		case 2:
			if event.Name != "decide" || (event.Status != statusCompleted && event.Status != statusFailed && event.Status != statusSkipped) {
				return fmt.Errorf("illegal evaluation transition")
			}
			switch event.Status {
			case statusFailed:
				step = 9
			case statusSkipped:
				step = 3
			default:
				step = 4
			}
		case 3:
			if event.Name != "verify" || event.Status != statusCompleted {
				return fmt.Errorf("illegal evaluation transition")
			}
			step = 9
		case 4:
			if event.Name != "verify" || (event.Status != statusCompleted && event.Status != statusFailed) {
				return fmt.Errorf("illegal evaluation transition")
			}
			step = 9
		default:
			return fmt.Errorf("illegal evaluation transition")
		}
	}
	if step != 3 && step != 9 {
		return fmt.Errorf("illegal evaluation transition")
	}
	return nil
}
