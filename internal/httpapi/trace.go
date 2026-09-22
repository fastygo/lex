package httpapi

import "github.com/fastygo/lex/internal/lifecycle"

// traceStage is one stage outcome as disclosed on the wire.
type traceStage struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// stages records the stage outcomes of one request in order. Every rendering
// is checked against the lifecycle machine, so an illegal sequence is a
// programming error surfaced as a panic, never a plausible-looking trace.
type stages struct {
	kind   lifecycle.Kind
	events []lifecycle.Event
}

func evaluationStages() *stages {
	return &stages{kind: lifecycle.Evaluation, events: []lifecycle.Event{{Name: "receive", Status: "completed"}}}
}

func (s *stages) completed(name string) *stages { return s.mark(name, "completed") }
func (s *stages) failed(name string) *stages    { return s.mark(name, "failed") }
func (s *stages) skipped(name string) *stages   { return s.mark(name, "skipped") }

func (s *stages) mark(name, status string) *stages {
	s.events = append(s.events, lifecycle.Event{Name: name, Status: status})
	return s
}

// ending returns the recorded stages plus one terminal outcome without
// mutating the recorder, for error sites that end the request.
func (s *stages) ending(name, status string) []traceStage {
	events := append(append([]lifecycle.Event(nil), s.events...), lifecycle.Event{Name: name, Status: status})
	return render(s.kind, events)
}

func (s *stages) view() []traceStage { return render(s.kind, s.events) }

func render(kind lifecycle.Kind, events []lifecycle.Event) []traceStage {
	if err := lifecycle.Run(kind, events); err != nil {
		panic(err)
	}
	view := make([]traceStage, 0, len(events))
	for _, event := range events {
		view = append(view, traceStage{Name: event.Name, Status: event.Status})
	}
	return view
}

func replayTrace() []traceStage { return replayTraceStatus("completed") }

func replayTraceStatus(status string) []traceStage {
	return render(lifecycle.Replay, []lifecycle.Event{{Name: "replay", Status: status}})
}
