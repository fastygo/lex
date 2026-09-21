package canonical

import (
	"strings"
	"testing"
)

func TestCanonicalizeSortsObjectKeys(t *testing.T) {
	got, err := Canonicalize([]byte(`{"z":1,"a":[true,null]}`))
	if err != nil {
		t.Fatalf("Canonicalize() error = %v", err)
	}
	const want = `{"a":[true,null],"z":1}`
	if string(got) != want {
		t.Fatalf("canonical JSON = %s, want %s", got, want)
	}
}

func TestHashJSONIgnoresObjectKeyOrder(t *testing.T) {
	first, err := HashJSON([]byte(`{"b":2,"a":1}`))
	if err != nil {
		t.Fatalf("HashJSON(first) error = %v", err)
	}
	second, err := HashJSON([]byte(`{"a":1,"b":2}`))
	if err != nil {
		t.Fatalf("HashJSON(second) error = %v", err)
	}
	if first != second {
		t.Fatalf("hashes differ: %s != %s", first, second)
	}
	const independent = "43258cff783fe7036d8a43033f830adfc60ec037382473548ac742b888292777"
	if first != independent {
		t.Fatalf("hash = %s, want independent digest %s", first, independent)
	}
}

func TestHashJSONMatchesIndependentUnicodeDigests(t *testing.T) {
	vectors := []struct {
		raw  string
		hash string
	}{
		{raw: "{\"cafe\":\"caf\u00e9\"}", hash: "801a86b42bae9df69aecb1337a75a6988bcb5d6c0b54bb54c4425e3dc1514369"},
		{raw: "{\"name\":\"\u20ac\"}", hash: "080466493ecc711eb2010d0339912c06fcc8d7921c380aecbfb4f0b86ed18b69"},
	}
	for _, vector := range vectors {
		got, err := HashJSON([]byte(vector.raw))
		if err != nil {
			t.Fatal(err)
		}
		if got != vector.hash {
			t.Fatalf("hash(%s) = %s, want %s", vector.raw, got, vector.hash)
		}
	}
}

func TestCanonicalizeNormalizesNumericEdges(t *testing.T) {
	vectors := []struct {
		raw  string
		form string
		hash string
	}{
		{raw: `{"n":-0}`, form: `{"n":0}`, hash: "f3013f933b9fb80ab6d995e7ad9da36f683837ba1d81e950c943d40111eac2f0"},
		{raw: `{"n":0.0}`, form: `{"n":0}`, hash: "f3013f933b9fb80ab6d995e7ad9da36f683837ba1d81e950c943d40111eac2f0"},
		{raw: `{"n":1.0}`, form: `{"n":1}`, hash: "2bfd14f43d17fc7cea24e0917a8879b4b2f880b8baeec1b9d90fbaad655e71bd"},
		{raw: `{"n":1e2}`, form: `{"n":100}`, hash: "b39022c4ed96525c42cd0e7ce55308533962a655f1c19d5dac2f03e9dd995b2c"},
		{raw: `{"n":1.2300}`, form: `{"n":1.23}`, hash: "c2f4a8099bdaf483ac3f465590b90ae2156f94d0d32c194bfbb06ca2289ad25f"},
	}
	for _, vector := range vectors {
		got, err := Canonicalize([]byte(vector.raw))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != vector.form {
			t.Fatalf("canonical %s = %s, want %s", vector.raw, got, vector.form)
		}
		hash, err := HashJSON([]byte(vector.raw))
		if err != nil {
			t.Fatal(err)
		}
		if hash != vector.hash {
			t.Fatalf("hash(%s) = %s, want %s", vector.raw, hash, vector.hash)
		}
	}
}

func TestCanonicalizeRejectsDuplicateKeys(t *testing.T) {
	if _, err := Canonicalize([]byte(`{"id":"first","id":"second"}`)); err == nil {
		t.Fatal("Canonicalize() accepted duplicate key")
	}
}

func TestCanonicalizeRejectsDeepNesting(t *testing.T) {
	if _, err := Canonicalize([]byte(nestedJSON(31))); err != nil {
		t.Fatal(err)
	}
	if _, err := Canonicalize([]byte(nestedJSON(32))); err == nil {
		t.Fatal("accepted JSON deeper than 32")
	}
}

func nestedJSON(depth int) string {
	return strings.Repeat(`{"k":`, depth) + `{"ok":true}` + strings.Repeat(`}`, depth)
}

func TestCanonicalizeRejectsNonFiniteNumbers(t *testing.T) {
	for _, raw := range []string{`{"n":1e309}`, `{"n":-1e309}`, `{"n":1e-400}`} {
		if _, err := Canonicalize([]byte(raw)); err == nil {
			t.Fatalf("Canonicalize(%s) accepted a non-finite number", raw)
		}
	}
	if _, err := Canonicalize([]byte(`{"n":1e-10}`)); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalizeRejectsTrailingValue(t *testing.T) {
	if _, err := Canonicalize([]byte(`{"id":"one"} {"id":"two"}`)); err == nil {
		t.Fatal("Canonicalize() accepted trailing JSON value")
	}
}
