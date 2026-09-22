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
	"github.com/fastygo/lex/internal/profile/claimvalidation"
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
	frozen := addressableFrozen(t)
	if tc.inference {
		frozen = inferenceOnlyFrozen()
	}
	raw, err := BuildBundle(claimvalidation.Profile(), BundleInput{
		Entity:         testEntity(),
		Frozen:         frozen,
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
		{name: "validated-direct", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "642945216b90f7e66af3248b9ad2b18407b3027b96ccfd9b5efb7650ff752bb9"},
		{name: "validated-hosted", adapter: "hosted-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "54d0792d6423b3f2910f17b3d9b4e3c4ab24b5b4ccfc3997c6691017850a0f06"},
		{name: "rejected", adapter: "direct-systemone", answers: goldenRefutationAnswers(), verdict: "rejected", finding: "negative_result", hash: "8b3389ae8d6386885048fce7e54e6dae2c6562f3b3991aa30ed40f447a530203"},
		{name: "insufficient", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.1, 0.1, 0.9, "proceed"), verdict: "insufficient", finding: "establishment_below_threshold", hash: "040e8e57f0be45b280bb3f3f63fc41a0a87203cec23bc3a5238de8918d46edc0"},
		{name: "conflict", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.9, 0.1, "proceed"), verdict: "conflict", finding: "evidence_conflict", hash: "3e0218afd4aa18543fc8058bc902941f1d121c4a7e7fe97e8d0e88c16fc2136a"},
		{name: "manual-review", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "manual_review"), verdict: "manual_review", finding: "review_required", hash: "ca121ae05834f275aeff5c2823ad6e8a2b9b2671eb21d31d6e3f9d4a2634598f"},
		{name: "error", adapter: "direct-systemone", answers: []byte(`{"support":{"type":"noul","noul":2}}`), verdict: "error", finding: "invalid_noul:support", hash: "dd41cfb29594f2f7ce4aaf4ff40bd95315bfc6af2f3a7b0751a12e48514dd970"},
		{name: "inference-only", adapter: "direct-systemone", inference: true, answers: goldenAnswers(0.99, 0.99, 0.01, 0.99, "proceed"), verdict: "insufficient", finding: "inference_only", hash: "587e37f851a1efac2a3a073bfdb59d4cdca1f025db2fe1a0e58b6794b2f4ee4f"},
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
