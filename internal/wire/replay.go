// Package wire validates versioned LeX wire messages.
package wire

import (
	"crypto/subtle"
	_ "embed"
	"fmt"
	"sync"

	"github.com/fastygo/lex/internal/canonical"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const replayBundleSchemaID = "https://lex.fastygo.dev/schema/v0.1/replay-bundle"

//go:embed schema/replay-bundle.schema.json
var replayBundleSchema []byte

var (
	replaySchemaOnce sync.Once
	replaySchema     *jsonschema.Schema
	replaySchemaErr  error
)

// ValidateReplayBundle rejects malformed, ambiguous, or unsupported replay bundles.
func ValidateReplayBundle(raw []byte) error {
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return fmt.Errorf("decode replay bundle: %w", err)
	}
	schema, err := compiledReplaySchema()
	if err != nil {
		return err
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("validate replay bundle: %w", err)
	}
	return nil
}

// VerifyReplayBundleHash validates a replay bundle and its self-hash exclusion.
func VerifyReplayBundleHash(raw []byte) error {
	if err := ValidateReplayBundle(raw); err != nil {
		return err
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		return fmt.Errorf("decode replay bundle: %w", err)
	}
	bundle, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("replay bundle must be a JSON object")
	}
	declared, ok := bundle["bundle_hash"].(string)
	if !ok {
		return fmt.Errorf("replay bundle hash must be a string")
	}
	delete(bundle, "bundle_hash")
	computed, err := canonical.HashValue(bundle)
	if err != nil {
		return fmt.Errorf("hash replay bundle: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(declared), []byte(computed)) != 1 {
		return fmt.Errorf("replay bundle hash does not match")
	}
	return nil
}

func compiledReplaySchema() (*jsonschema.Schema, error) {
	replaySchemaOnce.Do(func() {
		value, err := canonical.DecodeJSON(replayBundleSchema)
		if err != nil {
			replaySchemaErr = fmt.Errorf("decode embedded replay schema: %w", err)
			return
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		compiler.AssertFormat()
		if err := compiler.AddResource(replayBundleSchemaID, value); err != nil {
			replaySchemaErr = fmt.Errorf("register replay schema: %w", err)
			return
		}
		replaySchema, replaySchemaErr = compiler.Compile(replayBundleSchemaID)
		if replaySchemaErr != nil {
			replaySchemaErr = fmt.Errorf("compile replay schema: %w", replaySchemaErr)
		}
	})
	return replaySchema, replaySchemaErr
}
