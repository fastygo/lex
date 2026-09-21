package canonical

import "testing"

func TestDecodeRejectsInvalidUnicodeWithoutRepair(t *testing.T) {
	for _, raw := range []string{`{"x":"\ud800"}`, `{"x":"\udc00"}`, `{"\ud800":1}`, `{"x":"\ud800x"}`, "{\"x\":\"\xff\"}"} {
		if _, err := DecodeJSON([]byte(raw)); err == nil {
			t.Fatalf("DecodeJSON accepted %q", raw)
		}
		if _, err := HashJSON([]byte(raw)); err == nil {
			t.Fatalf("HashJSON accepted %q", raw)
		}
	}
	for _, raw := range []string{`{"x":"\ud83d\ude00"}`, `{"x":"�"}`} {
		if _, err := DecodeJSON([]byte(raw)); err != nil {
			t.Fatalf("valid Unicode %q: %v", raw, err)
		}
	}
}
