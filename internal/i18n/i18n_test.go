package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every key must exist in every language, apart from ".one" singular forms,
// which only languages that need them define.
func TestDictionariesHaveSameKeys(t *testing.T) {
	for lang, dict := range dictionaries {
		for key := range dictionaries[Default] {
			if strings.HasSuffix(key, ".one") {
				continue
			}
			if _, ok := dict[key]; !ok {
				t.Errorf("%s: missing key %q", lang, key)
			}
		}
		for key := range dict {
			if strings.HasSuffix(key, ".one") {
				continue
			}
			if _, ok := dictionaries[Default][key]; !ok {
				t.Errorf("%s: key %q is not in the default dictionary", lang, key)
			}
		}
	}
}

// Every literal key used in a template or handler must be translated, so no
// raw "some.key" ever shows up on a page.
func TestUsedKeysExist(t *testing.T) {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`call \$?\.T "([^"]+)"`),
		regexp.MustCompile(`i18n\.T\(lang, "([^"]+)"`),
		regexp.MustCompile(`\bt\("([^"]+)"\)`),
		regexp.MustCompile(`"TitleKey":\s+"([^"]+)"`),
	}

	var files []string
	for _, root := range []string{"../../web/templates", "../handlers", "../../cmd"} {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() && (strings.HasSuffix(path, ".html") || strings.HasSuffix(path, ".go")) {
				files = append(files, path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	found := 0
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var code []string
		for _, line := range strings.Split(string(src), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(line), "//") { // skip doc comments
				code = append(code, line)
			}
		}
		for _, re := range patterns {
			for _, m := range re.FindAllStringSubmatch(strings.Join(code, "\n"), -1) {
				found++
				if _, ok := dictionaries[Default][m[1]]; !ok {
					t.Errorf("%s: key %q is not translated", path, m[1])
				}
			}
		}
	}
	if found < 100 {
		t.Fatalf("only found %d key usages, the scan patterns are probably broken", found)
	}

	// Keys built dynamically: nav sections and score buckets.
	for _, section := range []string{"today", "feed", "friends", "profile", "settings"} {
		if _, ok := dictionaries[Default]["nav."+section]; !ok {
			t.Errorf("missing nav.%s", section)
		}
	}
	for _, b := range []string{"excellent", "great", "good", "low", "poor"} {
		if _, ok := dictionaries[Default]["bucket."+b]; !ok {
			t.Errorf("missing bucket.%s", b)
		}
	}
}

func TestTranslatorSingular(t *testing.T) {
	en := Translator("en")
	if got := en("months.days", 1); got != "1 day" {
		t.Errorf("en singular: got %q", got)
	}
	if got := en("months.days", 3); got != "3 days" {
		t.Errorf("en plural: got %q", got)
	}
	if got := en("mood.month_summary", "7.2", 1); got != "Average 7.2 · 1 day" {
		t.Errorf("en singular with two args: got %q", got)
	}
	if got := Translator("tr")("months.days", 1); got != "1 gün" {
		t.Errorf("tr: got %q", got)
	}
	if got := en("no.such.key"); got != "no.such.key" {
		t.Errorf("missing key should echo itself, got %q", got)
	}
}
