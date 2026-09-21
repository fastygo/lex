package conformance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// obligations maps each normative MUST line to the tests that exercise it.
// governance.md defines the keywords and is not a product obligation.
var obligations = []struct {
	needle string
	tests  []string
}{
	{"A decision MUST NOT be evidence", []string{"TestReplayRefusesInferenceOnlyEvidence"}},
	{"Evidence MUST be addressable", []string{"TestReplayRejectsSurfaceThatBreaksChecksum", "TestReplayRejectsPartialSourceSurface"}},
	{"Questions MUST be typed and atomic", []string{"TestReplayRejectsCompoundQuestionEvenWhenSelfHashMatches", "TestValidateAnswersAcceptsTypedAnswers", "TestQuestionsNameTheFrozenState"}},
	{"verified success MUST remain distinct", []string{"TestNoExecutionRoute", "TestEvaluationRejectsCallerPolicy", "TestInterpretKeepsConflictAheadOfInsufficient"}},
	{"MUST preserve `insufficient` and `conflict`", []string{"TestInterpretCoversEveryVerdict"}},
	{"Policy and thresholds MUST be external", []string{"TestEvaluationRejectsCallerPolicy"}},
	{"MUST require deterministic verification", []string{"TestInterpretAcceptsConsistentClaim"}},
	{"Runs MUST pin entity", []string{"TestReplayRejectsUnpinnedPolicyWithoutProviderCall", "TestReplayRejectsForeignAdapterVersion", "TestReplayRejectsFocusOutsideEmbeddedProfile", "TestReplayRejectsUnresolvedModelIdentity"}},
	{"verifier versions MUST also be recorded", []string{"TestReplayRejectsForeignVerifierVersion"}},
	{"criteria change MUST produce", []string{"TestCriteriaChangeProducesNewQuestionSetHash"}},
	{"MUST NOT substitute for a resolved model", []string{"TestReplayRejectsUnresolvedModelIdentity", "TestEvaluateRejectsSubstitutedModel"}},
	{"MUST NOT rewrite the semantic answers", []string{"TestEvaluatePreservesRawAnswers"}},
	{"Trace data MUST exclude credentials", []string{"TestProviderFailureDoesNotEchoSecrets", "TestProblemResponseOmitsBearerToken"}},
	{"It MUST NOT claim", []string{"TestNoExecutionRoute"}},
	{"validation verdict MUST NOT be treated as credentials", []string{"TestNoExecutionRoute", "TestSecurityProfileRejectsCookiesApprovalAndCORS"}},
	{"MUST NOT contact a retrieval service", []string{"TestReplayDoesNotUseNetwork"}},
	{"MUST compare a recomputation", []string{"TestReplayRejectsRewrittenPackChecksum"}},
	{"saved pack MUST NOT be replaced", []string{"TestReplayLeavesTheSavedBundleUnchanged"}},
	{"MUST refuse a replay whose entity project", []string{"TestReplayRefusesAnotherProject"}},
	{"The verifier MUST check", []string{"TestReplayRejectsPackHashThatDoesNotMatchContent", "TestValidateAnswersReportsMalformedAnswers", "TestReplayRejectsPartialSourceSurface", "TestReplayRefusesInferenceOnlyEvidence", "TestInterpretKeepsConflictAheadOfInsufficient", "TestInterpretRejectsEstablishedNegativeClaim", "TestSecurityProfileRejectsCookiesApprovalAndCORS"}},
	{"MUST preserve all material epistemic", []string{"TestInterpretKeepsSafetyGateBesideConflict"}},
	{"An adapter MUST:", []string{"TestEvaluationKeepsInjectedSourceTextOutOfQuestions", "TestEvaluateDeclaresQuestionCapabilitiesBeforeDial", "TestEvaluatePreservesRawAnswers", "TestReplayRejectsForeignAdapterVersion", "TestAdapterMetadataStaysOutOfTheReplayBundle", "TestEvaluateKeepsProviderMetadataOnlyWhenPresent", "TestRetryableProviderFailureIsNotADecisionError", "TestEvaluateClassifiesRetryableProviderStatus", "TestSelectRefusesSilentFallback", "TestEvaluateRejectsSubstitutedModel", "TestDirectAdapterConformanceFixture", "TestHostedAdapterConformanceFixture"}},
}

func TestNormativeMustLinesHaveTests(t *testing.T) {
	root := filepath.Join("..", "..")
	var lines []string
	for _, name := range []string{"protocol.md", "checks.md", "integration-stack.md"} {
		raw, err := os.ReadFile(filepath.Join(root, ".project", ".lex", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
			if strings.Contains(line, "MUST") {
				lines = append(lines, strings.TrimSpace(line))
			}
		}
	}
	for _, line := range lines {
		if !lineCovered(line) {
			t.Errorf("unmapped obligation: %s", line)
		}
	}
	names := testNames(t, root)
	for _, obligation := range obligations {
		matches := 0
		for _, line := range lines {
			if strings.Contains(line, obligation.needle) {
				matches++
			}
		}
		if matches != 1 {
			t.Errorf("needle %q matched %d MUST lines", obligation.needle, matches)
		}
		for _, name := range obligation.tests {
			if !names[name] {
				t.Errorf("missing test %s for %q", name, obligation.needle)
			}
		}
	}
}

func lineCovered(line string) bool {
	for _, obligation := range obligations {
		if strings.Contains(line, obligation.needle) {
			return true
		}
	}
	return false
}

func testNames(t *testing.T, root string) map[string]bool {
	t.Helper()
	names := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n") {
			line = strings.TrimSpace(line)
			if !strings.HasPrefix(line, "func Test") {
				continue
			}
			name := strings.TrimPrefix(line, "func ")
			if index := strings.IndexByte(name, '('); index > 0 {
				names[name[:index]] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}
