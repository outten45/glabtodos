package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	err := os.WriteFile(path, []byte(`host = "https://gitlab.example.com"
api_path = "/api/v3/"
op_path = "op://Personal/GitLab/API Token"
delay = "5m"
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, gotPath, err := loadFileConfig([]string{"glabtodos", "--config", path})
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != path || cfg.Host != "https://gitlab.example.com" || cfg.APIPath != "/api/v3/" || cfg.OPPath == "" || cfg.Delay != "5m" {
		t.Fatalf("unexpected config: path=%q config=%+v", gotPath, cfg)
	}
}

func TestLoadFileConfigExplicitMissingFile(t *testing.T) {
	_, _, err := loadFileConfig([]string{"glabtodos", "--config", filepath.Join(t.TempDir(), "missing.toml")})
	if err == nil {
		t.Fatal("expected an error for an explicitly missing config file")
	}
}

func TestLoadFileConfigNoConfig(t *testing.T) {
	cfg, _, err := loadFileConfig([]string{"glabtodos", "--no-config", "--config", "/does/not/exist"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg != (fileConfig{}) {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}
