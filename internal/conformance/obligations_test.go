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
	{"State and QuestionSet MUST reach the adapter", []string{"TestDecisionPassesCallerStateAndQuestionsWithoutDomainInterpretation"}},
	{"Questions MUST be typed and atomic", []string{"TestDecisionRequestRejectsAmbiguousOrInvalidPrimitives", "TestDecisionRejectsMalformedQuestionSetBeforeProvider", "TestValidateAnswersAcceptsTypedAnswers"}},
	{"LeX MUST NOT derive a domain verdict", []string{"TestCoreHasNoDomainVerdictOrProfile", "TestMalformedAnswerIsStructuralFindingNotVerdict", "TestCapabilitiesDiscloseTheDecisionContract"}},
	{"verified success MUST remain distinct", []string{"TestNoExecutionOrRemovedRoutes", "TestMalformedBodiesNeverReachTheProvider", "TestSecurityProfileRejectsCookiesAndCORS"}},
	{"Context binding MUST be reproduced by Context", []string{"TestContextThatContextCannotReproduceNeverReachesTheProvider", "TestContextBindingIsVerifiedByContextRebuild"}},
	{"Context MUST NOT be merged into State", []string{"TestDecisionBindsOptionalContextWithoutMergingItIntoState"}},
	{"Runs MUST pin project", []string{"TestReplayRejectsUnpinnedModelAdapterAndVerifier", "TestSealedStateTamperingBreaksTheBundleHash", "TestDecisionBundleReplaysAllPrimitives"}},
	{"criteria change MUST produce", []string{"TestCriteriaChangeProducesNewQuestionSetHash"}},
	{"MUST NOT substitute for a resolved model", []string{"TestEvaluateRejectsAliasAndRemoteEndpoint", "TestHostedPinRejectsAliasesAndRequiresConfiguration", "TestEvaluateRejectsSubstitutedModel"}},
	{"MUST NOT rewrite the semantic answers", []string{"TestEvaluatePreservesRawAnswers", "TestCaseFoldedAnswerFieldIsRefusedNotMerged"}},
	{"Trace data MUST exclude credentials", []string{"TestEvaluateDropsResponsesThatEchoTheCredential", "TestProviderFailuresAreClassifiedByStage", "TestPanicBecomesProblemJSON", "TestLoadConfigRejectsAmbiguousTokensWithoutEcho"}},
	{"`metadata` MUST NOT reach the adapter", []string{"TestAdapterMetadataStaysOutOfTheReplayBundle", "TestDecisionPassesCallerStateAndQuestionsWithoutDomainInterpretation"}},
	{"structurally invalid answer MUST be reported", []string{"TestMalformedAnswersRemainReplayableStructuralFailures"}},
	{"An adapter MUST:", []string{"TestEvaluateDeclaresQuestionCapabilitiesBeforeDial", "TestEvaluatePreservesRawAnswers", "TestEvaluateRecordsResolvedHostedModel", "TestEvaluateKeepsProviderMetadataOnlyWhenPresent", "TestEvaluateClassifiesRetryableProviderStatus", "TestSelectRefusesSilentFallback", "TestDirectAdapterConformanceFixture", "TestHostedAdapterConformanceFixture"}},
	{"Replay MUST NOT contact a retrieval service", []string{"TestReplayDoesNotUseNetwork"}},
	{"saved pack MUST NOT be replaced", []string{"TestReplayLeavesTheSavedBundleUnchanged", "TestContextBindingIsVerifiedByContextRebuild"}},
	{"Replay MUST refuse a bundle whose project", []string{"TestReplayRefusesAnotherProject"}},
	{"The verifier MUST check", []string{"TestSealedStateTamperingBreaksTheBundleHash", "TestReplayRejectsUnpinnedModelAdapterAndVerifier", "TestValidateAnswersReportsMalformedAnswers", "TestContextBindingIsVerifiedByContextRebuild", "TestReplayRefusesAnotherProject"}},
	{"retryable provider failure MUST NOT be retried", []string{"TestEvaluateClassifiesRetryableProviderStatus", "TestProviderFailuresAreClassifiedByStage"}},
	{"MUST be refused with `response_budget`", []string{"TestResponseBudgetIsCheckedBeforeAndAfterTheProvider", "TestFullAnswerBudgetStillFitsTheResponse"}},
}

func TestNormativeMustLinesHaveTests(t *testing.T) {
	root := filepath.Join("..", "..")
	var lines []string
	for _, name := range []string{"protocol.md", "checks.md"} {
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
