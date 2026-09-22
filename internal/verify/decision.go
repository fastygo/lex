package verify

import (
	"fmt"
	"math"
	"sort"
)

// QuestionType is a provider-neutral typed-decision primitive.
type QuestionType string

const (
	QuestionNoul   QuestionType = "noul"
	QuestionChoice QuestionType = "choice"
	QuestionScore  QuestionType = "score"
)

// Question declares the answer type and declared answer domain.
type Question struct {
	Type    QuestionType
	Choices []string
	Levels  int
}

// Answer preserves one unmodified typed provider answer.
type Answer struct {
	Type          QuestionType
	Noul          *float64
	Choice        string
	Probabilities map[string]float64
	Score         *float64
}

// ValidateAnswers verifies completeness, answer types, and numeric domains.
func ValidateAnswers(questions map[string]Question, answers map[string]Answer) []Finding {
	findings := make([]Finding, 0)
	ids := make([]string, 0, len(questions))
	for id := range questions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		question := questions[id]
		answer, ok := answers[id]
		if !ok {
			findings = append(findings, errorFinding(CodePrefixMissingAnswer, id))
			continue
		}
		if answer.Type != question.Type {
			findings = append(findings, errorFinding(CodePrefixAnswerTypeMismatch, id))
			continue
		}
		switch question.Type {
		case QuestionNoul:
			if answer.Noul == nil || !probability(*answer.Noul) {
				findings = append(findings, errorFinding(CodePrefixInvalidNoul, id))
			}
		case QuestionChoice:
			if !contains(question.Choices, answer.Choice) || !distribution(question.Choices, answer.Probabilities) || !winningChoice(answer.Choice, answer.Probabilities) {
				findings = append(findings, errorFinding(CodePrefixInvalidChoice, id))
			}
		case QuestionScore:
			if question.Levels < 2 || answer.Score == nil || math.IsNaN(*answer.Score) || math.IsInf(*answer.Score, 0) || *answer.Score < 0 || *answer.Score > float64(question.Levels-1) {
				findings = append(findings, errorFinding(CodePrefixInvalidScore, id))
			}
		default:
			findings = append(findings, errorFinding(CodePrefixUnsupportedQuestion, id))
		}
	}
	extra := make([]string, 0)
	for id := range answers {
		if _, ok := questions[id]; !ok {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	for _, id := range extra {
		findings = append(findings, errorFinding(CodePrefixUnexpectedAnswer, id))
	}
	return findings
}

func errorFinding(code, questionID string) Finding {
	return Finding{Code: code + ":" + questionID, Detail: fmt.Sprintf("decision answer failed %s", code)}
}

func probability(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
}

func distribution(keys []string, values map[string]float64) bool {
	if len(values) != len(keys) {
		return false
	}
	total := 0.0
	for _, key := range keys {
		value, ok := values[key]
		if !ok || !probability(value) {
			return false
		}
		total += value
	}
	return math.Abs(total-1) <= 1e-9
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// Ties are valid: the selected option must be one of the exact maxima.
func winningChoice(choice string, probabilities map[string]float64) bool {
	selected, ok := probabilities[choice]
	if !ok {
		return false
	}
	for _, value := range probabilities {
		if value > selected {
			return false
		}
	}
	return true
}
