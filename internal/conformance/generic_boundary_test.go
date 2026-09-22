package conformance

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestGenericDecisionCoreDoesNotImportLegacyClaimValidation(t *testing.T) {
	paths := []string{
		filepath.Join("..", "wire", "decision.go"),
		filepath.Join("..", "wire", "decision_request.go"),
		filepath.Join("..", "wire", "decision_replay.go"),
		filepath.Join("..", "httpapi", "decision.go"),
		filepath.Join("..", "httpapi", "decision_request.go"),
	}
	for _, path := range paths {
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imported := range file.Imports {
			name, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(name, "/profile") || strings.Contains(name, "claimvalidation") {
				t.Fatalf("%s imports compatibility policy package %q", path, name)
			}
		}
	}
}

func TestGenericFixturesDoNotBecomeProductionVocabulary(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "wire", "decision.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"portfolio", "catalogue", "database", "storefront"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("generic decision core embeds fixture vocabulary %q", forbidden)
		}
	}
}
