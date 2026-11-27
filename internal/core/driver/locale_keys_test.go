package driver_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lotustab/internal/core/driver"
	"lotustab/internal/drivers/clickhouse"
	"lotustab/internal/drivers/mysql"
	"lotustab/internal/drivers/postgres"
)

func TestLocaleCatalogsMatchDriverKeys(t *testing.T) {
	root := repoRoot(t)
	en := flattenLocale(t, filepath.Join(root, "frontend", "src", "locales", "en.json"))
	zh := flattenLocale(t, filepath.Join(root, "frontend", "src", "locales", "zh-CN.json"))

	if len(en) == 0 {
		t.Fatal("en.json has no message keys")
	}
	for key := range en {
		if _, ok := zh[key]; !ok {
			t.Errorf("zh-CN.json missing %s", key)
		}
	}
	for key := range zh {
		if _, ok := en[key]; !ok {
			t.Errorf("en.json missing %s", key)
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
