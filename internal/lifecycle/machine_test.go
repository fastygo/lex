package lifecycle

import "testing"

func TestRunAcceptsLegalEvaluationSequences(t *testing.T) {
	sequences := [][]Event{
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "failed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "skipped"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "skipped"}, {Name: "verify", Status: "completed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "failed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "completed"}, {Name: "verify", Status: "failed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "completed"}, {Name: "verify", Status: "completed"}},
	}
	for _, sequence := range sequences {
		if err := Run(Evaluation, sequence); err != nil {
			t.Fatalf("Run(%v) error = %v", sequence, err)
		}
	}
}

func TestRunRejectsIllegalEvaluationSequences(t *testing.T) {
	sequences := [][]Event{
		nil,
		{{Name: "decide", Status: "completed"}},
		{{Name: "receive", Status: "failed"}},
		{{Name: "receive", Status: "completed"}, {Name: "decide", Status: "completed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "failed"}, {Name: "decide", Status: "completed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "failed"}, {Name: "verify", Status: "completed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "skipped"}, {Name: "verify", Status: "failed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "completed"}},
		{{Name: "receive", Status: "completed"}, {Name: "receive", Status: "completed"}},
		{{Name: "replay", Status: "completed"}},
		{{Name: "receive", Status: "completed"}, {Name: "pack", Status: "completed"}, {Name: "decide", Status: "completed"}, {Name: "verify", Status: "completed"}, {Name: "verify", Status: "completed"}},
	}
	for _, sequence := range sequences {
		if err := Run(Evaluation, sequence); err == nil {
			t.Fatalf("Run(%v) accepted an illegal sequence", sequence)
		}
	}
}

func TestRunReplayAllowsOnlyCompletedReplay(t *testing.T) {
	if err := Run(Replay, []Event{{Name: "replay", Status: "completed"}}); err != nil {
		t.Fatal(err)
	}
	illegal := [][]Event{
		{{Name: "receive", Status: "completed"}},
		{{Name: "replay", Status: "failed"}},
		{{Name: "replay", Status: "completed"}, {Name: "decide", Status: "completed"}},
	}
	for _, sequence := range illegal {
		if err := Run(Replay, sequence); err == nil {
			t.Fatalf("Run(%v) accepted an illegal replay", sequence)
		}
	}
}
