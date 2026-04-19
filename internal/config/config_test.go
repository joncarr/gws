package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	profile, err := NewProfile("default")
	if err != nil {
		t.Fatal(err)
	}
	profile.Domain = "example.com"
	cfg := Empty()
	cfg.Profiles["default"] = profile
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ActiveProfile != "default" {
		t.Fatalf("ActiveProfile = %q", got.ActiveProfile)
	}
	if got.Profiles["default"].Domain != "example.com" {
		t.Fatalf("Domain = %q", got.Profiles["default"].Domain)
	}
}

func TestApplyEnv(t *testing.T) {
	t.Setenv("GWS_DOMAIN", "env.example.com")
	cfg := Empty()
	cfg.Profiles["default"] = Profile{Domain: "file.example.com"}
	got := ApplyEnv(cfg)
	if got.Profiles["default"].Domain != "env.example.com" {
		t.Fatalf("Domain = %q", got.Profiles["default"].Domain)
	}
	os.Unsetenv("GWS_DOMAIN")
}
