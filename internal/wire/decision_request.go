package wire

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"
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

// SchemaViolation locates the most specific schema failure as a JSON pointer
// and keyword. It never includes caller values.
func SchemaViolation(err error) (pointer, keyword string, ok bool) {
	var validation *jsonschema.ValidationError
	if !errors.As(err, &validation) {
		return "", "", false
	}
	leaf := deepestCause(validation)
	pointer = ""
	for _, token := range leaf.InstanceLocation {
		pointer += "/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(token)
	}
	if pointer == "" {
		pointer = "/"
	}
	if leaf.ErrorKind != nil {
		if path := leaf.ErrorKind.KeywordPath(); len(path) > 0 {
			keyword = path[len(path)-1]
		}
	}
	return pointer, keyword, true
}

func deepestCause(err *jsonschema.ValidationError) *jsonschema.ValidationError {
	if len(err.Causes) == 0 {
		return err
	}
	best := deepestCause(err.Causes[0])
	for _, cause := range err.Causes[1:] {
		if candidate := deepestCause(cause); len(candidate.InstanceLocation) > len(best.InstanceLocation) {
			best = candidate
		}
	}
	return best
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
