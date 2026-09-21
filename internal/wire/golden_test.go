package wire

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

func TestGoldenReplayAgreesOnVerdictAndHash(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("golden replay dialed the network")
		return nil, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })

	cases := []goldenCase{
		{name: "validated-direct", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "4815218466f23187c6f528a99fc89827d5486628d4fd923614a6c1eb45f78774"},
		{name: "validated-hosted", adapter: "hosted-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "382c4860aaec5321c4d4ac9ab3715782a03de513c3b03adffe8482ee80964e5a"},
		{name: "rejected", adapter: "direct-systemone", answers: goldenAnswers(0.1, 0.9, 0.1, 0.9, "reject"), verdict: "rejected", finding: "negative_result", hash: "620f52c60247e1893fddf132638755df68eaed01168a3372e79e1c383969fed0"},
		{name: "insufficient", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.1, 0.1, 0.9, "proceed"), verdict: "insufficient", finding: "establishment_below_threshold", hash: "3a181a517513d2db40221a9879ce257d974e541ce312552733e96ed611ce4c27"},
		{name: "conflict", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.9, 0.1, "proceed"), verdict: "conflict", finding: "evidence_conflict", hash: "b96bc43f893f4b1b6a69beecf0abb027365f5ae4c914f3c073f62ed705bf9859"},
		{name: "manual-review", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "manual_review"), verdict: "manual_review", finding: "review_required", hash: "a877785d0c774744e3cb0b531621d9c378fbac1d6a7f91b036ce11fcd55d000b"},
		{name: "error", adapter: "direct-systemone", answers: []byte(`{"support":{"type":"noul","noul":2}}`), verdict: "error", finding: "invalid_noul:support", hash: "5c6e612087c29e42043c22183e5738798fa7ef7266d669c2e9c7728d8f1e8d43"},
		{name: "inference-only", adapter: "direct-systemone", inference: true, answers: goldenAnswers(0.99, 0.99, 0.01, 0.99, "proceed"), verdict: "insufficient", finding: "inference_only", hash: "705b434cc1375a44c6ca88c3ae0f9f0a23de855d1e2a2fc0b3e4db0c2a7c56bb"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw := goldenBundle(t, tc)
			report, err := Replay(raw)
			if err != nil {
				t.Fatal(err)
			}
			if string(report.Verdict) != tc.verdict || (tc.finding != "" && !hasFinding(report, tc.finding)) {
				t.Fatalf("verdict = %s findings = %#v", report.Verdict, report.Findings)
			}
			hash, err := canonical.HashJSON(raw)
			if err != nil {
				t.Fatal(err)
			}
			if hash != tc.hash {
				t.Fatalf("bundle hash = %s", hash)
			}
		})
	}
}

type goldenCase struct {
	name      string
	adapter   string
	answers   []byte
	verdict   string
	finding   string
	hash      string
	inference bool
}

func goldenBundle(t *testing.T, tc goldenCase) []byte {
	t.Helper()
	pack := addressablePack()
	snapshot := any(addressableSnapshot())
	request := any(addressableRequest())
	if tc.inference {
		pack = []byte(`{"evidence_items":[{"id":"chunk_0000","class":"model_inference","trust_level":"project","surface":"The model says the claim is true."}]}`)
		snapshot = map[string]any{"id": "snapshot-1"}
		request = map[string]any{"query": "claim"}
	}
	raw, err := BuildBundle(BundleInput{
		Entity:         Entity{ID: "claim-1", ProjectID: "project-test", Type: "claim", SchemaVersion: "0.1", Version: "1"},
		Pack:           pack,
		Snapshot:       snapshot,
		PackRequest:    request,
		AdapterID:      tc.adapter,
		AdapterVersion: "0.1.0",
		ResolvedModel:  "fixture-v1",
		Answers:        tc.answers,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func goldenAnswers(support, established, conflict, safety float64, choice string) []byte {
	return []byte(fmt.Sprintf(`{"support":{"type":"noul","noul":%g},"established":{"type":"noul","noul":%g},"conflict":{"type":"noul","noul":%g},"safe_to_auto_act":{"type":"noul","noul":%g},"action":{"type":"choice","choice":%q,"probabilities":{"proceed":%g,"reject":%g,"manual_review":%g,"other":%g}}}`,
		support, established, conflict, safety, choice,
		choiceWeight(choice, "proceed"), choiceWeight(choice, "reject"), choiceWeight(choice, "manual_review"), choiceWeight(choice, "other")))
}

func choiceWeight(choice, name string) float64 {
	if choice == name {
		return 1
	}
	return 0
}
