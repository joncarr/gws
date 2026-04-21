package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/cli"
	"github.com/joncarr/gws/internal/config"
	"github.com/joncarr/gws/internal/google"
	"google.golang.org/api/googleapi"
)

type fakeDirectory struct {
	alias            google.AliasInfo
	addedMembers     *[]string
	addedMemberRoles *[]string
	aliases          []google.AliasInfo
	domain           google.WorkspaceDomainInfo
	domains          []google.WorkspaceDomainInfo
	dAlias           google.DomainAliasInfo
	dAliases         []google.DomainAliasInfo
	group            google.GroupInfo
	groups           []google.GroupInfo
	groupListOpts    *google.GroupListOptions
	info             google.DomainInfo
	member           google.MemberInfo
	members          []google.MemberInfo
	ou               google.OrgUnitInfo
	ous              []google.OrgUnitInfo
	user             google.UserInfo
	users            []google.UserInfo
	userListOpts     *google.UserListOptions
	usersDelay       time.Duration
	removedMembers   *[]string
	updatedMembers   *[]string
	updatedUser      *google.UserUpdate
	updatedGroup     *google.GroupInfo
	err              error
}

type fakeSheets struct {
	info                 google.SheetInfo
	gotTitle             string
	gotRows              [][]string
	readRows             [][]string
	gotReadSpreadsheetID string
	gotReadRange         string
	err                  error
}

type fakeGmail struct {
	delegate  google.DelegateInfo
	delegates []google.DelegateInfo
	err       error
}

func (f *fakeSheets) CreateSheet(_ context.Context, _ config.Profile, title string, rows [][]string) (google.SheetInfo, error) {
	f.gotTitle = title
	f.gotRows = append([][]string(nil), rows...)
	return f.info, f.err
}

func (f *fakeSheets) ReadRows(_ context.Context, _ config.Profile, spreadsheetID string, readRange string) ([][]string, error) {
	f.gotReadSpreadsheetID = spreadsheetID
	f.gotReadRange = readRange
	return append([][]string(nil), f.readRows...), f.err
}

func (f fakeGmail) Delegates(context.Context, config.Profile, string) ([]google.DelegateInfo, error) {
	return f.delegates, f.err
}

func (f fakeGmail) Delegate(context.Context, config.Profile, string, string) (google.DelegateInfo, error) {
	return f.delegate, f.err
}

func (f fakeGmail) CreateDelegate(context.Context, config.Profile, string, string) (google.DelegateInfo, error) {
	return f.delegate, f.err
}

func (f fakeGmail) DeleteDelegate(context.Context, config.Profile, string, string) error {
	return f.err
}

func (f fakeDirectory) DomainInfo(context.Context, config.Profile) (google.DomainInfo, error) {
	return f.info, f.err
}

func (f fakeDirectory) Domains(context.Context, config.Profile) ([]google.WorkspaceDomainInfo, error) {
	return f.domains, f.err
}

func (f fakeDirectory) Domain(context.Context, config.Profile, string) (google.WorkspaceDomainInfo, error) {
	return f.domain, f.err
}

func (f fakeDirectory) CreateDomain(context.Context, config.Profile, string) (google.WorkspaceDomainInfo, error) {
	return f.domain, f.err
}

func (f fakeDirectory) DeleteDomain(context.Context, config.Profile, string) error {
	return f.err
}

func (f fakeDirectory) DomainAliases(context.Context, config.Profile) ([]google.DomainAliasInfo, error) {
	return f.dAliases, f.err
}

func (f fakeDirectory) DomainAlias(context.Context, config.Profile, string) (google.DomainAliasInfo, error) {
	return f.dAlias, f.err
}

func (f fakeDirectory) CreateDomainAlias(context.Context, config.Profile, string, string) (google.DomainAliasInfo, error) {
	return f.dAlias, f.err
}

func (f fakeDirectory) DeleteDomainAlias(context.Context, config.Profile, string) error {
	return f.err
}

func (f fakeDirectory) Users(ctx context.Context, _ config.Profile, opts google.UserListOptions) ([]google.UserInfo, error) {
	if f.userListOpts != nil {
		*f.userListOpts = opts
	}
	if f.usersDelay > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(f.usersDelay):
		}
	}
	return f.users, f.err
}

func (f fakeDirectory) User(context.Context, config.Profile, string) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) CreateUser(context.Context, config.Profile, google.UserCreate) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) UpdateUser(_ context.Context, _ config.Profile, _ string, update google.UserUpdate) (google.UserInfo, error) {
	if f.updatedUser != nil {
		*f.updatedUser = update
	}
	return f.user, f.err
}

func (f fakeDirectory) DeleteUser(context.Context, config.Profile, string) error {
	return f.err
}

func (f fakeDirectory) SetUserAdmin(context.Context, config.Profile, string, bool) error {
	return f.err
}

func (f fakeDirectory) SetUserSuspended(context.Context, config.Profile, string, bool) (google.UserInfo, error) {
	return f.user, f.err
}

func (f fakeDirectory) UserAliases(context.Context, config.Profile, string) ([]google.AliasInfo, error) {
	return f.aliases, f.err
}

func (f fakeDirectory) CreateUserAlias(context.Context, config.Profile, string, string) (google.AliasInfo, error) {
	return f.alias, f.err
}

func (f fakeDirectory) DeleteUserAlias(context.Context, config.Profile, string, string) error {
	return f.err
}

func (f fakeDirectory) Groups(_ context.Context, _ config.Profile, opts google.GroupListOptions) ([]google.GroupInfo, error) {
	if f.groupListOpts != nil {
		*f.groupListOpts = opts
	}
	return f.groups, f.err
}

func (f fakeDirectory) Group(context.Context, config.Profile, string) (google.GroupInfo, error) {
	return f.group, f.err
}

func (f fakeDirectory) CreateGroup(context.Context, config.Profile, google.GroupInfo) (google.GroupInfo, error) {
	return f.group, f.err
}

func (f fakeDirectory) UpdateGroup(_ context.Context, _ config.Profile, _ string, group google.GroupInfo) (google.GroupInfo, error) {
	if f.updatedGroup != nil {
		*f.updatedGroup = group
	}
	return f.group, f.err
}

func (f fakeDirectory) DeleteGroup(context.Context, config.Profile, string) error {
	return f.err
}

func (f fakeDirectory) GroupAliases(context.Context, config.Profile, string) ([]google.AliasInfo, error) {
	return f.aliases, f.err
}

func (f fakeDirectory) CreateGroupAlias(context.Context, config.Profile, string, string) (google.AliasInfo, error) {
	return f.alias, f.err
}

func (f fakeDirectory) DeleteGroupAlias(context.Context, config.Profile, string, string) error {
	return f.err
}

func (f fakeDirectory) GroupMembers(context.Context, config.Profile, string, int64) ([]google.MemberInfo, error) {
	return f.members, f.err
}

func (f fakeDirectory) GroupMember(context.Context, config.Profile, string, string) (google.MemberInfo, error) {
	return f.member, f.err
}

func (f fakeDirectory) AddGroupMember(_ context.Context, _ config.Profile, _ string, memberEmail string, role string) (google.MemberInfo, error) {
	if f.addedMembers != nil {
		*f.addedMembers = append(*f.addedMembers, memberEmail)
	}
	if f.addedMemberRoles != nil {
		*f.addedMemberRoles = append(*f.addedMemberRoles, memberEmail+":"+role)
	}
	return f.member, f.err
}

func (f fakeDirectory) UpdateGroupMember(_ context.Context, _ config.Profile, _ string, memberEmail string, role string) (google.MemberInfo, error) {
	if f.updatedMembers != nil {
		*f.updatedMembers = append(*f.updatedMembers, memberEmail+":"+role)
	}
	return f.member, f.err
}

func (f fakeDirectory) RemoveGroupMember(_ context.Context, _ config.Profile, _ string, memberEmail string) error {
	if f.removedMembers != nil {
		*f.removedMembers = append(*f.removedMembers, memberEmail)
	}
	return f.err
}

func (f fakeDirectory) OrgUnits(context.Context, config.Profile) ([]google.OrgUnitInfo, error) {
	return f.ous, f.err
}

func (f fakeDirectory) OrgUnit(context.Context, config.Profile, string) (google.OrgUnitInfo, error) {
	return f.ou, f.err
}

func (f fakeDirectory) CreateOrgUnit(context.Context, config.Profile, google.OrgUnitCreate) (google.OrgUnitInfo, error) {
	return f.ou, f.err
}

func (f fakeDirectory) UpdateOrgUnit(context.Context, config.Profile, string, google.OrgUnitUpdate) (google.OrgUnitInfo, error) {
	return f.ou, f.err
}

func (f fakeDirectory) DeleteOrgUnit(context.Context, config.Profile, string) error {
	return f.err
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

func TestDomainCommandsUseDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		domain: google.WorkspaceDomainInfo{
			DomainName: "example.com",
			IsPrimary:  true,
			Verified:   true,
		},
		domains: []google.WorkspaceDomainInfo{
			{DomainName: "example.com", IsPrimary: true, Verified: true},
			{DomainName: "very-long-secondary.example.com", IsPrimary: false, Verified: true},
		},
	})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "print", args: []string{"print", "domains"}, want: "example.com"},
		{name: "info", args: []string{"info", "domain", "example.com"}, want: "Domain: example.com"},
		{name: "create", args: []string{"create", "domain", "example.com"}, want: "Domain created: example.com"},
		{name: "delete", args: []string{"delete", "domain", "example.com", "--confirm"}, want: "Domain deleted: example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out.Reset()
			parsed, err := cli.Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Run(context.Background(), parsed); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q", out.String())
			}
			if tc.name == "print" {
				assertAlignedColumn(t, out.String(), "Primary", []string{"true", "false"})
			}
		})
	}
}

func TestDomainAliasCommandsUseDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		dAlias: google.DomainAliasInfo{
			DomainAliasName:  "alias.example.com",
			ParentDomainName: "example.com",
			Verified:         true,
		},
		dAliases: []google.DomainAliasInfo{
			{DomainAliasName: "alias.example.com", ParentDomainName: "example.com", Verified: true},
		},
	})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "print", args: []string{"print", "domain-aliases"}, want: "alias.example.com"},
		{name: "info", args: []string{"info", "domain-alias", "alias.example.com"}, want: "Domain alias: alias.example.com"},
		{name: "create", args: []string{"create", "domain-alias", "alias.example.com", "--parent", "example.com"}, want: "Domain alias created: alias.example.com"},
		{name: "delete", args: []string{"delete", "domain-alias", "alias.example.com", "--confirm"}, want: "Domain alias deleted: alias.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out.Reset()
			parsed, err := cli.Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Run(context.Background(), parsed); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

func TestCreateDomainAliasRequiresParent(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"create", "domain-alias", "alias.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--parent is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestSetUserAdminCommandsUseDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "make", args: []string{"make", "admin", "ada@example.com"}, want: "User made admin: ada@example.com"},
		{name: "revoke", args: []string{"revoke", "admin", "ada@example.com"}, want: "User admin revoked: ada@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out.Reset()
			parsed, err := cli.Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Run(context.Background(), parsed); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q", out.String())
			}
		})
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
			{PrimaryEmail: "avery.long.user@example.com", Name: "Long User", OrgUnitPath: "/Engineering", Suspended: true},
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
	assertAlignedColumn(t, out.String(), "Name", []string{"Ada Lovelace", "Long User"})
	assertAlignedColumn(t, out.String(), "Suspended", []string{"false", "true"})
}

func TestPrintUsersPassesListOptions(t *testing.T) {
	var got google.UserListOptions
	r, _ := testRunnerWithConfig(t, fakeDirectory{
		userListOpts: &got,
		users:        []google.UserInfo{{PrimaryEmail: "ada@example.com"}},
	})
	parsed, err := cli.Parse([]string{
		"print", "users",
		"--limit", "25",
		"--domain", "example.org",
		"--query", "isSuspended=false",
		"--org-unit", "/Engineering",
		"--show-deleted",
		"--sort", "familyName",
		"--order", "desc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Limit != 25 {
		t.Fatalf("Limit = %d", got.Limit)
	}
	if got.Domain != "example.org" {
		t.Fatalf("Domain = %q", got.Domain)
	}
	if got.Query != "isSuspended=false orgUnitPath='/Engineering'" {
		t.Fatalf("Query = %q", got.Query)
	}
	if !got.ShowDeleted {
		t.Fatal("ShowDeleted = false")
	}
	if got.OrderBy != "familyName" || got.SortOrder != "DESCENDING" {
		t.Fatalf("sort = %q %q", got.OrderBy, got.SortOrder)
	}
}

func TestPrintUsersAcceptsLimitAll(t *testing.T) {
	var got google.UserListOptions
	r, _ := testRunnerWithConfig(t, fakeDirectory{
		userListOpts: &got,
		users:        []google.UserInfo{{PrimaryEmail: "ada@example.com"}},
	})
	parsed, err := cli.Parse([]string{"print", "users", "--limit", "all"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !got.FetchAll {
		t.Fatal("FetchAll = false")
	}
	if got.Limit != 0 {
		t.Fatalf("Limit = %d", got.Limit)
	}
}

func TestPrintUsersSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		users: []google.UserInfo{{
			PrimaryEmail: "ada@example.com",
			Name:         "Ada Lovelace",
			Suspended:    true,
		}},
	})
	parsed, err := cli.Parse([]string{"print", "users", "--fields", "primaryEmail,suspended", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Primary Email,Suspended\nada@example.com,true\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintUsersSupportsSelectedJSONFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		users: []google.UserInfo{{
			PrimaryEmail: "ada@example.com",
			Name:         "Ada Lovelace",
			Suspended:    true,
			OrgUnitPath:  "/Engineering",
		}},
	})
	parsed, err := cli.Parse([]string{"print", "users", "--format", "json", "--fields", "primaryEmail,orgUnitPath"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal json: %v\n%s", err, out.String())
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d", len(got))
	}
	if got[0]["primaryEmail"] != "ada@example.com" || got[0]["orgUnitPath"] != "/Engineering" {
		t.Fatalf("got = %#v", got[0])
	}
	if len(got[0]) != 2 {
		t.Fatalf("got keys = %#v", got[0])
	}
}

func TestPrintUsersExportsToSheet(t *testing.T) {
	sheets := &fakeSheets{info: google.SheetInfo{
		Title:          "gws users",
		SpreadsheetURL: "https://docs.google.com/spreadsheets/d/123",
	}}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		users: []google.UserInfo{{
			PrimaryEmail: "ada@example.com",
			Name:         "Ada Lovelace",
		}},
	})
	r.Sheets = sheets
	parsed, err := cli.Parse([]string{"print", "users", "--fields", "primaryEmail,name", "--sheet"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if sheets.gotTitle != "gws users" {
		t.Fatalf("title = %q", sheets.gotTitle)
	}
	wantRows := [][]string{
		{"Primary Email", "Name"},
		{"ada@example.com", "Ada Lovelace"},
	}
	if !equalStringRows(sheets.gotRows, wantRows) {
		t.Fatalf("rows = %#v", sheets.gotRows)
	}
	if !strings.Contains(out.String(), "Sheet created: gws users") || !strings.Contains(out.String(), sheets.info.SpreadsheetURL) {
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
			ID:                         "12345",
			PrimaryEmail:               "ada@example.com",
			Name:                       "Ada Lovelace",
			GivenName:                  "Ada",
			FamilyName:                 "Lovelace",
			Aliases:                    []string{"a@example.com"},
			NonEditableAliases:         []string{"ada@alias.example.com"},
			OrgUnitPath:                "/Engineering",
			IsArchived:                 true,
			IsAdmin:                    true,
			IsEnrolledIn2SV:            true,
			IsMailboxSetup:             true,
			IncludeInGlobalAddressList: true,
			RecoveryEmail:              "recovery@example.com",
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
	for _, want := range []string{
		"ID: 12345",
		"Given name: Ada",
		"Family name: Lovelace",
		"Archived: true",
		"Enrolled in 2SV: true",
		"Mailbox setup: true",
		"Recovery email: recovery@example.com",
		"Aliases: a@example.com",
		"Non-editable aliases: ada@alias.example.com",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
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

func TestPrintGroupsPassesListOptions(t *testing.T) {
	var got google.GroupListOptions
	r, _ := testRunnerWithConfig(t, fakeDirectory{
		groupListOpts: &got,
		groups:        []google.GroupInfo{{Email: "eng@example.com"}},
	})
	parsed, err := cli.Parse([]string{
		"print", "groups",
		"--limit", "30",
		"--domain", "example.org",
		"--user", "ada@example.com",
		"--query", "email:eng",
		"--sort", "email",
		"--order", "asc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got.Limit != 30 {
		t.Fatalf("Limit = %d", got.Limit)
	}
	if got.Domain != "example.org" {
		t.Fatalf("Domain = %q", got.Domain)
	}
	if got.UserKey != "ada@example.com" {
		t.Fatalf("UserKey = %q", got.UserKey)
	}
	if got.Query != "email:eng" {
		t.Fatalf("Query = %q", got.Query)
	}
	if got.OrderBy != "email" || got.SortOrder != "ASCENDING" {
		t.Fatalf("sort = %q %q", got.OrderBy, got.SortOrder)
	}
}

func TestPrintGroupsAcceptsLimitAll(t *testing.T) {
	var got google.GroupListOptions
	r, _ := testRunnerWithConfig(t, fakeDirectory{
		groupListOpts: &got,
		groups:        []google.GroupInfo{{Email: "eng@example.com"}},
	})
	parsed, err := cli.Parse([]string{"print", "groups", "--limit", "all"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !got.FetchAll {
		t.Fatal("FetchAll = false")
	}
	if got.Limit != 0 {
		t.Fatalf("Limit = %d", got.Limit)
	}
}

func TestPrintGroupsSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		groups: []google.GroupInfo{{
			Email:              "eng@example.com",
			DirectMembersCount: 7,
		}},
	})
	parsed, err := cli.Parse([]string{"print", "groups", "--fields", "email,directMembersCount", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Email,Direct Members\neng@example.com,7\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintDomainsSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		domains: []google.WorkspaceDomainInfo{{
			DomainName: "example.com",
			IsPrimary:  true,
		}},
	})
	parsed, err := cli.Parse([]string{"print", "domains", "--fields", "domainName,isPrimary", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Domain,Primary\nexample.com,true\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintDomainAliasesSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		dAliases: []google.DomainAliasInfo{{
			DomainAliasName:  "alias.example.com",
			ParentDomainName: "example.com",
		}},
	})
	parsed, err := cli.Parse([]string{"print", "domain-aliases", "--fields", "domainAliasName,parentDomainName", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Alias,Parent Domain\nalias.example.com,example.com\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintGroupsExportsToSheet(t *testing.T) {
	sheets := &fakeSheets{info: google.SheetInfo{
		Title:          "gws groups",
		SpreadsheetURL: "https://docs.google.com/spreadsheets/d/456",
	}}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		groups: []google.GroupInfo{{
			Email:              "eng@example.com",
			DirectMembersCount: 7,
		}},
	})
	r.Sheets = sheets
	parsed, err := cli.Parse([]string{"print", "groups", "--fields", "email,directMembersCount", "--sheet"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	wantRows := [][]string{
		{"Email", "Direct Members"},
		{"eng@example.com", "7"},
	}
	if !equalStringRows(sheets.gotRows, wantRows) {
		t.Fatalf("rows = %#v", sheets.gotRows)
	}
	if !strings.Contains(out.String(), "Sheet created: gws groups") || !strings.Contains(out.String(), sheets.info.SpreadsheetURL) {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintGroupMembersSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{{
			Email: "ada@example.com",
			Role:  "OWNER",
		}},
	})
	parsed, err := cli.Parse([]string{"print", "group-members", "eng@example.com", "--fields", "email,role", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Email,Role\nada@example.com,OWNER\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintOrgUnitsSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		ous: []google.OrgUnitInfo{{
			OrgUnitPath: "/Engineering",
			Name:        "Engineering",
		}},
	})
	parsed, err := cli.Parse([]string{"print", "ous", "--fields", "orgUnitPath,name", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Path,Name\n/Engineering,Engineering\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintUserAliasesSupportsCSVAndFields(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		aliases: []google.AliasInfo{{
			Alias:        "ada.alias@example.com",
			PrimaryEmail: "ada@example.com",
		}},
	})
	parsed, err := cli.Parse([]string{"print", "user-aliases", "ada@example.com", "--fields", "alias,primaryEmail", "--format", "csv"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "Alias,Primary Email\nada.alias@example.com,ada@example.com\n" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestPrintListRejectsInvalidFilterFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "user sort", args: []string{"print", "users", "--sort", "name"}, want: "--sort for users"},
		{name: "group sort", args: []string{"print", "groups", "--sort", "name"}, want: "--sort for groups"},
		{name: "order", args: []string{"print", "users", "--order", "sideways"}, want: "--order must be asc or desc"},
		{name: "show deleted", args: []string{"print", "users", "--show-deleted", "maybe"}, want: "--show-deleted must be true or false"},
		{name: "limit", args: []string{"print", "users", "--limit", "everything"}, want: "--limit must be a positive integer or 'all'"},
		{name: "user field", args: []string{"print", "users", "--fields", "bogus"}, want: "unknown user field"},
		{name: "group field", args: []string{"print", "groups", "--fields", "bogus"}, want: "unknown group field"},
		{name: "member field", args: []string{"print", "group-members", "eng@example.com", "--fields", "bogus"}, want: "unknown group member field"},
		{name: "org unit field", args: []string{"print", "ous", "--fields", "bogus"}, want: "unknown org unit field"},
		{name: "alias field", args: []string{"print", "user-aliases", "ada@example.com", "--fields", "bogus"}, want: "unknown alias field"},
		{name: "domain field", args: []string{"print", "domains", "--fields", "bogus"}, want: "unknown domain field"},
		{name: "domain alias field", args: []string{"print", "domain-aliases", "--fields", "bogus"}, want: "unknown domain alias field"},
		{name: "format", args: []string{"print", "users", "--format", "yaml"}, want: "--format must be text, csv, or json"},
		{name: "json format conflict", args: []string{"print", "users", "--json", "--format", "csv"}, want: "--json cannot be combined with --format=csv"},
		{name: "sheet json conflict", args: []string{"print", "users", "--sheet", "--json"}, want: "--sheet cannot be combined with --json"},
		{name: "sheet format conflict", args: []string{"print", "users", "--sheet", "--format", "csv"}, want: "--sheet cannot be combined with --format=csv"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := testRunnerWithConfig(t, fakeDirectory{})
			parsed, err := cli.Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			err = r.Run(context.Background(), parsed)
			if err == nil {
				t.Fatal("Run() error = nil")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v", err)
			}
		})
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
	captured := google.GroupInfo{}
	r := Runner{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Config: configPath,
		Directory: fakeDirectory{group: google.GroupInfo{
			Email: "eng@example.com",
			Name:  "Engineering Team",
		}, updatedGroup: &captured},
	}
	parsed, err := cli.Parse([]string{"update", "group", "eng@example.com", "--email", "engineering@example.com", "--name", "Engineering Team"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group updated: eng@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if captured.Email != "engineering@example.com" {
		t.Fatalf("updated email = %q", captured.Email)
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

func TestInfoGroupMemberUsesDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{member: google.MemberInfo{
		Email:            "ada@example.com",
		ID:               "member-id",
		Role:             "OWNER",
		Type:             "USER",
		Status:           "ACTIVE",
		DeliverySettings: "ALL_MAIL",
	}})
	parsed, err := cli.Parse([]string{"info", "group-member", "eng@example.com", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Email: ada@example.com",
		"ID: member-id",
		"Role: OWNER",
		"Delivery settings: ALL_MAIL",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
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

func TestUpdateGroupMemberUsesDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{member: google.MemberInfo{
		Email: "ada@example.com",
		Role:  "OWNER",
	}})
	parsed, err := cli.Parse([]string{"update", "group-member", "eng@example.com", "ada@example.com", "--role", "owner"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group member updated: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
	if !strings.Contains(out.String(), "Role: OWNER") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUpdateGroupMemberRejectsInvalidRole(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"update", "group-member", "eng@example.com", "ada@example.com", "--role", "bad"})
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

func TestSyncGroupMembersDryRun(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "MEMBER"},
			{Email: "grace@example.com", Role: "MEMBER"},
		},
	})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members", "ada@example.com,linus@example.com",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Group member sync plan: eng@example.com",
		"Would add: 1",
		"Would remove: 1",
		"Unchanged: 1",
		"Add members: linus@example.com",
		"Remove members: grace@example.com",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersAppliesChanges(t *testing.T) {
	added := []string{}
	removed := []string{}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "MEMBER"},
			{Email: "grace@example.com", Role: "MEMBER"},
		},
		member:         google.MemberInfo{Email: "linus@example.com"},
		addedMembers:   &added,
		removedMembers: &removed,
	})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members", "ada@example.com,linus@example.com",
		"--confirm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !equalStringSlices(added, []string{"linus@example.com"}) {
		t.Fatalf("added = %#v", added)
	}
	if !equalStringSlices(removed, []string{"grace@example.com"}) {
		t.Fatalf("removed = %#v", removed)
	}
	for _, want := range []string{
		"Group member sync complete: eng@example.com",
		"Added: 1",
		"Removed: 1",
		"Unchanged: 1",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersRoleAwareDryRun(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "OWNER"},
		},
	})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members", "ada@example.com",
		"--role", "MEMBER",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Group member sync plan: eng@example.com",
		"Role: MEMBER",
		"Would add: 0",
		"Would remove: 0",
		"Would update roles: 1",
		"Unchanged: 0",
		"Update roles: ada@example.com (OWNER -> MEMBER)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersRoleAwareAppliesRoleChanges(t *testing.T) {
	updated := []string{}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "OWNER"},
		},
		member:         google.MemberInfo{Email: "ada@example.com", Role: "MEMBER"},
		updatedMembers: &updated,
	})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members", "ada@example.com",
		"--role", "MEMBER",
		"--confirm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !equalStringSlices(updated, []string{"ada@example.com:MEMBER"}) {
		t.Fatalf("updated = %#v", updated)
	}
	for _, want := range []string{
		"Group member sync complete: eng@example.com",
		"Role: MEMBER",
		"Added: 0",
		"Removed: 0",
		"Updated roles: 1",
		"Unchanged: 0",
		"Updated role members: ada@example.com (OWNER -> MEMBER)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersIgnoreRoleRemovesAnyExtraRole(t *testing.T) {
	removed := []string{}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "owner@example.com", Role: "OWNER"},
			{Email: "member@example.com", Role: "MEMBER"},
		},
		removedMembers: &removed,
	})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members", "member@example.com",
		"--ignore-role",
		"--confirm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !equalStringSlices(removed, []string{"owner@example.com"}) {
		t.Fatalf("removed = %#v", removed)
	}
	for _, want := range []string{
		"Group member sync complete: eng@example.com",
		"Mode: ignore role",
		"Added: 0",
		"Removed: 1",
		"Updated roles: 0",
		"Unchanged: 1",
		"Removed members: owner@example.com",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersRejectsIgnoreRoleWithExplicitRole(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members", "ada@example.com",
		"--role", "OWNER",
		"--ignore-role",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--ignore-role cannot be combined with --role") {
		t.Fatalf("error = %v", err)
	}
}

func TestSyncGroupMembersStructuredCSVAppliesMultiRoleChanges(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "members.csv")
	if err := os.WriteFile(csvPath, []byte("email,role\nada@example.com,OWNER\nlinus@example.com,MANAGER\n"), 0600); err != nil {
		t.Fatal(err)
	}
	added := []string{}
	updated := []string{}
	removed := []string{}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "MEMBER"},
			{Email: "grace@example.com", Role: "MEMBER"},
		},
		addedMemberRoles: &added,
		updatedMembers:   &updated,
		removedMembers:   &removed,
		member:           google.MemberInfo{},
	})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members-csv", csvPath,
		"--confirm",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !equalStringSlices(added, []string{"linus@example.com:MANAGER"}) {
		t.Fatalf("added = %#v", added)
	}
	if !equalStringSlices(updated, []string{"ada@example.com:OWNER"}) {
		t.Fatalf("updated = %#v", updated)
	}
	if !equalStringSlices(removed, []string{"grace@example.com"}) {
		t.Fatalf("removed = %#v", removed)
	}
	for _, want := range []string{
		"Group member sync complete: eng@example.com",
		"Mode: explicit roles",
		"Added: 1",
		"Removed: 1",
		"Updated roles: 1",
		"Added members: linus@example.com (MANAGER)",
		"Removed members: grace@example.com (MEMBER)",
		"Updated role members: ada@example.com (MEMBER -> OWNER)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersStructuredSheetDryRun(t *testing.T) {
	sheets := &fakeSheets{
		readRows: [][]string{
			{"email", "role"},
			{"ada@example.com", "OWNER"},
			{"grace@example.com", "MEMBER"},
		},
	}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		members: []google.MemberInfo{
			{Email: "ada@example.com", Role: "OWNER"},
			{Email: "linus@example.com", Role: "MANAGER"},
		},
	})
	r.Sheets = sheets
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members-sheet", "https://docs.google.com/spreadsheets/d/abc123/edit#gid=0",
		"--sheet-range", "Members!A:B",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if sheets.gotReadSpreadsheetID != "abc123" {
		t.Fatalf("spreadsheetID = %q", sheets.gotReadSpreadsheetID)
	}
	if sheets.gotReadRange != "Members!A:B" {
		t.Fatalf("range = %q", sheets.gotReadRange)
	}
	for _, want := range []string{
		"Group member sync plan: eng@example.com",
		"Mode: explicit roles",
		"Would add: 1",
		"Would remove: 1",
		"Would update roles: 0",
		"Unchanged: 1",
		"Add members: grace@example.com (MEMBER)",
		"Remove members: linus@example.com (MANAGER)",
		"Unchanged members: ada@example.com (OWNER)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestSyncGroupMembersStructuredInputRejectsRoleFlag(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "members.csv")
	if err := os.WriteFile(csvPath, []byte("email,role\nada@example.com,OWNER\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{
		"sync", "group-members", "eng@example.com",
		"--members-csv", csvPath,
		"--role", "OWNER",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--role cannot be combined with --members-csv or --members-sheet") {
		t.Fatalf("error = %v", err)
	}
}

func TestSyncGroupMembersRequiresConfirm(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"sync", "group-members", "eng@example.com", "--members", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "sync group-members requires --confirm") {
		t.Fatalf("error = %v", err)
	}
}

func TestSyncGroupMembersRequiresDesiredMembers(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"sync", "group-members", "eng@example.com", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "requires --members, --members-file, --members-csv, or --members-sheet") {
		t.Fatalf("error = %v", err)
	}
}

func TestUserAliasCommandsUseDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		alias:   google.AliasInfo{Alias: "ada.alias@example.com", PrimaryEmail: "ada@example.com", ID: "alias-id"},
		aliases: []google.AliasInfo{{Alias: "ada.alias@example.com", PrimaryEmail: "ada@example.com", ID: "alias-id"}},
	})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "print", args: []string{"print", "user-aliases", "ada@example.com"}, want: "ada.alias@example.com"},
		{name: "create", args: []string{"create", "user-alias", "ada@example.com", "ada.alias@example.com"}, want: "User alias created: ada.alias@example.com"},
		{name: "delete", args: []string{"delete", "user-alias", "ada@example.com", "ada.alias@example.com", "--confirm"}, want: "User alias deleted: ada.alias@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out.Reset()
			parsed, err := cli.Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Run(context.Background(), parsed); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

func TestGroupAliasCommandsUseDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{
		alias:   google.AliasInfo{Alias: "eng.alias@example.com", PrimaryEmail: "eng@example.com", ID: "alias-id"},
		aliases: []google.AliasInfo{{Alias: "eng.alias@example.com", PrimaryEmail: "eng@example.com", ID: "alias-id"}},
	})
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{name: "print", args: []string{"print", "group-aliases", "eng@example.com"}, want: "eng.alias@example.com"},
		{name: "create", args: []string{"create", "group-alias", "eng@example.com", "eng.alias@example.com"}, want: "Group alias created: eng.alias@example.com"},
		{name: "delete", args: []string{"delete", "group-alias", "eng@example.com", "eng.alias@example.com", "--confirm"}, want: "Group alias deleted: eng.alias@example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out.Reset()
			parsed, err := cli.Parse(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			if err := r.Run(context.Background(), parsed); err != nil {
				t.Fatalf("Run() error = %v", err)
			}
			if !strings.Contains(out.String(), tc.want) {
				t.Fatalf("output = %q", out.String())
			}
		})
	}
}

func TestDeleteAliasRequiresConfirm(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"delete", "user-alias", "ada@example.com", "ada.alias@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "delete user-alias requires --confirm") {
		t.Fatalf("error = %v", err)
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

func TestCreateOrgUnitUsesDirectoryClient(t *testing.T) {
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
		}},
	}
	parsed, err := cli.Parse([]string{"create", "ou", "--name", "Engineering", "--parent", "/", "--description", "Builders"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Org unit created: /Engineering") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestCreateOrgUnitRequiresName(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"create", "ou"})
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

func TestUpdateOrgUnitUsesDirectoryClient(t *testing.T) {
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
		}},
	}
	parsed, err := cli.Parse([]string{"update", "ou", "/Engineering", "--description", "Builders"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Org unit updated: /Engineering") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestUpdateOrgUnitRequiresChange(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"update", "ou", "/Engineering"})
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

func TestExplainAPICommandFailureTranslatesScopeErrors(t *testing.T) {
	profile := config.Profile{
		AuthMethod: auth.MethodOAuth,
		Scopes:     []string{"https://www.googleapis.com/auth/admin.directory.user"},
	}
	err := explainAPICommandFailure(profile, "print users", &googleapi.Error{
		Code:    403,
		Message: "Request had insufficient authentication scopes.",
		Errors: []googleapi.ErrorItem{
			{Reason: "insufficientPermissions", Message: "Insufficient Permission"},
		},
	})
	if !strings.Contains(err.Error(), "does not have enough scopes") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "rerun `gws setup`") {
		t.Fatalf("error = %v", err)
	}
}

func TestExplainAPICommandFailureTranslatesDisabledAPI(t *testing.T) {
	profile := config.Profile{AuthMethod: auth.MethodOAuth}
	err := explainAPICommandFailure(profile, "print users", &googleapi.Error{
		Code:    403,
		Message: "Admin SDK API has not been used in project 123 before or it is disabled.",
		Errors:  []googleapi.ErrorItem{{Reason: "accessNotConfigured"}},
	})
	if !strings.Contains(err.Error(), "Enable `Admin SDK API`") {
		t.Fatalf("error = %v", err)
	}
}

func TestExplainAPICommandFailureTranslatesDelegationErrors(t *testing.T) {
	profile := config.Profile{
		AuthMethod:   auth.MethodServiceAccount,
		AdminSubject: "admin@example.com",
	}
	err := explainAPICommandFailure(profile, "print users", &googleapi.Error{
		Code:    403,
		Message: "Not Authorized to access this resource/api",
	})
	if !strings.Contains(err.Error(), "domain-wide delegation") {
		t.Fatalf("error = %v", err)
	}
	if !strings.Contains(err.Error(), "admin@example.com") {
		t.Fatalf("error = %v", err)
	}
}

func TestExplainAPICommandFailureTranslatesInvalidQueryErrors(t *testing.T) {
	profile := config.Profile{AuthMethod: auth.MethodOAuth}
	err := explainAPICommandFailure(profile, "print users", &googleapi.Error{
		Code:    400,
		Message: "Invalid Input: query",
		Errors:  []googleapi.ErrorItem{{Reason: "invalid"}},
	})
	if !strings.Contains(err.Error(), "Check the query") {
		t.Fatalf("error = %v", err)
	}
}

func TestExplainSheetExportFailureTranslatesDisabledAPI(t *testing.T) {
	profile := config.Profile{AuthMethod: auth.MethodOAuth}
	err := explainSheetExportFailure(profile, "print users", &googleapi.Error{
		Code:    403,
		Message: "Google Sheets API has not been used in project 123 before or it is disabled.",
		Errors:  []googleapi.ErrorItem{{Reason: "SERVICE_DISABLED"}},
	})
	if !strings.Contains(err.Error(), "Enable `Google Sheets API`") {
		t.Fatalf("error = %v", err)
	}
}

func TestBatchRunDryRun(t *testing.T) {
	dir := t.TempDir()
	batchPath := filepath.Join(dir, "commands.txt")
	if err := os.WriteFile(batchPath, []byte("# comment\nversion\nprint users --limit 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "run", "--file", batchPath, "--workers", "3", "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Batch plan: 2 command(s)",
		"Workers: 3",
		"line 2: version",
		"line 3: print users --limit 1",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestBatchRunReportsFailures(t *testing.T) {
	dir := t.TempDir()
	batchPath := filepath.Join(dir, "commands.txt")
	if err := os.WriteFile(batchPath, []byte("version\nbogus command\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "run", "--file", batchPath, "--workers", "2"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	for _, want := range []string{
		"OK   line 1: version",
		"stdout:",
		"gws 0.1.0-dev",
		"FAIL line 2: bogus command",
		"Batch complete: 2 command(s), 1 failure(s), 2 worker(s)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestBatchRunCapturesPerCommandOutputInOrder(t *testing.T) {
	dir := t.TempDir()
	batchPath := filepath.Join(dir, "commands.txt")
	if err := os.WriteFile(batchPath, []byte("version\nhelp\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "run", "--file", batchPath, "--workers", "2"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := out.String()
	versionIndex := strings.Index(output, "OK   line 1: version")
	helpIndex := strings.Index(output, "OK   line 2: help")
	if versionIndex < 0 || helpIndex < 0 {
		t.Fatalf("batch output missing command status lines:\n%s", output)
	}
	if versionIndex > helpIndex {
		t.Fatalf("version output appeared after help output:\n%s", output)
	}
	if !strings.Contains(output, "    gws 0.1.0-dev") {
		t.Fatalf("output missing captured version stdout:\n%s", output)
	}
	if !strings.Contains(output, "    gws administers Google Workspace from the command line.") {
		t.Fatalf("output missing captured help stdout:\n%s", output)
	}
}

func TestBatchRunRequiresFile(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "run"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws batch run --file PATH") {
		t.Fatalf("error = %v", err)
	}
}

func TestBatchCSVRunDryRun(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "users.csv")
	if err := os.WriteFile(csvPath, []byte("email,orgUnit\nada@example.com,/Engineering\ngrace@example.com,/Sales\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{
		"batch", "csv",
		"--file", csvPath,
		"--command", `update user "{{email}}" --org-unit "{{orgUnit}}"`,
		"--workers", "4",
		"--dry-run",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Batch plan: 2 command(s)",
		"Workers: 4",
		"line 2: update user ada@example.com --org-unit /Engineering",
		"line 3: update user grace@example.com --org-unit /Sales",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestBatchCSVRequiresTemplate(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "users.csv")
	if err := os.WriteFile(csvPath, []byte("email\nada@example.com\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "csv", "--file", csvPath})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Usage: gws batch csv --file PATH --command TEMPLATE") {
		t.Fatalf("error = %v", err)
	}
}

func TestBatchRunRejectsInvalidTimeout(t *testing.T) {
	dir := t.TempDir()
	batchPath := filepath.Join(dir, "commands.txt")
	if err := os.WriteFile(batchPath, []byte("version\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "run", "--file", batchPath, "--timeout", "soon"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--timeout must be a positive duration") {
		t.Fatalf("error = %v", err)
	}
}

func TestBatchTemplateShowsWorkflow(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "template", "user-update"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Workflow: user-update",
		"Headers: email,orgUnit,recoveryEmail,recoveryPhone",
		`gws batch csv --file ./user-update.csv --command 'update user "{{email}}"`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestBatchTemplateExample(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "template", "group-create", "--example"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Example CSV:",
		"email,name,description",
		"eng@example.com,Engineering,Primary engineering discussion group",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestBatchTemplateNewWorkflow(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "template", "group-member-add"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Workflow: group-member-add",
		"Headers: groupEmail,memberEmail,role",
		`gws batch csv --file ./group-member-add.csv --command 'add group-member "{{groupEmail}}" "{{memberEmail}}" --role "{{role}}"'`,
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
}

func TestBatchTemplateRequiresWorkflow(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"batch", "template"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "Available workflows:") {
		t.Fatalf("error = %v", err)
	}
}

func TestGmailDelegateCommandsRequireServiceAccount(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"print", "gmail-delegates", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "requires a service account profile with domain-wide delegation") {
		t.Fatalf("error = %v", err)
	}
}

func TestGmailDelegateCommandsUseGmailClient(t *testing.T) {
	dir := t.TempDir()
	credentials := filepath.Join(dir, "service-account.json")
	if err := os.WriteFile(credentials, []byte(`{"type":"service_account","client_email":"svc@example.iam.gserviceaccount.com"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := config.Empty()
	cfg.Profiles["default"] = config.Profile{
		Domain:          "example.com",
		AdminSubject:    "admin@example.com",
		CredentialsFile: credentials,
		AuthMethod:      "service_account",
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
		Gmail: fakeGmail{
			delegate: google.DelegateInfo{
				DelegateEmail:      "grace@example.com",
				VerificationStatus: "accepted",
			},
			delegates: []google.DelegateInfo{
				{DelegateEmail: "grace@example.com", VerificationStatus: "accepted"},
				{DelegateEmail: "linus@example.com", VerificationStatus: "accepted"},
			},
		},
		Directory: fakeDirectory{},
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{args: []string{"print", "gmail-delegates", "ada@example.com"}, want: "grace@example.com"},
		{args: []string{"info", "gmail-delegate", "ada@example.com", "grace@example.com"}, want: "Verification status: accepted"},
		{args: []string{"create", "gmail-delegate", "ada@example.com", "grace@example.com"}, want: "Gmail delegate created: grace@example.com"},
		{args: []string{"delete", "gmail-delegate", "ada@example.com", "grace@example.com", "--confirm"}, want: "Gmail delegate deleted: grace@example.com"},
	} {
		out.Reset()
		parsed, err := cli.Parse(tc.args)
		if err != nil {
			t.Fatal(err)
		}
		if err := r.Run(context.Background(), parsed); err != nil {
			t.Fatalf("Run(%v) error = %v", tc.args, err)
		}
		if !strings.Contains(out.String(), tc.want) {
			t.Fatalf("output missing %q:\n%s", tc.want, out.String())
		}
	}
}

func TestBatchRunPerCommandTimeout(t *testing.T) {
	dir := t.TempDir()
	batchPath := filepath.Join(dir, "commands.txt")
	if err := os.WriteFile(batchPath, []byte("print users --limit 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r, out := testRunnerWithConfig(t, fakeDirectory{
		usersDelay: 50 * time.Millisecond,
		users:      []google.UserInfo{{PrimaryEmail: "ada@example.com"}},
	})
	parsed, err := cli.Parse([]string{"batch", "run", "--file", batchPath, "--timeout", "5ms"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	output := out.String()
	if !strings.Contains(output, "FAIL line 1: print users --limit 1") {
		t.Fatalf("output = %s", output)
	}
	if !strings.Contains(output, "context deadline exceeded") {
		t.Fatalf("output = %s", output)
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
	captured := google.UserUpdate{}
	r := Runner{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Config: configPath,
		Directory: fakeDirectory{user: google.UserInfo{
			PrimaryEmail:  "ada@example.com",
			Name:          "Ada Byron",
			OrgUnitPath:   "/Engineering",
			RecoveryEmail: "recover@example.com",
			RecoveryPhone: "+15551234567",
			IsArchived:    true,
		}, updatedUser: &captured},
	}
	parsed, err := cli.Parse([]string{
		"update", "user", "ada@example.com",
		"--given-name", "Ada",
		"--family-name", "Byron",
		"--org-unit", "Engineering",
		"--recovery-email", "recover@example.com",
		"--recovery-phone", "+15551234567",
		"--change-password-at-next-login", "true",
		"--archived", "true",
		"--include-in-global-address-list", "false",
		"--phones-json", `[{"value":"+15551234567","type":"work"}]`,
	})
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
	if captured.OrgUnitPath != "/Engineering" {
		t.Fatalf("OrgUnitPath = %q", captured.OrgUnitPath)
	}
	if captured.RecoveryEmail == nil || *captured.RecoveryEmail != "recover@example.com" {
		t.Fatalf("RecoveryEmail = %#v", captured.RecoveryEmail)
	}
	if captured.RecoveryPhone == nil || *captured.RecoveryPhone != "+15551234567" {
		t.Fatalf("RecoveryPhone = %#v", captured.RecoveryPhone)
	}
	if captured.ChangePasswordAtNextLogin == nil || !*captured.ChangePasswordAtNextLogin {
		t.Fatalf("ChangePasswordAtNextLogin = %#v", captured.ChangePasswordAtNextLogin)
	}
	if captured.Archived == nil || !*captured.Archived {
		t.Fatalf("Archived = %#v", captured.Archived)
	}
	if captured.IncludeInGlobalAddressList == nil || *captured.IncludeInGlobalAddressList {
		t.Fatalf("IncludeInGlobalAddressList = %#v", captured.IncludeInGlobalAddressList)
	}
	if captured.Phones == nil {
		t.Fatal("Phones = nil")
	}
}

func TestUpdateUserRejectsInvalidJSONField(t *testing.T) {
	r, _ := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"update", "user", "ada@example.com", "--phones-json", "{"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "--phones-json must be valid JSON") {
		t.Fatalf("error = %v", err)
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

func TestDeleteUserRequiresConfirm(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    &out,
		Config:    filepath.Join(t.TempDir(), "config.json"),
		Directory: fakeDirectory{},
	}
	parsed, err := cli.Parse([]string{"delete", "user", "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	err = r.Run(context.Background(), parsed)
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if !strings.Contains(err.Error(), "delete user requires --confirm") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeleteUserUsesDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"delete", "user", "ada@example.com", "--confirm"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "User deleted: ada@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDeleteGroupUsesDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"delete", "group", "eng@example.com", "--confirm"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Group deleted: eng@example.com") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDeleteOrgUnitUsesDirectoryClient(t *testing.T) {
	r, out := testRunnerWithConfig(t, fakeDirectory{})
	parsed, err := cli.Parse([]string{"delete", "ou", "Engineering", "--confirm"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(out.String(), "Org unit deleted: /Engineering") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestHelpIncludesRegisteredCommands(t *testing.T) {
	var out bytes.Buffer
	r := Runner{
		Stdin:  strings.NewReader(""),
		Stdout: &out,
		Stderr: &bytes.Buffer{},
		Config: filepath.Join(t.TempDir(), "config.json"),
	}
	parsed, err := cli.Parse([]string{"help"})
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Run(context.Background(), parsed); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	for _, want := range []string{
		"Usage:\n  gws <command> [flags]",
		"Getting started:",
		"Configuration and auth:",
		"Users:",
		"Groups:",
		"Org units:",
		"gws version",
		"gws setup [--profile default]",
		"gws print user-aliases user@example.com [--fields alias,primaryEmail] [--format text|csv|json] [--sheet] [--json]",
		"gws print users [--limit 100|all] [--domain example.com] [--org-unit /PATH]",
		"gws print groups [--limit 100|all] [--domain example.com] [--user user@example.com]",
		"gws print group-members group@example.com [--limit 100] [--fields email,role,type,status]",
		"gws print domains [--fields domainName,isPrimary]",
		"gws batch run --file PATH [--workers N] [--timeout 30s]",
		"gws batch csv --file PATH --command TEMPLATE [--workers N] [--timeout 30s]",
		"gws batch template WORKFLOW [--example]",
		"gws print gmail-delegates user@example.com [--json]",
		"gws create gmail-delegate user@example.com delegate@example.com [--json]",
		"gws sync group-members group@example.com (--members user1@example.com,user2@example.com | --members-file PATH | --members-csv PATH | --members-sheet SHEET_ID_OR_URL",
		"gws info group-member group@example.com member@example.com [--json]",
		"gws update group-member group@example.com member@example.com --role OWNER|MANAGER|MEMBER [--json]",
		"gws delete user user@example.com --confirm",
		"gws update ou /PATH [--name NAME]",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help output missing %q:\n%s", want, out.String())
		}
	}
}

func testRunnerWithConfig(t *testing.T, directory fakeDirectory) (Runner, *bytes.Buffer) {
	t.Helper()
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
	out := &bytes.Buffer{}
	return Runner{
		Stdin:     strings.NewReader(""),
		Stdout:    out,
		Config:    configPath,
		Directory: directory,
		Gmail:     fakeGmail{},
	}, out
}

func assertAlignedColumn(t *testing.T, table, header string, values []string) {
	t.Helper()
	lines := strings.Split(strings.TrimSuffix(table, "\n"), "\n")
	if len(lines) < len(values)+1 {
		t.Fatalf("table has %d lines, want at least %d:\n%s", len(lines), len(values)+1, table)
	}
	want := strings.Index(lines[0], header)
	if want < 0 {
		t.Fatalf("header %q not found:\n%s", header, table)
	}
	for i, value := range values {
		got := strings.Index(lines[i+1], value)
		if got != want {
			t.Fatalf("column %q row %d starts at %d, want %d:\n%s", header, i+1, got, want, table)
		}
	}
}

func equalStringRows(got, want [][]string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if len(got[i]) != len(want[i]) {
			return false
		}
		for j := range got[i] {
			if got[i][j] != want[i][j] {
				return false
			}
		}
	}
	return true
}

func equalStringSlices(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
