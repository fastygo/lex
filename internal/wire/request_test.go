package wire

import "testing"

func TestEvaluationRequestSchemaRejectsCallerPolicy(t *testing.T) {
	const valid = `{"project_id":"project-test","entity":{"id":"claim-1","type":"claim","schema_version":"0.1","version":"1"},"query":"account","sources":[{"id":"source-1","version":"v1","text":"The account is locked."}]}`
	if err := ValidateEvaluationRequest([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	const policy = `{"project_id":"project-test","entity":{"id":"claim-1","type":"claim","schema_version":"0.1","version":"1"},"query":"account","policy":{"support_min":0},"sources":[{"id":"source-1","version":"v1","text":"The account is locked."}]}`
	if err := ValidateEvaluationRequest([]byte(policy)); err == nil {
		t.Fatal("schema accepted a caller policy")
	}
}
