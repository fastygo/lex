package wire

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/fastygo/lex/internal/canonical"
)

// TestBundleTypeMirrorsSchema keeps the hand-maintained Bundle type equal to
// the published schema: same object properties at every level the type
// names, no more and no fewer.
func TestBundleTypeMirrorsSchema(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("schema", "replay-bundle.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	value, err := canonical.DecodeJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	schema := value.(map[string]any)
	properties := schema["properties"].(map[string]any)

	expectMirror(t, "bundle", reflect.TypeOf(Bundle{}), keys(properties))
	expectMirror(t, "entity", reflect.TypeOf(EntityRecord{}), keys(properties["entity"].(map[string]any)["properties"].(map[string]any)))
	expectMirror(t, "context", reflect.TypeOf(ContextRecord{}), keys(properties["context"].(map[string]any)["properties"].(map[string]any)))
	expectMirror(t, "question_set", reflect.TypeOf(QuestionSetRecord{}), keys(properties["question_set"].(map[string]any)["properties"].(map[string]any)))
	expectMirror(t, "policy", reflect.TypeOf(PolicyRecord{}), keys(properties["policy"].(map[string]any)["properties"].(map[string]any)))

	decisionUnion := map[string]struct{}{}
	for _, variant := range properties["decision_set"].(map[string]any)["oneOf"].([]any) {
		for _, key := range keys(variant.(map[string]any)["properties"].(map[string]any)) {
			decisionUnion[key] = struct{}{}
		}
	}
	expectMirror(t, "decision_set", reflect.TypeOf(DecisionRecord{}), keys(decisionUnion))
}

func TestDecodeBundleRoundTripsSealedBytes(t *testing.T) {
	raw := sealedBundle(t)
	bundle, err := DecodeBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.ProtocolVersion != ProtocolVersion || bundle.VerifierVersion != VerifierVersion || bundle.Entity.ProjectID != "project-test" || bundle.DecisionSet.Skipped || len(bundle.DecisionSet.Answers) == 0 {
		t.Fatalf("bundle = %+v", bundle)
	}
	again, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	first, err := canonical.HashJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonical.HashJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("typed decode and re-encode changed the canonical bundle")
	}
}

func expectMirror(t *testing.T, name string, typ reflect.Type, want []string) {
	t.Helper()
	got := make([]string, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("json")
		got = append(got, strings.Split(tag, ",")[0])
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: type fields %v, schema properties %v", name, got, want)
	}
}

func keys[V any](object map[string]V) []string {
	out := make([]string, 0, len(object))
	for key := range object {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}
