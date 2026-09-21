package wire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/fastygo/lex/internal/verify"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestOpenAPIMatchesSynchronousContract(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("schema", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatalf("openapi document is not strict JSON: %v", err)
	}
	document, ok := value.(map[string]any)
	if !ok || document["openapi"] != "3.1.1" {
		t.Fatalf("openapi version = %#v", document["openapi"])
	}
	paths, _ := document["paths"].(map[string]any)
	for _, path := range []string{"/healthz", "/v1/capabilities", "/v1/evaluations", "/v1/replays"} {
		if _, exists := paths[path]; !exists {
			t.Fatalf("missing path %s", path)
		}
	}
	evaluations, _ := paths["/v1/evaluations"].(map[string]any)
	post, _ := evaluations["post"].(map[string]any)
	responses, _ := post["responses"].(map[string]any)
	if _, exists := responses["202"]; exists {
		t.Fatal("evaluation declares an asynchronous 202 response")
	}
	if _, exists := responses["200"]; !exists {
		t.Fatal("evaluation lacks a synchronous 200 response")
	}

	if _, exists := responses["504"]; !exists {
		t.Fatal("evaluation lacks a deadline response")
	}
	if _, exists := responses["406"]; !exists {
		t.Fatal("evaluation lacks an Accept negotiation response")
	}

	components, _ := document["components"].(map[string]any)
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	if err := compiler.AddResource("https://lex.fastygo.dev/openapi.json", document); err != nil {
		t.Fatal(err)
	}
	if err := compiler.AddResource("https://lex.fastygo.dev/schema/v0.1/check-problem", map[string]any{
		"$ref": "https://lex.fastygo.dev/openapi.json#/components/schemas/Problem",
	}); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("https://lex.fastygo.dev/schema/v0.1/check-problem")
	if err != nil {
		t.Fatal(err)
	}
	responseComponents, _ := components["responses"].(map[string]any)
	problem, _ := responseComponents["Problem"].(map[string]any)
	content, _ := problem["content"].(map[string]any)
	media, _ := content["application/problem+json"].(map[string]any)
	if err := schema.Validate(media["example"]); err != nil {
		t.Fatalf("problem example: %v", err)
	}
	technical := map[string]any{
		"type": "https://lex.fastygo.dev/problems/decision_error", "title": "Unprocessable Entity",
		"status": jsonNumber(422), "detail": "typed answers failed deterministic checks", "reason": "decision_error",
		"verdict": "error", "findings": []any{},
	}
	if err := schema.Validate(technical); err != nil {
		t.Fatalf("technical verdict problem: %v", err)
	}
}

func TestFindingCodePatternMatchesOpenAPI(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("schema", "openapi.json"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	document := value.(map[string]any)
	components := document["components"].(map[string]any)
	schemas := components["schemas"].(map[string]any)
	finding := schemas["Finding"].(map[string]any)
	properties := finding["properties"].(map[string]any)
	code := properties["code"].(map[string]any)
	if code["pattern"] != verify.FindingCodePattern() {
		t.Fatalf("pattern = %s", verify.FindingCodePattern())
	}
}

func jsonNumber(value int) json.Number {
	return json.Number(strconv.Itoa(value))
}
