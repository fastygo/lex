package wire

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/profile"
)

func TestGoldenReplayAgreesOnVerdictAndHash(t *testing.T) {
	previous := http.DefaultTransport
	http.DefaultTransport = roundTripper(func(*http.Request) (*http.Response, error) {
		t.Fatal("golden replay dialed the network")
		return nil, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })

	cases := []goldenCase{
		{name: "validated-direct", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "a16adca26c13d7ddf23dce68564255d25c03d32a04ca6a174ccc6a879f392743"},
		{name: "validated-hosted", adapter: "hosted-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"), verdict: "validated", hash: "794ca4065ef98b45febb49645fe6557269d44063b45a462e237055d9bbf2e15f"},
		{name: "rejected", adapter: "direct-systemone", answers: goldenAnswers(0.1, 0.9, 0.1, 0.9, "reject"), verdict: "rejected", finding: "negative_result", hash: "aeb277fa59acd8ac118a5f5a1b1876546b826332147735707c2e975fce699cb3"},
		{name: "insufficient", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.1, 0.1, 0.9, "proceed"), verdict: "insufficient", finding: "establishment_below_threshold", hash: "1b7ad52d8bfbc5aa052961c7fdca35a2de32fcf0bb499738c3712bda6fc4e6ea"},
		{name: "conflict", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.9, 0.1, "proceed"), verdict: "conflict", finding: "evidence_conflict", hash: "8873ba2c22fff4a739227dd5dae91d49e5281c10608abb3e6a3d52414db16af2"},
		{name: "manual-review", adapter: "direct-systemone", answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "manual_review"), verdict: "manual_review", finding: "review_required", hash: "1739bf74c97c9398b66b722372d9c7c4fc51de35edd3cfb08be8b88565b6b1c9"},
		{name: "error", adapter: "direct-systemone", answers: []byte(`{"support":{"type":"noul","noul":2}}`), verdict: "error", finding: "invalid_noul:support", hash: "71ad1d941125c76741e786ebcbd9a8546ade53c22a56277738c9606b1b520b1c"},
		{name: "inference-only", adapter: "direct-systemone", inference: true, answers: goldenAnswers(0.99, 0.99, 0.01, 0.99, "proceed"), verdict: "insufficient", finding: "inference_only", hash: "050d81aeeafe347c272874993e10d26b34f4a3b168369a70cd6615d518f25d32"},
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
		return profile.HostedModel + "-20260901"
	}
	return profile.DirectModel
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

func TestPythonAgreesOnValidatedBundleSelfHash(t *testing.T) {
	raw := goldenBundle(t, goldenCase{
		adapter: "direct-systemone",
		answers: goldenAnswers(0.9, 0.9, 0.1, 0.9, "proceed"),
	})
	script := filepath.Join("..", "..", "scripts", "jcs_vectors.py")
	command := exec.Command("python", script, "--bundle")
	command.Stdin = bytes.NewReader(raw)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("python canonicalizer: %v", err)
	}
	pythonHash := strings.TrimSpace(string(output))
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
}
