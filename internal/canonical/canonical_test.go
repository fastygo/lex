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
