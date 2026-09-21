package wire

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

func TestValidateReplayBundleAcceptsValidBundle(t *testing.T) {
	if err := ValidateReplayBundle([]byte(validReplayBundle)); err != nil {
		t.Fatalf("ValidateReplayBundle() error = %v", err)
	}
}

func TestValidateReplayBundleRejectsUnknownTopLevelField(t *testing.T) {
	raw := strings.TrimSpace(validReplayBundle)
	raw = strings.TrimSuffix(raw, "}") + `,"unexpected":true}`
	if err := ValidateReplayBundle([]byte(raw)); err == nil {
		t.Fatal("ValidateReplayBundle() accepted an unknown top-level field")
	}
}

func TestValidateReplayBundleRejectsDuplicateKey(t *testing.T) {
	raw := strings.Replace(validReplayBundle, `"protocol_version":"0.1-draft"`, `"protocol_version":"0.1-draft","protocol_version":"0.1-draft"`, 1)
	if err := ValidateReplayBundle([]byte(raw)); err == nil {
		t.Fatal("ValidateReplayBundle() accepted a duplicate key")
	}
}

func TestVerifyReplayBundleHash(t *testing.T) {
	raw := hashedReplayBundle(t)
	if err := VerifyReplayBundleHash(raw); err != nil {
		t.Fatalf("VerifyReplayBundleHash() error = %v", err)
	}
}

func TestVerifyReplayBundleHashRejectsTampering(t *testing.T) {
	raw := strings.Replace(string(hashedReplayBundle(t)), `"entity-1"`, `"entity-2"`, 1)
	if err := VerifyReplayBundleHash([]byte(raw)); err == nil {
		t.Fatal("VerifyReplayBundleHash() accepted altered bundle")
	}
}

func hashedReplayBundle(t *testing.T) []byte {
	t.Helper()
	value, err := canonical.DecodeJSON([]byte(validReplayBundle))
	if err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	bundle := value.(map[string]any)
	delete(bundle, "bundle_hash")
	hash, err := canonical.HashValue(bundle)
	if err != nil {
		t.Fatalf("HashValue() error = %v", err)
	}
	bundle["bundle_hash"] = hash
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return raw
}

const validReplayBundle = `{
  "protocol_version":"0.1-draft",
  "verifier_version":"v0.1.0",
  "entity":{
    "id":"entity-1",
    "project_id":"project-1",
    "type":"claim",
    "schema_version":"v1",
    "version":"v1",
    "checksum":"0000000000000000000000000000000000000000000000000000000000000000"
  },
  "context":{
    "pack":{},
    "snapshot":{},
    "pack_request":{},
    "pack_hash":"0000000000000000000000000000000000000000000000000000000000000000"
  },
  "question_set":{
    "id":"questions-1",
    "version":"v1",
    "hash":"0000000000000000000000000000000000000000000000000000000000000000",
    "questions":{
      "support":{
        "type":"noul",
        "instructions":"Is the claim supported?"
      }
    }
  },
  "policy":{
    "id":"policy-1",
    "version":"v1",
    "hash":"0000000000000000000000000000000000000000000000000000000000000000"
  },
  "decision_set":{
    "context_pack_hash":"0000000000000000000000000000000000000000000000000000000000000000",
    "question_set_hash":"0000000000000000000000000000000000000000000000000000000000000000",
    "policy_hash":"0000000000000000000000000000000000000000000000000000000000000000",
    "adapter_id":"fixture",
    "adapter_version":"v1",
    "resolved_model":"fixture-v1",
    "answers":{"support":{"type":"noul","noul":0.9}}
  },
  "bundle_hash":"0000000000000000000000000000000000000000000000000000000000000000"
}`
