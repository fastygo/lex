package wire

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

//go:embed schema/openapi.json
var openAPIDocument []byte

const evaluationRequestSchemaID = "https://lex.fastygo.dev/schema/v0.1/evaluation-request"

var (
	evaluationSchemaOnce sync.Once
	evaluationSchema     *jsonschema.Schema
	evaluationSchemaErr  error
)

// ValidateEvaluationRequest checks one evaluation body against the published schema.
func ValidateEvaluationRequest(raw []byte) error {
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return err
	}
	schema, err := compiledEvaluationSchema()
	if err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("evaluation request: %w", err)
	}
	return nil
}

func compiledEvaluationSchema() (*jsonschema.Schema, error) {
	evaluationSchemaOnce.Do(func() {
		document, err := canonical.DecodeJSON(openAPIDocument)
		if err != nil {
			evaluationSchemaErr = fmt.Errorf("decode OpenAPI document: %w", err)
			return
		}
		root, _ := document.(map[string]any)
		components, _ := root["components"].(map[string]any)
		schemas, _ := components["schemas"].(map[string]any)
		schema, ok := schemas["EvaluationRequest"]
		if !ok {
			evaluationSchemaErr = fmt.Errorf("OpenAPI document has no evaluation request schema")
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		if err := compiler.AddResource(evaluationRequestSchemaID, schema); err != nil {
			evaluationSchemaErr = fmt.Errorf("register evaluation schema: %w", err)
			return
		}
		evaluationSchema, evaluationSchemaErr = compiler.Compile(evaluationRequestSchemaID)
	})
	return evaluationSchema, evaluationSchemaErr
}
