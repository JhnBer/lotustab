package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/clickhouse"
	"lotustab/internal/drivers/mysql"
	"lotustab/internal/drivers/postgres"
)

func TestLocaleCatalogsMatchDriverKeys(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "frontend", "src", "locales")
	en := flattenLocale(t, filepath.Join(dir, "en.json"))
	if len(en) == 0 {
		t.Fatal("en.json has no message keys")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || name == "en.json" {
			continue
		}
		other := flattenLocale(t, filepath.Join(dir, name))
		for key := range en {
			if _, ok := other[key]; !ok {
				t.Errorf("%s missing %s", name, key)
			}
		}
		for key := range other {
			if _, ok := en[key]; !ok {
				t.Errorf("en.json missing %s from %s", key, name)
			}
		}
	}

	reg := driver.NewRegistry()
	for _, d := range []driver.Driver{postgres.New(), mysql.New(), clickhouse.New()} {
		if err := reg.Register(d); err != nil {
			t.Fatal(err)
		}
	}
	for _, info := range reg.All() {
		for _, field := range info.ConnectionSpec() {
			requireLocaleKey(t, en, field.LabelKey)
			for _, option := range field.Options {
				requireLocaleKey(t, en, option.LabelKey)
			}
		}
	}
}

func requireLocaleKey(t *testing.T, keys map[string]struct{}, key string) {
	t.Helper()
	if key == "" {
		t.Fatal("empty label key")
	}
	if _, ok := keys[key]; !ok {
		t.Errorf("en.json missing driver key %s", key)
	}
}

func flattenLocale(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var tree any
	if err := json.Unmarshal(raw, &tree); err != nil {
		t.Fatal(err)
	}
	out := map[string]struct{}{}
	flattenLocaleValue(t, "", tree, out)
	return out
}

func flattenLocaleValue(t *testing.T, prefix string, value any, out map[string]struct{}) {
	t.Helper()
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			next := key
			if prefix != "" {
				next = prefix + "." + key
			}
			flattenLocaleValue(t, next, child, out)
		}
	case string:
		if prefix == "" {
			t.Fatal("locale file has a string at the root")
		}
		out[prefix] = struct{}{}
	default:
		t.Fatalf("locale key %s is not a string or object", prefix)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
