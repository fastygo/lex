package verify

import (
	"os"
	"strings"
	"testing"
)

func TestKnownFindingAcceptsCatalog(t *testing.T) {
	for _, code := range FindingCodes() {
		if !KnownFinding(code) {
			t.Fatalf("stable code %s is not recognized", code)
		}
	}
	for _, prefix := range AnswerCodePrefixes() {
		if !KnownFinding(prefix+":support") || KnownFinding(prefix) || KnownFinding(prefix+":") {
			t.Fatalf("prefix %s was classified incorrectly", prefix)
		}
	}
	if KnownFinding("not-a-code") {
		t.Fatal("accepted an unknown finding code")
	}
}

func TestEmitSitesUseFindingConstants(t *testing.T) {
	for _, path := range []string{"decision.go", "../wire/context.go", "../wire/decision_replay.go", "../wire/answers.go"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), `Code: "`) {
			t.Fatalf("%s still uses a literal finding code", path)
		}
	}
}
