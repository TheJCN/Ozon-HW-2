package product

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInvalidCatalog(t *testing.T) {
	for _, data := range []string{`broken`, `{"0":{"price":1}}`, `{"abc":{"price":1}}`, `{"1":{"price":-1}}`, `{"1":{"price":2147483648}}`} {
		path := filepath.Join(t.TempDir(), "products.json")
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("accepted invalid catalog: %s", data)
		}
	}
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("accepted missing catalog")
	}
}
