package config_test

import (
	"path/filepath"
	"testing"

	"github.com/voska/zonasul/internal/config"
)

func TestLoadSaveConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := &config.Config{
		CEP:          "22240-003",
		Street:       "Rua das Laranjeiras",
		Number:       "100",
		Complement:   "Apto 200",
		Neighborhood: "Laranjeiras",
		City:         "Rio de Janeiro",
		State:        "RJ",
		OrderFormID:  "abc123",
	}

	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.CEP != "22240-003" || loaded.Number != "100" || loaded.OrderFormID != "abc123" {
		t.Errorf("config mismatch: %+v", loaded)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := config.Load("/nonexistent/path.json")
	if err != nil {
		t.Fatal("should not error on missing file")
	}
	if cfg.CEP != "" {
		t.Error("expected empty config")
	}
}

func TestDefaultPath(t *testing.T) {
	p := config.DefaultPath()
	if !filepath.IsAbs(p) {
		t.Errorf("expected absolute path, got %s", p)
	}
}
