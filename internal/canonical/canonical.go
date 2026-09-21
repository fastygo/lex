// Package canonical implements the LeX RFC 8785 and SHA-256 hash profile.
package canonical

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	jsoncanonicalizer "github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

// Canonicalize validates one JSON object or array and returns its RFC 8785 form.
func Canonicalize(raw []byte) ([]byte, error) {
	if err := validateSingleJSON(raw); err != nil {
		return nil, err
	}
	canonical, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize JSON: %w", err)
	}
	return canonical, nil
}

// HashJSON returns the SHA-256 digest of a RFC 8785 canonical JSON value.
func HashJSON(raw []byte) (string, error) {
	canonical, err := Canonicalize(raw)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

// HashValue marshals an application value before applying the LeX hash profile.
func HashValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal JSON value: %w", err)
	}
	return HashJSON(raw)
}

// DecodeJSON validates one JSON object or array and decodes it without
// converting JSON numbers to binary floating-point values.
func DecodeJSON(raw []byte) (any, error) {
	if err := validateSingleJSON(raw); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func validateSingleJSON(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := validateValue(decoder, true); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return fmt.Errorf("read trailing JSON: %w", err)
	}
	return nil
}

func validateValue(decoder *json.Decoder, topLevel bool) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		if topLevel {
			return fmt.Errorf("top-level JSON value must be an object or array")
		}
		return nil
	}

	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if _, duplicate := seen[key]; duplicate {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = struct{}{}
			if err := validateValue(decoder, false); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := validateValue(decoder, false); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}

	end, err := decoder.Token()
	if err != nil {
		return err
	}
	expected := byte('}')
	if delimiter == '[' {
		expected = ']'
	}
	if end != json.Delim(expected) {
		return fmt.Errorf("expected JSON delimiter %q", expected)
	}
	return nil
}
