package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateCredentialsFileOAuthInstalled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(`{"installed":{"client_id":"abc.apps.googleusercontent.com"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ValidateCredentialsFile(path)
	if err != nil {
		t.Fatalf("ValidateCredentialsFile() error = %v", err)
	}
	if got.Type != "oauth_installed" || got.ClientID == "" {
		t.Fatalf("CredentialInfo = %+v", got)
	}
}

func TestValidateCredentialsFileRejectsUnknownShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(`{"hello":"world"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateCredentialsFile(path); err == nil {
		t.Fatal("ValidateCredentialsFile() error = nil")
	}
}

func TestValidateCredentialsFileServiceAccount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "service-account.json")
	if err := os.WriteFile(path, []byte(`{"type":"service_account","client_id":"123456789","client_email":"svc@example.iam.gserviceaccount.com"}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := ValidateCredentialsFile(path)
	if err != nil {
		t.Fatalf("ValidateCredentialsFile() error = %v", err)
	}
	if got.Type != MethodServiceAccount || got.ClientID != "123456789" {
		t.Fatalf("CredentialInfo = %+v", got)
	}
}
