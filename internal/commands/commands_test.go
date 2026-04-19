package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joncarr/gws/internal/cli"
	"github.com/joncarr/gws/internal/config"
	"github.com/joncarr/gws/internal/google"
)

type fakeDirectory struct {
	info google.DomainInfo
	err  error
}

func (f fakeDirectory) DomainInfo(context.Context, config.Profile) (google.DomainInfo, error) {
	return f.info, f.err
}

func TestSetupWritesConfig(t *testing.T) {
	dir := t.TempDir()
	credentials := filepath.Join(dir, "client.json")
	if err := os.WriteFile(credentials, []byte(`{"type":"service_account","client_email":"svc@example.iam.gserviceaccount.com"}`), 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	var out bytes.Buffer
	r := Runner{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &bytes.Buffer{},
		Config: configPath,
		Directory: fakeDirectory{info: google.DomainInfo{
			CustomerID:    "C01",
			PrimaryDomain: "example.com",
		}},
	}
	parsed, err := cli.Parse([]string{"setup", "--profile", "prod", "--domain", "example.com", "--admin", "admin@example.com", "--credentials", credentials})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ActiveProfile != "prod" {
		t.Fatalf("ActiveProfile = %q", cfg.ActiveProfile)
	}
	if cfg.Profiles["prod"].AdminSubject != "admin@example.com" {
		t.Fatalf("AdminSubject = %q", cfg.Profiles["prod"].AdminSubject)
	}
}

func TestCheckConnectionUsesDirectoryClient(t *testing.T) {
	dir := t.TempDir()
	credentials := filepath.Join(dir, "client.json")
	token := filepath.Join(dir, "token.json")
	if err := os.WriteFile(credentials, []byte(`{"installed":{"client_id":"abc"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(token, []byte(`{"access_token":"token","token_type":"Bearer"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Empty()
	cfg.Profiles["default"] = config.Profile{
		Domain:          "example.com",
		AdminSubject:    "admin@example.com",
		CredentialsFile: credentials,
		TokenFile:       token,
		AuthMethod:      "oauth",
		Scopes:          []string{"scope"},
		Output:          "text",
	}
	configPath := filepath.Join(dir, "config.json")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	r := Runner{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Config: configPath,
		Directory: fakeDirectory{info: google.DomainInfo{
			CustomerID:    "C01",
			PrimaryDomain: "example.com",
		}},
	}
	parsed, err := cli.Parse([]string{"check", "connection"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Connection OK") {
		t.Fatalf("output = %q", out.String())
	}
}
