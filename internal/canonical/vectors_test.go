package canonical

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
)

type vectorFile struct {
	Producer string   `json:"producer"`
	Vectors  []vector `json:"vectors"`
}

type vector struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Input     string `json:"input"`
	Canonical string `json:"canonical"`
	SHA256    string `json:"sha256"`
}

func TestIndependentVectorFile(t *testing.T) {
	raw, err := os.ReadFile("testdata/jcs-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file vectorFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	if file.Producer != "scripts/jcs_vectors.py" || len(file.Vectors) < 16 {
		t.Fatalf("producer = %s vectors = %d", file.Producer, len(file.Vectors))
	}
	byName := make(map[string]vector, len(file.Vectors))
	for _, item := range file.Vectors {
		switch item.Kind {
		case "jcs":
			form, err := Canonicalize([]byte(item.Input))
			if err != nil {
				t.Fatalf("%s: %v", item.Name, err)
			}
			if string(form) != item.Canonical {
				t.Fatalf("%s canonical = %s, want %s", item.Name, form, item.Canonical)
			}
			sum := sha256.Sum256([]byte(item.Canonical))
			if hex.EncodeToString(sum[:]) != item.SHA256 {
				t.Fatalf("%s published digest does not match SHA-256 of the canonical bytes", item.Name)
			}
			hash, err := HashJSON([]byte(item.Input))
			if err != nil {
				t.Fatal(err)
			}
			if hash != item.SHA256 {
				t.Fatalf("%s hash = %s, want %s", item.Name, hash, item.SHA256)
			}
		case "source_bytes":
			sum := sha256.Sum256([]byte(item.Input))
			if hex.EncodeToString(sum[:]) != item.SHA256 {
				t.Fatalf("%s source digest = %s, want %s", item.Name, hex.EncodeToString(sum[:]), item.SHA256)
			}
		case "reject":
			if _, err := Canonicalize([]byte(item.Input)); err == nil {
				t.Fatalf("%s was accepted", item.Name)
			}
		default:
			t.Fatalf("%s has kind %s", item.Name, item.Kind)
		}
		byName[item.Name] = item
	}
	if byName["source-bytes"].SHA256 == byName["source-bytes-one-edit"].SHA256 {
		t.Fatal("a one-byte source edit kept the same digest")
	}
	if byName["unicode-cafe"].SHA256 != byName["unicode-cafe-escaped"].SHA256 {
		t.Fatal("equivalent Unicode inputs produced different digests")
	}
	if byName["without-self-hash"].SHA256 == byName["with-self-hash"].SHA256 {
		t.Fatal("including a self-hash field left the digest unchanged")
	}
}
