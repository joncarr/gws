package commands

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/cli"
	"github.com/joncarr/gws/internal/config"
	"github.com/joncarr/gws/internal/google"
)

type fakeDirectory struct {
	group   google.GroupInfo
	groups  []google.GroupInfo
	info    google.DomainInfo
	member  google.MemberInfo
	members []google.MemberInfo
	ou      google.OrgUnitInfo
	ous     []google.OrgUnitInfo
	user    google.UserInfo
	users   []google.UserInfo
	err     error
}

func (f fakeDirectory) DomainInfo(context.Context, config.Profile) (google.DomainInfo, error) {
	return f.info, f.err
}

func (f fakeDirectory) Users(context.Context, config.Profile, int64) ([]google.UserInfo, error) {
	return f.users, f.err
}

func (f fakeDirectory) User(context.Context, config.Profile, string) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) CreateUser(context.Context, config.Profile, google.UserCreate) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) UpdateUser(context.Context, config.Profile, string, google.UserUpdate) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) SetUserSuspended(context.Context, config.Profile, string, bool) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) Groups(context.Context, config.Profile, int64) ([]google.GroupInfo, error) {
	return f.groups, f.err
}

func (f fakeDirectory) Group(context.Context, config.Profile, string) (google.GroupInfo, error) {
	return f.group, f.err
}

func (f fakeDirectory) CreateGroup(context.Context, config.Profile, google.GroupInfo) (google.GroupInfo, error) {
	return f.group, f.err
}

func (f fakeDirectory) UpdateGroup(context.Context, config.Profile, string, google.GroupInfo) (google.GroupInfo, error) {
	return f.group, f.err
}

func (f fakeDirectory) GroupMembers(context.Context, config.Profile, string, int64) ([]google.MemberInfo, error) {
	return f.members, f.err
}

func (f fakeDirectory) AddGroupMember(context.Context, config.Profile, string, string, string) (google.MemberInfo, error) {
	return f.member, f.err
}

func (f fakeDirectory) RemoveGroupMember(context.Context, config.Profile, string, string) error {
	return f.err
}

func (f fakeDirectory) OrgUnits(context.Context, config.Profile) ([]google.OrgUnitInfo, error) {
	return f.ous, f.err
}

func (f fakeDirectory) OrgUnit(context.Context, config.Profile, string) (google.OrgUnitInfo, error) {
	return f.ou, f.err
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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

func TestAuthStatusReportsReadyOAuthProfile(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
		Output:          "text",
	}
	configPath := filepath.Join(dir, "config.json")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    configPath,
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"auth", "status"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Status: ready for API validation") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintUsersUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{users: []google.UserInfo{
			{PrimaryEmail: "ada@example.com", Name: "Ada Lovelace", OrgUnitPath: "/", Suspended: false},
		}},
	}
	parsed, err := cli.Parse([]string{"print", "users", "--limit", "10"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInfoUserUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{user: google.UserInfo{
			PrimaryEmail: "ada@example.com",
			Name:         "Ada Lovelace",
			OrgUnitPath:  "/Engineering",
			IsAdmin:      true,
		}},
	}
	parsed, err := cli.Parse([]string{"info", "user", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Primary email: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), "Admin: true") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInfoUserRequiresEmail(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"info", "user"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws info user user@example.com") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateUserUsesDirectoryClient(t *testing.T) {
	dir := t.TempDir()
	credentials := filepath.Join(dir, "client.json")
	token := filepath.Join(dir, "token.json")
	passwordFile := filepath.Join(dir, "password.txt")
	if err := os.WriteFile(credentials, []byte(`{"installed":{"client_id":"abc"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(token, []byte(`{"access_token":"token","token_type":"Bearer"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(passwordFile, []byte("not-a-real-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Empty()
	cfg.Profiles["default"] = config.Profile{
		Domain:          "example.com",
		AdminSubject:    "admin@example.com",
		CredentialsFile: credentials,
		TokenFile:       token,
		AuthMethod:      "oauth",
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{user: google.UserInfo{
			PrimaryEmail: "ada@example.com",
			Name:         "Ada Lovelace",
			OrgUnitPath:  "/Engineering",
		}},
	}
	parsed, err := cli.Parse([]string{"create", "user", "ada@example.com", "--given-name", "Ada", "--family-name", "Lovelace", "--password-file", passwordFile, "--org-unit", "Engineering"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "User created: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if strings.Contains(out.String(), "not-a-real-password") {
		t.Fatalf("output leaked password: %q", out.String())
	}
}

func TestCreateUserRequiresNames(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"create", "user", "ada@example.com", "--family-name", "Lovelace", "--password", "secret"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--given-name is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateUserRequiresPassword(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"create", "user", "ada@example.com", "--given-name", "Ada", "--family-name", "Lovelace"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--password-file") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateUserRejectsTwoPasswordSources(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"create", "user", "ada@example.com", "--given-name", "Ada", "--family-name", "Lovelace", "--password", "secret", "--password-file", "password.txt"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrintGroupsUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{groups: []google.GroupInfo{
			{Email: "eng@example.com", Name: "Engineering", DirectMembersCount: 7, AdminCreated: true},
		}},
	}
	parsed, err := cli.Parse([]string{"print", "groups", "--limit", "10"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "eng@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInfoGroupUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{group: google.GroupInfo{
			Email:              "eng@example.com",
			Name:               "Engineering",
			Description:        "Engineering team",
			DirectMembersCount: 7,
			AdminCreated:       true,
		}},
	}
	parsed, err := cli.Parse([]string{"info", "group", "eng@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Email: eng@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), "Direct members: 7") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInfoGroupRequiresEmail(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"info", "group"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws info group group@example.com") {
		t.Fatalf("error = %v", err)
	}
}

func TestCreateGroupUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{group: google.GroupInfo{
			Email:       "eng@example.com",
			Name:        "Engineering",
			Description: "Builders",
		}},
	}
	parsed, err := cli.Parse([]string{"create", "group", "eng@example.com", "--name", "Engineering", "--description", "Builders"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group created: eng@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCreateGroupRequiresName(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"create", "group", "eng@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateGroupUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{group: google.GroupInfo{
			Email: "eng@example.com",
			Name:  "Engineering Team",
		}},
	}
	parsed, err := cli.Parse([]string{"update", "group", "eng@example.com", "--name", "Engineering Team"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group updated: eng@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUpdateGroupRequiresChange(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"update", "group", "eng@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrintGroupMembersUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "MEMBER", Type: "USER", Status: "ACTIVE"},
		}},
	}
	parsed, err := cli.Parse([]string{"print", "group-members", "eng@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAddGroupMemberUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{member: google.MemberInfo{
			Email: "ada@example.com",
			Role:  "MANAGER",
		}},
	}
	parsed, err := cli.Parse([]string{"add", "group-member", "eng@example.com", "ada@example.com", "--role", "manager"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group member added: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), "Role: MANAGER") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestAddGroupMemberRejectsInvalidRole(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"add", "group-member", "eng@example.com", "ada@example.com", "--role", "bad"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--role must be OWNER, MANAGER, or MEMBER") {
		t.Fatalf("error = %v", err)
	}
}

func TestRemoveGroupMemberUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
		Output:          "text",
	}
	configPath := filepath.Join(dir, "config.json")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    configPath,
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"remove", "group-member", "eng@example.com", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group member removed: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintOrgUnitsUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{ous: []google.OrgUnitInfo{
			{OrgUnitPath: "/", Name: "root"},
			{OrgUnitPath: "/Engineering", Name: "Engineering", ParentOrgUnitPath: "/", Description: "Builders"},
		}},
	}
	parsed, err := cli.Parse([]string{"print", "ous"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "/Engineering") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInfoOrgUnitUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{ou: google.OrgUnitInfo{
			OrgUnitPath:       "/Engineering",
			Name:              "Engineering",
			ParentOrgUnitPath: "/",
			Description:       "Builders",
			OrgUnitID:         "id:eng",
		}},
	}
	parsed, err := cli.Parse([]string{"info", "ou", "Engineering"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Path: /Engineering") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), "Description: Builders") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestInfoOrgUnitRequiresPath(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"info", "ou"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws info ou /Engineering") {
		t.Fatalf("error = %v", err)
	}
}

func TestPrintUsersReportsMissingScopeBeforeAPICall(t *testing.T) {
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
		Scopes:          []string{"https://www.googleapis.com/auth/admin.directory.customer.readonly"},
		Output:          "text",
	}
	configPath := filepath.Join(dir, "config.json")
	if err := config.Save(configPath, cfg); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    configPath,
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"print", "users"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "admin.directory.user") {
		t.Fatalf("error = %v", err)
	}
}

func TestSuspendUserUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{user: google.UserInfo{
			PrimaryEmail: "ada@example.com",
			Suspended:    true,
		}},
	}
	parsed, err := cli.Parse([]string{"suspend", "user", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "User suspended: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUnsuspendUserUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{user: google.UserInfo{
			PrimaryEmail: "ada@example.com",
			Suspended:    false,
		}},
	}
	parsed, err := cli.Parse([]string{"unsuspend", "user", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "User unsuspended: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestSuspendUserRequiresEmail(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"suspend", "user"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws suspend user user@example.com") {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateUserUsesDirectoryClient(t *testing.T) {
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
		Scopes:          append([]string(nil), auth.RequiredScopes...),
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
		Directory: fakeDirectory{user: google.UserInfo{
			PrimaryEmail: "ada@example.com",
			Name:         "Ada Byron",
			OrgUnitPath:  "/Engineering",
		}},
	}
	parsed, err := cli.Parse([]string{"update", "user", "ada@example.com", "--given-name", "Ada", "--family-name", "Byron", "--org-unit", "Engineering"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "User updated: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), "Org unit: /Engineering") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUpdateUserRequiresEmail(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"update", "user"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws update user user@example.com") {
		t.Fatalf("error = %v", err)
	}
}

func TestUpdateUserRequiresChange(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"update", "user", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("error = %v", err)
	}
}
