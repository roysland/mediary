package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var templateKeyPattern = regexp.MustCompile(`t\s+"([^"]+)"`)

func TestTemplateTranslationKeysAreDefined(t *testing.T) {
	viewsDir := projectViewsDir(t)

	keySource := map[string]string{}
	err := filepath.WalkDir(viewsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".html" {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		matches := templateKeyPattern.FindAllStringSubmatchIndex(string(content), -1)
		for _, m := range matches {
			key := string(content[m[2]:m[3]])
			if _, exists := keySource[key]; exists {
				continue
			}

			line := strings.Count(string(content[:m[2]]), "\n") + 1
			relPath, relErr := filepath.Rel(viewsDir, path)
			if relErr != nil {
				relPath = path
			}
			keySource[key] = relPath + ":" + strconv.Itoa(line)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to scan templates: %v", err)
	}

	missing := make([]string, 0)
	for key, source := range keySource {
		if T(key) == key {
			missing = append(missing, key+" ("+source+")")
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("missing i18n keys used in templates:\n%s", strings.Join(missing, "\n"))
	}
}

func projectViewsDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return filepath.Join(wd, "..", "..", "internal", "views")
}
