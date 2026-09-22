package conformance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// coreFiles returns every production Go file of the service.
func coreFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	for _, root := range []string{filepath.Join("..", "..", "internal"), filepath.Join("..", "..", "cmd")} {
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func TestCoreHasNoDomainVerdictOrProfile(t *testing.T) {
	forbidden := regexp.MustCompile(`\b(Verdict|Profile|Thresholds?|ClaimValidation)\b|/profile"`)
	for _, path := range coreFiles(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if match := forbidden.Find(raw); match != nil {
			t.Fatalf("%s embeds domain interpretation %q", path, match)
		}
	}
}

func TestFixturesDoNotBecomeProductionVocabulary(t *testing.T) {
	for _, path := range coreFiles(t) {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"portfolio", "catalogue", "storefront", "database"} {
			if strings.Contains(strings.ToLower(string(raw)), forbidden) {
				t.Fatalf("%s embeds fixture vocabulary %q", path, forbidden)
			}
		}
	}
}
