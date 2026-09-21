package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMalformedDecisionsRemainReplayableErrors(t *testing.T) {
	cases := map[string]string{
		"nonwinning-choice":  strings.Replace(passingAnswers, `"proceed":1,"reject":0,"manual_review":0`, `"proceed":0,"reject":0,"manual_review":1`, 1),
		"case-folded-noul":   strings.ReplaceAll(passingAnswers, `"noul":`, `"NOUL":`),
		"shadowed-noul":      strings.Replace(passingAnswers, `"support":{"type":"noul",`, `"support":{"type":"noul","NOUL":1,`, 1),
		"missing-refutation": strings.Replace(passingAnswers, `"refuted":{"type":"noul","noul":0.1},`, "", 1),
	}
	for name, answers := range cases {
		t.Run(name, func(t *testing.T) {
			handler := mustHandlerWithDecider(t, &scriptedDecider{answers: []byte(answers)})
			rec := integrityRequest(handler, "/v1/evaluations", evaluationBody)
			if rec.Code != 422 {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body)
			}
			var response struct {
				ReplayBundle json.RawMessage `json:"replay_bundle"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			var bundle struct {
				Decision struct {
					Answers json.RawMessage `json:"answers"`
				} `json:"decision_set"`
			}
			if err := json.Unmarshal(response.ReplayBundle, &bundle); err != nil {
				t.Fatal(err)
			}
			var want, got any
			_ = json.Unmarshal([]byte(answers), &want)
			_ = json.Unmarshal(bundle.Decision.Answers, &got)
			wantJSON, _ := json.Marshal(want)
			gotJSON, _ := json.Marshal(got)
			if !bytes.Equal(wantJSON, gotJSON) {
				t.Fatal("raw answer values were changed")
			}
			replay := integrityRequest(handler, "/v1/replays", string(response.ReplayBundle))
			if replay.Code != 200 || !strings.Contains(replay.Body.String(), `"verdict":"error"`) {
				t.Fatalf("replay status=%d body=%s", replay.Code, replay.Body)
			}
		})
	}
}

func TestEvaluationRejectsInvalidUnicodeBeforeProvider(t *testing.T) {
	for _, invalid := range []string{`\ud800`, `\udc00`, `\ud800x`, string([]byte{0xff})} {
		decider := &scriptedDecider{answers: []byte(passingAnswers)}
		handler := mustHandlerWithDecider(t, decider)
		body := strings.Replace(evaluationBody, "The account is locked.", "The account is locked. "+invalid, 1)
		rec := integrityRequest(handler, "/v1/evaluations", body)
		if rec.Code != 400 || decider.calls != 0 {
			t.Fatalf("status=%d calls=%d", rec.Code, decider.calls)
		}
	}
}

func integrityRequest(handler http.Handler, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
