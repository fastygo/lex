package wire

import (
	"encoding/json"
	"strings"

	"github.com/fastygo/lex/internal/verify"
)

func parseAnswers(bundle map[string]any) (map[string]verify.Answer, []verify.Finding) {
	decision, _ := bundle["decision_set"].(map[string]any)
	raw, err := json.Marshal(decision["answers"])
	if err != nil {
		return nil, []verify.Finding{errorFinding(verify.CodeInvalidAnswers)}
	}
	var decoded map[string]map[string]json.RawMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return nil, []verify.Finding{errorFinding(verify.CodeInvalidAnswers)}
	}
	answers := make(map[string]verify.Answer, len(decoded))
	for id, fields := range decoded {
		var answer verify.Answer
		targets := map[string]any{
			"type": &answer.Type, "noul": &answer.Noul, "choice": &answer.Choice,
			"probabilities": &answer.Probabilities, "score": &answer.Score,
		}
		for key, value := range fields {
			for reserved := range targets {
				if key != reserved && strings.EqualFold(key, reserved) {
					return nil, []verify.Finding{errorFinding(verify.CodeInvalidAnswers)}
				}
			}
			if target, ok := targets[key]; ok {
				if err := json.Unmarshal(value, target); err != nil {
					return nil, []verify.Finding{errorFinding(verify.CodeInvalidAnswers)}
				}
			}
		}
		answers[id] = answer
	}
	return answers, nil
}
