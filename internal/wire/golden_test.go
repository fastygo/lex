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
		{name: "validated-direct", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "9328322c5b6b447ed2a97da5f6db2762125381b5ba615aa182ddf1c22b32b0b3"},
		{name: "validated-hosted", adapter: "hosted-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "6426fb976432543b6a70fe05be0bc67dd7e1863ce74f93765f1deb5fd84ea076"},
		{name: "rejected", adapter: "direct-systemone", answers: goldenAnswers(0.1, 0.9, 0.1, 0.9, "reject"), verdict: "rejected", finding: "negative_result", hash: "0a2a35fa22a973c72f96f949630e7c8ab224318eacd5f8c8b942d334b0cfa17e"},
		{name: "insufficient", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.1, 0.1, 0.9, "proceed"), verdict: "insufficient", finding: "establishment_below_threshold", hash: "87a56aae08ccf42a68ef95ca2c7e0af673ba55f96cf1a5b733111b6830812d42"},
		{name: "conflict", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.9, 0.1, "proceed"), verdict: "conflict", finding: "evidence_conflict", hash: "c0292958eb00b862ddfdb349906621712b2e9959e42fc3e89bd9bef8bcfdcdaa"},
		{name: "manual-review", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "manual_review"), verdict: "manual_review", finding: "review_required", hash: "8809b2b35c63fde45a1a48d01e57d00e0d0cf8d442d98cf88b57f6fb3bbe685b"},
		{name: "error", adapter: "direct-systemone", answers: []byte(`{"support":{"type":"noul","noul":2}}`), verdict: "error", finding: "invalid_noul:support", hash: "4fad2913d57dd3e180e0da296e7aaa6febfb04c5b619e6c0cbc94b5eaff6b05c"},
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
