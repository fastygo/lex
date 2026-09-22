package wire

import (
	_ "embed"
	"fmt"
	"sync"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const decisionRequestSchemaID = "https://lex.fastygo.dev/schema/v0.2/decision-request"

//go:embed schema/decision-request.schema.json
var decisionRequestSchema []byte

var (
	decisionRequestSchemaOnce sync.Once
	compiledDecisionRequest   *jsonschema.Schema
	decisionRequestSchemaErr  error
)

// ValidateDecisionRequest checks one generic decision body against the
// published schema. Semantic primitive checks run after strict decoding.
func ValidateDecisionRequest(raw []byte) error {
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return err
	}
	schema, err := compiledGenericDecisionRequest()
	if err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("decision request: %w", err)
	}
	return nil
}

func compiledGenericDecisionRequest() (*jsonschema.Schema, error) {
	decisionRequestSchemaOnce.Do(func() {
		value, err := canonical.DecodeJSON(decisionRequestSchema)
		if err != nil {
			decisionRequestSchemaErr = fmt.Errorf("decode decision request schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		if err := compiler.AddResource(decisionRequestSchemaID, value); err != nil {
			decisionRequestSchemaErr = fmt.Errorf("register decision request schema: %w", err)
			return
		}
		compiledDecisionRequest, decisionRequestSchemaErr = compiler.Compile(decisionRequestSchemaID)
	})
	return compiledDecisionRequest, decisionRequestSchemaErr
}
