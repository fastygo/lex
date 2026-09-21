package wire

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
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

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	schemaValue, err := canonical.DecodeJSON([]byte(`{"type":"object","additionalProperties":false,"required":["type","title","status","detail","reason"],"properties":{"type":{"type":"string"},"title":{"type":"string"},"status":{"type":"integer"},"detail":{"type":"string"},"reason":{"type":"string"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := compiler.AddResource("problem.json", schemaValue); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("problem.json")
	if err != nil {
		t.Fatal(err)
	}
	components, _ := document["components"].(map[string]any)
	responseComponents, _ := components["responses"].(map[string]any)
	problem, _ := responseComponents["Problem"].(map[string]any)
	content, _ := problem["content"].(map[string]any)
	media, _ := content["application/problem+json"].(map[string]any)
	if err := schema.Validate(media["example"]); err != nil {
		t.Fatalf("problem example: %v", err)
	}
}
