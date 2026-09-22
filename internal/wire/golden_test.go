package wire

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/adapters/openrouter"
	"github.com/fastygo/lex/internal/adapters/typesafe"
	"github.com/fastygo/lex/internal/canonical"
)

func TestGoldenReplayAgreesOnVerdictAndHash(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("golden replay dialed the network")
		return nil, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })

	for _, tc := range goldenCases() {
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
		ResolvedModel:  goldenModel(tc.adapter),
		Answers:        tc.answers,
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func goldenModel(adapter string) string {
	if adapter == "hosted-systemone" {
		return openrouter.Model + "-20260901"
	}
	return typesafe.Model
}

func goldenAnswers(support, established, conflict, safety float64, choice string) []byte {
	return []byte(fmt.Sprintf(`{"support":{"type":"noul","noul":%g},"refuted":{"type":"noul","noul":0.1},"established":{"type":"noul","noul":%g},"conflict":{"type":"noul","noul":%g},"safe_to_auto_act":{"type":"noul","noul":%g},"action":{"type":"choice","choice":%q,"probabilities":{"proceed":%g,"reject":%g,"manual_review":%g,"other":%g}}}`,
		support, established, conflict, safety, choice,
		choiceWeight(choice, "proceed"), choiceWeight(choice, "reject"), choiceWeight(choice, "manual_review"), choiceWeight(choice, "other")))
}

func goldenRefutationAnswers() []byte {
	return bytes.Replace(goldenAnswers(0.1, 0.1, 0.1, 0.9, "reject"), []byte(`"refuted":{"type":"noul","noul":0.1}`), []byte(`"refuted":{"type":"noul","noul":0.9}`), 1)
}

func choiceWeight(choice, name string) float64 {
	if choice == name {
		return 1
	}
	return 0
}

func TestPythonConsumerAgreesOnGoldenVerdictsAndHashes(t *testing.T) {
	for _, tc := range goldenCases() {
		t.Run(tc.name, func(t *testing.T) {
			raw := goldenBundle(t, tc)
			if got := pythonLine(t, raw, "verdict_gate.py"); got != tc.verdict {
				t.Fatalf("python verdict = %s, want %s", got, tc.verdict)
			}
			pythonHash := pythonLine(t, raw, "jcs_vectors.py", "--bundle")
			value, err := canonical.DecodeJSON(raw)
			if err != nil {
				t.Fatal(err)
			}
			bundle := value.(map[string]any)
			declared, _ := bundle["bundle_hash"].(string)
			delete(bundle, "bundle_hash")
			goHash, err := canonical.HashValue(bundle)
			if err != nil {
				t.Fatal(err)
			}
			if pythonHash != goHash || pythonHash != declared {
				t.Fatalf("python = %s go = %s declared = %s", pythonHash, goHash, declared)
			}
		})
	}
}

func goldenCases() []goldenCase {
	return []goldenCase{
		{name: "validated-direct", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "0bbc4dd911b3cb85e0dcec38b3dae638470cd81b82eb0415efc2001cb40f07a3"},
		{name: "validated-hosted", adapter: "hosted-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "813332b8666234dc4b49fd7251410c5ff625b345c8815bda5580b863147a1630"},
		{name: "rejected", adapter: "direct-systemone", answers: goldenRefutationAnswers(), verdict: "rejected", finding: "negative_result", hash: "041186fd201cc1c68c489f2b7d72b9f7ad1fd59538996570675adc61c79efbe3"},
		{name: "insufficient", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.1, 0.1, 0.9, "proceed"), verdict: "insufficient", finding: "establishment_below_threshold", hash: "dbc534a407b712f5a5f644a85cf2410977a76d0822361953b7f4b79275e4505d"},
		{name: "conflict", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.9, 0.1, "proceed"), verdict: "conflict", finding: "evidence_conflict", hash: "0ceca6b9b99a2392cefa8a4ee114159d7eb6064131817272292809d2253065df"},
		{name: "manual-review", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "manual_review"), verdict: "manual_review", finding: "review_required", hash: "b88bac4cde8041f681aaf65b511eba49b92b20fe51cdf1697c30963bb4faddf4"},
		{name: "error", adapter: "direct-systemone", answers: []byte(`{"support":{"type":"noul","noul":2}}`), verdict: "error", finding: "invalid_noul:support", hash: "3b3e313c52d55fe12c9222ed7ffdfc30a6106a0d074ed5685221c5ee3af905f3"},
		{name: "inference-only", adapter: "direct-systemone", inference: true, answers: goldenAnswers(0.99, 0.99, 0.01, 0.99, "proceed"), verdict: "insufficient", finding: "inference_only", hash: "349f0f744a1138d088419d858123c2cff4e81762a649dd7c82370f3e89141a2c"},
	}
}

func pythonLine(t *testing.T, raw []byte, script string, args ...string) string {
	t.Helper()
	command := exec.Command("python", append([]string{filepath.Join("..", "..", "scripts", script)}, args...)...)
	command.Stdin = bytes.NewReader(raw)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("python %s: %v\n%s", script, err, output)
	}
	return strings.TrimSpace(string(output))
}
