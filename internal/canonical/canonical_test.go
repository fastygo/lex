package canonical

import "testing"

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

func TestCanonicalizeRejectsDuplicateKeys(t *testing.T) {
	if _, err := Canonicalize([]byte(`{"id":"first","id":"second"}`)); err == nil {
		t.Fatal("Canonicalize() accepted duplicate key")
	}
}

func TestCanonicalizeRejectsTrailingValue(t *testing.T) {
	if _, err := Canonicalize([]byte(`{"id":"one"} {"id":"two"}`)); err == nil {
		t.Fatal("Canonicalize() accepted trailing JSON value")
	}
}
