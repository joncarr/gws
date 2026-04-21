package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/batch"
	"github.com/joncarr/gws/internal/cli"
	"github.com/joncarr/gws/internal/config"
	"github.com/joncarr/gws/internal/google"
	"github.com/joncarr/gws/internal/output"
	"google.golang.org/api/googleapi"
)

const Version = "0.1.0-dev"

type Runner struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Config    string
	Directory google.DirectoryClient
	Gmail     google.GmailClient
	Sheets    google.SheetService
}

type batchOutput struct {
	stdout string
	stderr string
}

type batchTemplate struct {
	name        string
	description string
	headers     []string
	command     string
	exampleRows [][]string
}

type commandSpec struct {
	key     string
	group   string
	usage   string
	handler func(context.Context, cli.Parsed) error
}

type outputFormat string

const (
	formatText outputFormat = "text"
	formatJSON outputFormat = "json"
	formatCSV  outputFormat = "csv"
)

type userFieldSpec struct {
	key    string
	header string
	text   func(google.UserInfo) string
	value  func(google.UserInfo) any
}

type groupFieldSpec struct {
	key    string
	header string
	text   func(google.GroupInfo) string
	value  func(google.GroupInfo) any
}

type memberFieldSpec struct {
	key    string
	header string
	text   func(google.MemberInfo) string
	value  func(google.MemberInfo) any
}

type orgUnitFieldSpec struct {
	key    string
	header string
	text   func(google.OrgUnitInfo) string
	value  func(google.OrgUnitInfo) any
}

type aliasFieldSpec struct {
	key    string
	header string
	text   func(google.AliasInfo) string
	value  func(google.AliasInfo) any
}

type domainFieldSpec struct {
	key    string
	header string
	text   func(google.WorkspaceDomainInfo) string
	value  func(google.WorkspaceDomainInfo) any
}

type domainAliasFieldSpec struct {
	key    string
	header string
	text   func(google.DomainAliasInfo) string
	value  func(google.DomainAliasInfo) any
}

func (r Runner) Run(ctx context.Context, parsed cli.Parsed) error {
	key := commandKey(parsed.Positionals)
	if key == "" {
		r.help()
		return nil
	}
	for _, command := range r.commands() {
		if command.key == key {
			return command.handler(ctx, parsed)
		}
	}
	return fmt.Errorf("unknown command %q\n\nRun `gws help` to see supported commands.", strings.Join(parsed.Positionals, " "))
}

func (r Runner) commands() []commandSpec {
	return []commandSpec{
		{
			key:   "version",
			group: "Getting started",
			usage: "gws version",
			handler: func(context.Context, cli.Parsed) error {
				output.New(r.Stdout).Printf("gws %s\n", Version)
				return nil
			},
		},
		{
			key:   "help",
			group: "Getting started",
			usage: "gws help",
			handler: func(context.Context, cli.Parsed) error {
				r.help()
				return nil
			},
		},
		{
			key:     "setup",
			group:   "Getting started",
			usage:   "gws setup [--profile default] [--domain example.com] [--admin admin@example.com] [--credentials client.json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.setup(ctx, parsed.Flags) },
		},
		{
			key:     "batch run",
			group:   "Batch",
			usage:   "gws batch run --file PATH [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.batchRun(ctx, parsed.Flags) },
		},
		{
			key:     "batch csv",
			group:   "Batch",
			usage:   "gws batch csv --file PATH --command TEMPLATE [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.batchCSV(ctx, parsed.Flags) },
		},
		{
			key:   "batch template",
			group: "Batch",
			usage: "gws batch template WORKFLOW [--example]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.batchTemplate(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:     "config show",
			group:   "Configuration and auth",
			usage:   "gws config show [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.configShow(parsed.Flags) },
		},
		{
			key:     "auth status",
			group:   "Configuration and auth",
			usage:   "gws auth status [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.authStatus(ctx, parsed.Flags) },
		},
		{
			key:     "check connection",
			group:   "Configuration and auth",
			usage:   "gws check connection",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.checkConnection(ctx, parsed.Flags) },
		},
		{
			key:   "info domain",
			group: "Domain",
			usage: "gws info domain [example.com]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoDomain(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "info domain-alias",
			group: "Domain",
			usage: "gws info domain-alias alias.example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoDomainAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "info group",
			group: "Groups",
			usage: "gws info group group@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoGroup(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "info group-member",
			group: "Groups",
			usage: "gws info group-member group@example.com member@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoGroupMember(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "info gmail-delegate",
			group: "Gmail",
			usage: "gws info gmail-delegate user@example.com delegate@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoGmailDelegate(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "info ou",
			group: "Org units",
			usage: "gws info ou /Engineering [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoOrgUnit(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "info user",
			group: "Users",
			usage: "gws info user user@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.infoUser(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:     "print groups",
			group:   "Groups",
			usage:   "gws print groups [--limit 100|all] [--domain example.com] [--user user@example.com] [--query QUERY] [--sort email] [--order asc|desc] [--fields email,name] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.printGroups(ctx, parsed.Flags) },
		},
		{
			key:     "print domains",
			group:   "Domain",
			usage:   "gws print domains [--fields domainName,isPrimary] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.printDomains(ctx, parsed.Flags) },
		},
		{
			key:     "print domain-aliases",
			group:   "Domain",
			usage:   "gws print domain-aliases [--fields domainAliasName,parentDomainName] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.printDomainAliases(ctx, parsed.Flags) },
		},
		{
			key:   "print group-aliases",
			group: "Groups",
			usage: "gws print group-aliases group@example.com [--fields alias,primaryEmail] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.printGroupAliases(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "print group-members",
			group: "Groups",
			usage: "gws print group-members group@example.com [--limit 100] [--fields email,role,type,status] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.printGroupMembers(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "print gmail-delegates",
			group: "Gmail",
			usage: "gws print gmail-delegates user@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.printGmailDelegates(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:     "print ous",
			group:   "Org units",
			usage:   "gws print ous [--fields orgUnitPath,name,parentOrgUnitPath] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.printOrgUnits(ctx, parsed.Flags) },
		},
		{
			key:     "print users",
			group:   "Users",
			usage:   "gws print users [--limit 100|all] [--domain example.com] [--org-unit /PATH] [--query QUERY] [--show-deleted] [--sort email|familyName|givenName] [--order asc|desc] [--fields primaryEmail,name] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.printUsers(ctx, parsed.Flags) },
		},
		{
			key:   "print user-aliases",
			group: "Users",
			usage: "gws print user-aliases user@example.com [--fields alias,primaryEmail] [--format text|csv|json] [--sheet] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.printUserAliases(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "add group-member",
			group: "Groups",
			usage: "gws add group-member group@example.com user@example.com [--role MEMBER] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.addGroupMember(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "create group",
			group: "Groups",
			usage: "gws create group group@example.com --name NAME [--description TEXT] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createGroup(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "create domain",
			group: "Domain",
			usage: "gws create domain example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createDomain(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "create domain-alias",
			group: "Domain",
			usage: "gws create domain-alias alias.example.com --parent example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createDomainAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "create group-alias",
			group: "Groups",
			usage: "gws create group-alias group@example.com alias@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createGroupAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "create gmail-delegate",
			group: "Gmail",
			usage: "gws create gmail-delegate user@example.com delegate@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createGmailDelegate(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:     "create ou",
			group:   "Org units",
			usage:   "gws create ou --name NAME --parent /PATH [--description TEXT] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error { return r.createOrgUnit(ctx, parsed.Flags) },
		},
		{
			key:   "create user",
			group: "Users",
			usage: "gws create user user@example.com --given-name NAME --family-name NAME --password-file PATH [--org-unit /PATH] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createUser(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "create user-alias",
			group: "Users",
			usage: "gws create user-alias user@example.com alias@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.createUserAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete group",
			group: "Groups",
			usage: "gws delete group group@example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteGroup(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete domain",
			group: "Domain",
			usage: "gws delete domain example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteDomain(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete domain-alias",
			group: "Domain",
			usage: "gws delete domain-alias alias.example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteDomainAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete group-alias",
			group: "Groups",
			usage: "gws delete group-alias group@example.com alias@example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteGroupAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete gmail-delegate",
			group: "Gmail",
			usage: "gws delete gmail-delegate user@example.com delegate@example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteGmailDelegate(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete ou",
			group: "Org units",
			usage: "gws delete ou /PATH --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteOrgUnit(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete user-alias",
			group: "Users",
			usage: "gws delete user-alias user@example.com alias@example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteUserAlias(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "delete user",
			group: "Users",
			usage: "gws delete user user@example.com --confirm",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.deleteUser(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "remove group-member",
			group: "Groups",
			usage: "gws remove group-member group@example.com user@example.com",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.removeGroupMember(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "sync group-members",
			group: "Groups",
			usage: "gws sync group-members group@example.com (--members user1@example.com,user2@example.com | --members-file PATH | --members-csv PATH | --members-sheet SHEET_ID_OR_URL [--sheet-range RANGE]) [--role MEMBER] [--ignore-role] [--dry-run] [--confirm]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.syncGroupMembers(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "suspend user",
			group: "Users",
			usage: "gws suspend user user@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.setUserSuspended(ctx, parsed.Positionals, parsed.Flags, true)
			},
		},
		{
			key:   "make admin",
			group: "Users",
			usage: "gws make admin user@example.com",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.setUserAdmin(ctx, parsed.Positionals, parsed.Flags, true)
			},
		},
		{
			key:   "revoke admin",
			group: "Users",
			usage: "gws revoke admin user@example.com",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.setUserAdmin(ctx, parsed.Positionals, parsed.Flags, false)
			},
		},
		{
			key:   "unsuspend user",
			group: "Users",
			usage: "gws unsuspend user user@example.com [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.setUserSuspended(ctx, parsed.Positionals, parsed.Flags, false)
			},
		},
		{
			key:   "update group",
			group: "Groups",
			usage: "gws update group group@example.com [--email new-group@example.com] [--name NAME] [--description TEXT] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.updateGroup(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "update group-member",
			group: "Groups",
			usage: "gws update group-member group@example.com member@example.com --role OWNER|MANAGER|MEMBER [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.updateGroupMember(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "update ou",
			group: "Org units",
			usage: "gws update ou /PATH [--name NAME] [--parent /PATH] [--description TEXT] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.updateOrgUnit(ctx, parsed.Positionals, parsed.Flags)
			},
		},
		{
			key:   "update user",
			group: "Users",
			usage: "gws update user user@example.com [--given-name NAME] [--family-name NAME] [--org-unit /PATH] [--recovery-email addr@example.com] [--recovery-phone +15551234567] [--change-password-at-next-login true|false] [--archived true|false] [--include-in-global-address-list true|false] [--phones-json JSON] [--addresses-json JSON] [--organizations-json JSON] [--locations-json JSON] [--relations-json JSON] [--external-ids-json JSON] [--json]",
			handler: func(ctx context.Context, parsed cli.Parsed) error {
				return r.updateUser(ctx, parsed.Positionals, parsed.Flags)
			},
		},
	}
}

func commandKey(args []string) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) == 1 {
		return args[0]
	}
	return args[0] + " " + args[1]
}

func (r Runner) help() {
	w := output.New(r.Stdout)
	w.Println("gws administers Google Workspace from the command line.")
	w.Println("")
	w.Println("Usage:")
	w.Println("  gws <command> [flags]")
	commands := r.commands()
	for _, group := range []string{
		"Getting started",
		"Configuration and auth",
		"Batch",
		"Domain",
		"Users",
		"Groups",
		"Gmail",
		"Org units",
	} {
		w.Println("")
		w.Printf("%s:\n", group)
		for _, command := range commands {
			if command.group == group {
				w.Printf("  %s\n", command.usage)
			}
		}
	}
	w.Println("")
	w.Println("Setup guides you through credentials, scopes, authorization, saved files, and a real Admin SDK validation.")
}

func (r Runner) setup(ctx context.Context, flags map[string]string) error {
	reader := bufio.NewReader(r.Stdin)
	w := output.New(r.Stdout)
	configPath, err := r.configPath()
	if err != nil {
		return err
	}
	w.Println("gws setup")
	w.Println("")
	w.Println("Step 1 of 5: Explain requirements")
	w.Println("This setup connects gws to Google Workspace in small, visible steps.")
	w.Println("You need a Google Cloud project with the Admin SDK API enabled.")
	w.Println("If you plan to manage Gmail delegates, also enable the Gmail API and use a service account with domain-wide delegation.")
	w.Println("You can use either a Desktop OAuth client JSON or a service account JSON configured for domain-wide delegation.")
	w.Println("")
	w.Println("Required scopes for the current command set:")
	for _, scope := range auth.RequiredScopes {
		w.Printf("  %s\n", scope)
	}
	w.Println("")
	w.Println("Step 2 of 5: Collect profile details")
	profileName, err := valueOrPrompt(reader, w, flags["profile"], "Profile name", config.DefaultProfileName)
	if err != nil {
		return err
	}
	domain, err := valueOrPrompt(reader, w, flags["domain"], "Primary Workspace domain", "")
	if err != nil {
		return err
	}
	admin, err := valueOrPrompt(reader, w, flags["admin"], "Workspace admin email for validation", "")
	if err != nil {
		return err
	}
	credentials, err := valueOrPrompt(reader, w, flags["credentials"], "Path to OAuth client or service account JSON", "")
	if err != nil {
		return err
	}
	if domain == "" {
		return errors.New("domain is required; rerun setup with --domain example.com")
	}
	if admin == "" {
		return errors.New("admin subject is required; rerun setup with --admin admin@example.com")
	}
	if credentials == "" {
		return errors.New("credentials file is required; rerun setup with --credentials /path/to/client.json")
	}
	w.Println("")
	w.Println("Step 3 of 5: Inspect credentials")
	credInfo, err := auth.ValidateCredentialsFile(credentials)
	if err != nil {
		return fmt.Errorf("credentials check failed: %w\n\nDownload a Desktop OAuth client JSON or service account JSON from Google Cloud Console and pass it with --credentials.", err)
	}
	w.Println("")
	w.Printf("Credentials detected: %s\n", credInfo.Type)
	if credInfo.ClientID != "" {
		w.Printf("Credential client ID: %s\n", credInfo.ClientID)
	}
	if credInfo.Type == auth.MethodServiceAccount {
		w.Println("Service account setup requires domain-wide delegation in the Google Admin console.")
		w.Println("Authorize the service account client ID for the scope shown above, then gws will impersonate the admin subject for validation.")
	} else {
		w.Println("OAuth setup will open a local callback URL and ask Google for an offline token.")
		w.Println("Sign in as a Workspace admin that can read domain/customer information.")
	}

	w.Println("")
	w.Println("Step 4 of 5: Save local configuration")
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	profile, err := config.NewProfile(profileName)
	if err != nil {
		return err
	}
	profile.Domain = domain
	profile.AdminSubject = admin
	profile.CredentialsFile = credentials
	profile.AuthMethod = auth.AuthMethod(credInfo)
	if profile.AuthMethod == auth.MethodServiceAccount {
		profile.TokenFile = ""
	}
	cfg.ActiveProfile = profileName
	cfg.Profiles[profileName] = profile
	if err := config.Save(configPath, cfg); err != nil {
		return err
	}

	if auth.IsOAuth(credInfo) {
		if flags["skip-auth"] == "true" {
			w.Println("")
			w.Println("OAuth authorization skipped by --skip-auth.")
		} else if err := auth.RunOAuthLocalFlow(ctx, credentials, profile.Scopes, profile.TokenFile, r.Stdout); err != nil {
			return fmt.Errorf("OAuth setup failed: %w\n\nCheck that your OAuth consent screen is configured and that this OAuth client allows loopback redirects.", err)
		}
	}

	w.Println("")
	w.Println("Step 5 of 5: Validate Admin SDK access")
	w.Println("Setup saved:")
	w.Printf("  Config file: %s\n", configPath)
	w.Printf("  Active profile: %s\n", profileName)
	w.Printf("  Domain: %s\n", domain)
	w.Printf("  Admin subject: %s\n", admin)
	w.Printf("  Credentials: %s (%s)\n", credentials, credInfo.Type)
	if profile.TokenFile != "" {
		w.Printf("  Token file: %s\n", profile.TokenFile)
	}
	w.Println("")
	if auth.IsOAuth(credInfo) && flags["skip-auth"] == "true" {
		w.Println("Next step:")
		w.Println("  Rerun `gws setup` without --skip-auth to authorize OAuth, then run `gws check connection`.")
		return nil
	}
	w.Println("Validating Admin SDK access...")
	info, err := r.validateGoogle(ctx)
	if err != nil {
		return explainValidationFailure(profile, err)
	}
	w.Println("")
	w.Println("Setup complete. Connection OK.")
	w.Printf("  Customer ID: %s\n", info.CustomerID)
	w.Printf("  Primary domain: %s\n", info.PrimaryDomain)
	return nil
}

type authStatus struct {
	ConfigFile      string   `json:"config_file"`
	ActiveProfile   string   `json:"active_profile"`
	Configured      bool     `json:"configured"`
	Domain          string   `json:"domain,omitempty"`
	AdminSubject    string   `json:"admin_subject,omitempty"`
	CredentialsFile string   `json:"credentials_file,omitempty"`
	CredentialType  string   `json:"credential_type,omitempty"`
	TokenFile       string   `json:"token_file,omitempty"`
	TokenPresent    bool     `json:"token_present"`
	AuthMethod      string   `json:"auth_method,omitempty"`
	Scopes          []string `json:"scopes,omitempty"`
	MissingScopes   []string `json:"missing_scopes,omitempty"`
	Ready           bool     `json:"ready"`
	Message         string   `json:"message"`
}

func (r Runner) authStatus(ctx context.Context, flags map[string]string) error {
	_ = ctx
	status := r.localAuthStatus()
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(status)
	}
	w.Printf("Config file: %s\n", status.ConfigFile)
	w.Printf("Active profile: %s\n", status.ActiveProfile)
	if !status.Configured {
		w.Println("Status: not configured")
		w.Println(status.Message)
		return nil
	}
	w.Printf("Domain: %s\n", status.Domain)
	w.Printf("Admin subject: %s\n", status.AdminSubject)
	w.Printf("Auth method: %s\n", status.AuthMethod)
	w.Printf("Credentials: %s", status.CredentialsFile)
	if status.CredentialType != "" {
		w.Printf(" (%s)", status.CredentialType)
	}
	w.Println("")
	if status.TokenFile != "" {
		w.Printf("Token file: %s\n", status.TokenFile)
		w.Printf("Token present: %t\n", status.TokenPresent)
	}
	w.Println("Scopes:")
	for _, scope := range status.Scopes {
		w.Printf("  %s\n", scope)
	}
	if len(status.MissingScopes) > 0 {
		w.Println("Missing required scopes:")
		for _, scope := range status.MissingScopes {
			w.Printf("  %s\n", scope)
		}
	}
	if status.Ready {
		w.Println("Status: ready for API validation")
		w.Println("Run `gws check connection` to make a live Admin SDK call.")
		return nil
	}
	w.Println("Status: needs attention")
	w.Println(status.Message)
	return nil
}

func valueOrPrompt(reader *bufio.Reader, w output.Writer, value, label, fallback string) (string, error) {
	if value != "" {
		return value, nil
	}
	if fallback != "" {
		w.Printf("%s [%s]: ", label, fallback)
	} else {
		w.Printf("%s: ", label)
	}
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return fallback, nil
	}
	return line, nil
}

func (r Runner) batchRun(ctx context.Context, flags map[string]string) error {
	path := strings.TrimSpace(flags["file"])
	if path == "" {
		return errors.New("batch file is required\n\nUsage: gws batch run --file PATH [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open batch file: %w", err)
	}
	defer file.Close()
	commands, err := batch.Parse(file)
	if err != nil {
		return err
	}
	if len(commands) == 0 {
		return errors.New("batch file did not contain any commands")
	}
	return r.runBatchCommands(ctx, commands, flags)
}

func (r Runner) batchCSV(ctx context.Context, flags map[string]string) error {
	path := strings.TrimSpace(flags["file"])
	if path == "" {
		return errors.New("csv file is required\n\nUsage: gws batch csv --file PATH --command TEMPLATE [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]")
	}
	template := strings.TrimSpace(flags["command"])
	if template == "" {
		return errors.New("command template is required\n\nUsage: gws batch csv --file PATH --command TEMPLATE [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]")
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()
	commands, err := batch.ExpandCSV(file, template)
	if err != nil {
		return err
	}
	return r.runBatchCommands(ctx, commands, flags)
}

func (r Runner) batchTemplate(ctx context.Context, args []string, flags map[string]string) error {
	_ = ctx
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return fmt.Errorf("workflow name is required\n\nUsage: gws batch template WORKFLOW [--example]\n\nAvailable workflows: %s", strings.Join(batchTemplateNames(), ", "))
	}
	template, ok := batchTemplateByName(strings.TrimSpace(args[2]))
	if !ok {
		return fmt.Errorf("unknown batch workflow %q\n\nAvailable workflows: %s", args[2], strings.Join(batchTemplateNames(), ", "))
	}
	showExample, err := optionalBoolFlag(flags["example"], "--example")
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	w.Printf("Workflow: %s\n", template.name)
	w.Printf("Purpose: %s\n", template.description)
	w.Printf("Headers: %s\n", strings.Join(template.headers, ","))
	w.Println("Command template:")
	w.Printf("  %s\n", template.command)
	if showExample {
		w.Println("Example CSV:")
		_ = w.CSV(template.exampleRows)
	}
	return nil
}

func (r Runner) runBatchCommands(ctx context.Context, commands []batch.Command, flags map[string]string) error {
	dryRun, err := optionalBoolFlag(flags["dry-run"], "--dry-run")
	if err != nil {
		return err
	}
	workers, err := batchWorkersFlag(flags["workers"])
	if err != nil {
		return err
	}
	failFast, err := optionalBoolFlag(flags["fail-fast"], "--fail-fast")
	if err != nil {
		return err
	}
	timeout, err := batchTimeoutFlag(flags["timeout"])
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	if dryRun {
		w.Printf("Batch plan: %d command(s)\n", len(commands))
		w.Printf("Workers: %d\n", workers)
		if timeout > 0 {
			w.Printf("Timeout: %s per command\n", timeout)
		}
		for _, command := range commands {
			w.Printf("  line %d: %s\n", command.Line, strings.Join(command.Args, " "))
		}
		return nil
	}
	outputs := make([]batchOutput, len(commands))
	results := batch.Run(ctx, commands, batch.Options{
		Workers:  workers,
		FailFast: failFast,
		Execute: func(execCtx context.Context, index int, command batch.Command) error {
			if timeout > 0 {
				var cancel context.CancelFunc
				execCtx, cancel = context.WithTimeout(execCtx, timeout)
				defer cancel()
			}
			parsed, err := cli.Parse(command.Args)
			if err != nil {
				return err
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			child := Runner{
				Stdin:     strings.NewReader(""),
				Stdout:    &stdout,
				Stderr:    &stderr,
				Config:    r.Config,
				Directory: r.Directory,
				Sheets:    r.Sheets,
			}
			err = child.Run(execCtx, parsed)
			outputs[index] = batchOutput{
				stdout: strings.TrimRight(stdout.String(), "\n"),
				stderr: strings.TrimRight(stderr.String(), "\n"),
			}
			return err
		},
	})
	failures := 0
	for i, result := range results {
		if result.Command.Line == 0 {
			continue
		}
		if result.Err == nil {
			w.Printf("OK   line %d: %s\n", result.Command.Line, strings.Join(result.Command.Args, " "))
			printBatchCommandOutput(w, outputs[i])
			continue
		}
		if errors.Is(result.Err, context.Canceled) && failFast {
			w.Printf("SKIP line %d: %s\n", result.Command.Line, strings.Join(result.Command.Args, " "))
			printBatchCommandOutput(w, outputs[i])
			continue
		}
		failures++
		w.Printf("FAIL line %d: %s\n", result.Command.Line, strings.Join(result.Command.Args, " "))
		printBatchCommandOutput(w, outputs[i])
		w.Printf("  %v\n", result.Err)
	}
	w.Printf("Batch complete: %d command(s), %d failure(s), %d worker(s)\n", len(commands), failures, workers)
	if failures > 0 {
		return fmt.Errorf("batch run failed with %d command error(s)", failures)
	}
	return nil
}

func printBatchCommandOutput(w output.Writer, captured batchOutput) {
	if captured.stdout != "" {
		w.Println("  stdout:")
		for _, line := range strings.Split(captured.stdout, "\n") {
			w.Printf("    %s\n", line)
		}
	}
	if captured.stderr != "" {
		w.Println("  stderr:")
		for _, line := range strings.Split(captured.stderr, "\n") {
			w.Printf("    %s\n", line)
		}
	}
}

func batchTemplateNames() []string {
	names := make([]string, 0, len(batchTemplates()))
	for _, template := range batchTemplates() {
		names = append(names, template.name)
	}
	return names
}

func batchTemplateByName(name string) (batchTemplate, bool) {
	name = strings.TrimSpace(name)
	for _, template := range batchTemplates() {
		if template.name == name {
			return template, true
		}
	}
	return batchTemplate{}, false
}

func batchTemplates() []batchTemplate {
	return []batchTemplate{
		{
			name:        "user-update",
			description: "Bulk update common user profile fields.",
			headers:     []string{"email", "orgUnit", "recoveryEmail", "recoveryPhone"},
			command:     `gws batch csv --file ./user-update.csv --command 'update user "{{email}}" --org-unit "{{orgUnit}}" --recovery-email "{{recoveryEmail}}" --recovery-phone "{{recoveryPhone}}"'`,
			exampleRows: [][]string{
				{"email", "orgUnit", "recoveryEmail", "recoveryPhone"},
				{"ada@example.com", "/Engineering", "ada.recovery@example.com", "+15551234567"},
				{"grace@example.com", "/Sales", "grace.recovery@example.com", "+15557654321"},
			},
		},
		{
			name:        "user-unsuspend",
			description: "Bulk unsuspend users by primary email.",
			headers:     []string{"email"},
			command:     `gws batch csv --file ./user-unsuspend.csv --command 'unsuspend user "{{email}}"'`,
			exampleRows: [][]string{
				{"email"},
				{"returning-user1@example.com"},
				{"returning-user2@example.com"},
			},
		},
		{
			name:        "group-member-sync",
			description: "Bulk reconcile group membership from per-group CSV templates.",
			headers:     []string{"groupEmail", "csvPath"},
			command:     `gws batch csv --file ./group-sync.csv --command 'sync group-members "{{groupEmail}}" --members-csv "{{csvPath}}" --confirm'`,
			exampleRows: [][]string{
				{"groupEmail", "csvPath"},
				{"eng@example.com", "./eng-members.csv"},
				{"sales@example.com", "./sales-members.csv"},
			},
		},
		{
			name:        "user-suspend",
			description: "Bulk suspend users by primary email.",
			headers:     []string{"email"},
			command:     `gws batch csv --file ./user-suspend.csv --command 'suspend user "{{email}}"'`,
			exampleRows: [][]string{
				{"email"},
				{"former-user1@example.com"},
				{"former-user2@example.com"},
			},
		},
		{
			name:        "user-make-admin",
			description: "Bulk grant super admin status.",
			headers:     []string{"email"},
			command:     `gws batch csv --file ./user-make-admin.csv --command 'make admin "{{email}}"'`,
			exampleRows: [][]string{
				{"email"},
				{"lead-admin1@example.com"},
				{"lead-admin2@example.com"},
			},
		},
		{
			name:        "user-revoke-admin",
			description: "Bulk revoke super admin status.",
			headers:     []string{"email"},
			command:     `gws batch csv --file ./user-revoke-admin.csv --command 'revoke admin "{{email}}"'`,
			exampleRows: [][]string{
				{"email"},
				{"former-admin1@example.com"},
				{"former-admin2@example.com"},
			},
		},
		{
			name:        "group-create",
			description: "Bulk create groups with names and descriptions.",
			headers:     []string{"email", "name", "description"},
			command:     `gws batch csv --file ./group-create.csv --command 'create group "{{email}}" --name "{{name}}" --description "{{description}}"'`,
			exampleRows: [][]string{
				{"email", "name", "description"},
				{"eng@example.com", "Engineering", "Primary engineering discussion group"},
				{"sales@example.com", "Sales", "Primary sales discussion group"},
			},
		},
		{
			name:        "group-member-add",
			description: "Bulk add members to groups with explicit roles.",
			headers:     []string{"groupEmail", "memberEmail", "role"},
			command:     `gws batch csv --file ./group-member-add.csv --command 'add group-member "{{groupEmail}}" "{{memberEmail}}" --role "{{role}}"'`,
			exampleRows: [][]string{
				{"groupEmail", "memberEmail", "role"},
				{"eng@example.com", "ada@example.com", "MEMBER"},
				{"eng@example.com", "lead@example.com", "MANAGER"},
			},
		},
		{
			name:        "group-member-remove",
			description: "Bulk remove members from groups.",
			headers:     []string{"groupEmail", "memberEmail"},
			command:     `gws batch csv --file ./group-member-remove.csv --command 'remove group-member "{{groupEmail}}" "{{memberEmail}}"'`,
			exampleRows: [][]string{
				{"groupEmail", "memberEmail"},
				{"eng@example.com", "former-member1@example.com"},
				{"sales@example.com", "former-member2@example.com"},
			},
		},
		{
			name:        "ou-create",
			description: "Bulk create organizational units.",
			headers:     []string{"name", "parent", "description"},
			command:     `gws batch csv --file ./ou-create.csv --command 'create ou --name "{{name}}" --parent "{{parent}}" --description "{{description}}"'`,
			exampleRows: [][]string{
				{"name", "parent", "description"},
				{"Engineering", "/", "Engineering users"},
				{"Sales", "/", "Sales users"},
			},
		},
		{
			name:        "ou-update",
			description: "Bulk update organizational unit descriptions or parents.",
			headers:     []string{"path", "parent", "description"},
			command:     `gws batch csv --file ./ou-update.csv --command 'update ou "{{path}}" --parent "{{parent}}" --description "{{description}}"'`,
			exampleRows: [][]string{
				{"path", "parent", "description"},
				{"/Engineering", "/", "Engineering users"},
				{"/Sales", "/", "Sales users"},
			},
		},
	}
}

func (r Runner) configShow(flags map[string]string) error {
	cfg, path, err := r.loadConfig()
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(cfg)
	}
	profile, ok := cfg.Active()
	w.Printf("Config file: %s\n", path)
	w.Printf("Active profile: %s\n", cfg.ActiveProfile)
	if !ok {
		w.Println("Status: not configured")
		w.Println("Run `gws setup` to create a profile.")
		return nil
	}
	w.Printf("Domain: %s\n", profile.Domain)
	w.Printf("Admin subject: %s\n", profile.AdminSubject)
	w.Printf("Credentials file: %s\n", profile.CredentialsFile)
	if profile.TokenFile == "" {
		w.Println("Token file: not used for service account profiles")
	} else {
		w.Printf("Token file: %s\n", profile.TokenFile)
	}
	w.Printf("Auth method: %s\n", profile.AuthMethod)
	w.Println("Scopes:")
	for _, scope := range profile.Scopes {
		w.Printf("  %s\n", scope)
	}
	return nil
}

func (r Runner) checkConnection(ctx context.Context, flags map[string]string) error {
	info, err := r.validateGoogle(ctx)
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(info)
	}
	w.Println("Connection OK")
	printDomainDetails(w, info)
	return nil
}

func (r Runner) infoDomain(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) >= 3 && strings.TrimSpace(args[2]) != "" {
		return r.infoWorkspaceDomain(ctx, strings.TrimSpace(args[2]), flags)
	}
	info, err := r.validateGoogle(ctx)
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(info)
	}
	printDomainDetails(w, info)
	return nil
}

func (r Runner) printDomains(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	domains, err := r.Directory.Domains(ctx, profile)
	if err != nil {
		return explainAPICommandFailure(profile, "print domains", err)
	}
	fields, err := selectedDomainFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildDomainRows(domains, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws domains", rows); err != nil {
			return explainSheetExportFailure(profile, "print domains", err)
		}
		return nil
	}
	return renderDomains(r.Stdout, domains, fields, flags)
}

func (r Runner) infoWorkspaceDomain(ctx context.Context, domainName string, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	domain, err := r.Directory.Domain(ctx, profile, domainName)
	if err != nil {
		return explainAPICommandFailure(profile, "info domain", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(domain)
	}
	printWorkspaceDomainDetails(w, domain)
	return nil
}

func (r Runner) createDomain(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("domain name is required\n\nUsage: gws create domain example.com")
	}
	domainName := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	domain, err := r.Directory.CreateDomain(ctx, profile, domainName)
	if err != nil {
		return explainAPICommandFailure(profile, "create domain", err)
	}
	return printWorkspaceDomainResult(r.Stdout, domain, flags["json"] == "true", "Domain created")
}

func (r Runner) deleteDomain(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("domain name is required\n\nUsage: gws delete domain example.com --confirm")
	}
	if err := requireConfirm(flags, "delete domain"); err != nil {
		return err
	}
	domainName := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteDomain(ctx, profile, domainName); err != nil {
		return explainAPICommandFailure(profile, "delete domain", err)
	}
	output.New(r.Stdout).Printf("Domain deleted: %s\n", domainName)
	return nil
}

func (r Runner) printDomainAliases(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	aliases, err := r.Directory.DomainAliases(ctx, profile)
	if err != nil {
		return explainAPICommandFailure(profile, "print domain-aliases", err)
	}
	fields, err := selectedDomainAliasFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildDomainAliasRows(aliases, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws domain aliases", rows); err != nil {
			return explainSheetExportFailure(profile, "print domain-aliases", err)
		}
		return nil
	}
	return renderDomainAliases(r.Stdout, aliases, fields, flags)
}

func (r Runner) infoDomainAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("domain alias name is required\n\nUsage: gws info domain-alias alias.example.com")
	}
	aliasName := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	alias, err := r.Directory.DomainAlias(ctx, profile, aliasName)
	if err != nil {
		return explainAPICommandFailure(profile, "info domain-alias", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(alias)
	}
	printDomainAliasDetails(w, alias)
	return nil
}

func (r Runner) createDomainAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("domain alias name is required\n\nUsage: gws create domain-alias alias.example.com --parent example.com")
	}
	parent := strings.TrimSpace(flags["parent"])
	if parent == "" {
		return errors.New("--parent is required when creating a domain alias")
	}
	aliasName := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	alias, err := r.Directory.CreateDomainAlias(ctx, profile, parent, aliasName)
	if err != nil {
		return explainAPICommandFailure(profile, "create domain-alias", err)
	}
	return printDomainAliasResult(r.Stdout, alias, flags["json"] == "true", "Domain alias created")
}

func (r Runner) deleteDomainAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("domain alias name is required\n\nUsage: gws delete domain-alias alias.example.com --confirm")
	}
	if err := requireConfirm(flags, "delete domain-alias"); err != nil {
		return err
	}
	aliasName := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteDomainAlias(ctx, profile, aliasName); err != nil {
		return explainAPICommandFailure(profile, "delete domain-alias", err)
	}
	output.New(r.Stdout).Printf("Domain alias deleted: %s\n", aliasName)
	return nil
}

func (r Runner) infoUser(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws info user user@example.com")
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	user, err := r.Directory.User(ctx, profile, email)
	if err != nil {
		return explainAPICommandFailure(profile, "info user", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(user)
	}
	printUserDetails(w, user)
	return nil
}

func (r Runner) createUser(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws create user user@example.com --given-name NAME --family-name NAME --password-file PATH")
	}
	givenName := strings.TrimSpace(flags["given-name"])
	familyName := strings.TrimSpace(flags["family-name"])
	if givenName == "" {
		return errors.New("--given-name is required when creating a user")
	}
	if familyName == "" {
		return errors.New("--family-name is required when creating a user")
	}
	password, err := passwordFromFlags(flags)
	if err != nil {
		return err
	}
	changePassword, err := boolFlag(flags["change-password-at-next-login"], true)
	if err != nil {
		return fmt.Errorf("--change-password-at-next-login must be true or false")
	}
	create := google.UserCreate{
		PrimaryEmail:              strings.TrimSpace(args[2]),
		GivenName:                 givenName,
		FamilyName:                familyName,
		Password:                  password,
		ChangePasswordAtNextLogin: changePassword,
	}
	if orgUnit, ok := flags["org-unit"]; ok {
		create.OrgUnitPath = normalizeOrgUnitPath(orgUnit)
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	user, err := r.Directory.CreateUser(ctx, profile, create)
	if err != nil {
		return explainAPICommandFailure(profile, "create user", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(user)
	}
	w.Printf("User created: %s\n", user.PrimaryEmail)
	if user.Name != "" {
		w.Printf("Name: %s\n", user.Name)
	}
	w.Printf("Org unit: %s\n", user.OrgUnitPath)
	w.Printf("Change password at next login: %t\n", changePassword)
	return nil
}

func (r Runner) infoGroup(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws info group group@example.com")
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	group, err := r.Directory.Group(ctx, profile, email)
	if err != nil {
		return explainAPICommandFailure(profile, "info group", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(group)
	}
	printGroupDetails(w, group)
	return nil
}

func (r Runner) printGmailDelegates(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws print gmail-delegates user@example.com [--json]")
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := requireServiceAccountProfile(profile, "print gmail-delegates"); err != nil {
		return err
	}
	delegates, err := r.Gmail.Delegates(ctx, profile, strings.TrimSpace(args[2]))
	if err != nil {
		return explainGmailCommandFailure(profile, "print gmail-delegates", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(delegates)
	}
	rows := [][]string{{"Delegate Email", "Status"}}
	for _, delegate := range delegates {
		rows = append(rows, []string{delegate.DelegateEmail, delegate.VerificationStatus})
	}
	return w.Table(rows)
}

func (r Runner) infoGmailDelegate(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("user email and delegate email are required\n\nUsage: gws info gmail-delegate user@example.com delegate@example.com [--json]")
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := requireServiceAccountProfile(profile, "info gmail-delegate"); err != nil {
		return err
	}
	delegate, err := r.Gmail.Delegate(ctx, profile, strings.TrimSpace(args[2]), strings.TrimSpace(args[3]))
	if err != nil {
		return explainGmailCommandFailure(profile, "info gmail-delegate", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(delegate)
	}
	printGmailDelegateDetails(w, delegate)
	return nil
}

func (r Runner) createGmailDelegate(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("user email and delegate email are required\n\nUsage: gws create gmail-delegate user@example.com delegate@example.com [--json]")
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := requireServiceAccountProfile(profile, "create gmail-delegate"); err != nil {
		return err
	}
	delegate, err := r.Gmail.CreateDelegate(ctx, profile, strings.TrimSpace(args[2]), strings.TrimSpace(args[3]))
	if err != nil {
		return explainGmailCommandFailure(profile, "create gmail-delegate", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(delegate)
	}
	w.Printf("Gmail delegate created: %s\n", delegate.DelegateEmail)
	printGmailDelegateDetails(w, delegate)
	return nil
}

func (r Runner) deleteGmailDelegate(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("user email and delegate email are required\n\nUsage: gws delete gmail-delegate user@example.com delegate@example.com --confirm")
	}
	if err := requireConfirm(flags, "delete gmail-delegate"); err != nil {
		return err
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := requireServiceAccountProfile(profile, "delete gmail-delegate"); err != nil {
		return err
	}
	delegateEmail := strings.TrimSpace(args[3])
	if err := r.Gmail.DeleteDelegate(ctx, profile, strings.TrimSpace(args[2]), delegateEmail); err != nil {
		return explainGmailCommandFailure(profile, "delete gmail-delegate", err)
	}
	output.New(r.Stdout).Printf("Gmail delegate deleted: %s\n", delegateEmail)
	return nil
}

func (r Runner) createGroup(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws create group group@example.com --name NAME")
	}
	email := strings.TrimSpace(args[2])
	name := strings.TrimSpace(flags["name"])
	if name == "" {
		return errors.New("--name is required when creating a group")
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	group, err := r.Directory.CreateGroup(ctx, profile, google.GroupInfo{
		Email:       email,
		Name:        name,
		Description: strings.TrimSpace(flags["description"]),
	})
	if err != nil {
		return explainAPICommandFailure(profile, "create group", err)
	}
	return printGroupResult(r.Stdout, group, flags["json"] == "true", "Group created")
}

func (r Runner) updateGroup(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws update group group@example.com [--email new-group@example.com] [--name NAME] [--description TEXT]")
	}
	newEmail, hasEmail := flags["email"]
	name, hasName := flags["name"]
	description, hasDescription := flags["description"]
	if !hasEmail && !hasName && !hasDescription {
		return errors.New("nothing to update; provide --email, --name, or --description")
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	group, err := r.Directory.UpdateGroup(ctx, profile, email, google.GroupInfo{
		Email:       strings.TrimSpace(newEmail),
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	})
	if err != nil {
		return explainAPICommandFailure(profile, "update group", err)
	}
	return printGroupResult(r.Stdout, group, flags["json"] == "true", "Group updated")
}

func (r Runner) deleteGroup(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws delete group group@example.com --confirm")
	}
	if err := requireConfirm(flags, "delete group"); err != nil {
		return err
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteGroup(ctx, profile, email); err != nil {
		return explainAPICommandFailure(profile, "delete group", err)
	}
	output.New(r.Stdout).Printf("Group deleted: %s\n", email)
	return nil
}

func (r Runner) printGroupAliases(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws print group-aliases group@example.com")
	}
	groupEmail := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	aliases, err := r.Directory.GroupAliases(ctx, profile, groupEmail)
	if err != nil {
		return explainAPICommandFailure(profile, "print group-aliases", err)
	}
	fields, err := selectedAliasFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildAliasRows(aliases, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws group aliases", rows); err != nil {
			return explainSheetExportFailure(profile, "print group-aliases", err)
		}
		return nil
	}
	return renderAliases(r.Stdout, aliases, fields, flags)
}

func (r Runner) createGroupAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("group email and alias email are required\n\nUsage: gws create group-alias group@example.com alias@example.com")
	}
	groupEmail := strings.TrimSpace(args[2])
	aliasEmail := strings.TrimSpace(args[3])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	alias, err := r.Directory.CreateGroupAlias(ctx, profile, groupEmail, aliasEmail)
	if err != nil {
		return explainAPICommandFailure(profile, "create group-alias", err)
	}
	return printAliasResult(r.Stdout, alias, flags["json"] == "true", "Group alias created")
}

func (r Runner) deleteGroupAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("group email and alias email are required\n\nUsage: gws delete group-alias group@example.com alias@example.com --confirm")
	}
	if err := requireConfirm(flags, "delete group-alias"); err != nil {
		return err
	}
	groupEmail := strings.TrimSpace(args[2])
	aliasEmail := strings.TrimSpace(args[3])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteGroupAlias(ctx, profile, groupEmail, aliasEmail); err != nil {
		return explainAPICommandFailure(profile, "delete group-alias", err)
	}
	output.New(r.Stdout).Printf("Group alias deleted: %s\n", aliasEmail)
	return nil
}

func (r Runner) infoOrgUnit(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("org unit path is required\n\nUsage: gws info ou /Engineering")
	}
	path := normalizeOrgUnitPath(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	ou, err := r.Directory.OrgUnit(ctx, profile, path)
	if err != nil {
		return explainAPICommandFailure(profile, "info ou", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(ou)
	}
	printOrgUnitDetails(w, ou)
	return nil
}

func (r Runner) createOrgUnit(ctx context.Context, flags map[string]string) error {
	name := strings.TrimSpace(flags["name"])
	if name == "" {
		return errors.New("--name is required when creating an org unit")
	}
	parent := normalizeOrgUnitPath(flags["parent"])
	if flags["parent"] == "" {
		parent = "/"
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	ou, err := r.Directory.CreateOrgUnit(ctx, profile, google.OrgUnitCreate{
		Name:              name,
		ParentOrgUnitPath: parent,
		Description:       strings.TrimSpace(flags["description"]),
	})
	if err != nil {
		return explainAPICommandFailure(profile, "create ou", err)
	}
	return printOrgUnitResult(r.Stdout, ou, flags["json"] == "true", "Org unit created")
}

func (r Runner) updateOrgUnit(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("org unit path is required\n\nUsage: gws update ou /Engineering [--name NAME] [--parent /PATH] [--description TEXT]")
	}
	name, hasName := flags["name"]
	parent, hasParent := flags["parent"]
	description, hasDescription := flags["description"]
	if !hasName && !hasParent && !hasDescription {
		return errors.New("nothing to update; provide --name, --parent, or --description")
	}
	update := google.OrgUnitUpdate{
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	}
	if hasParent {
		update.ParentOrgUnitPath = normalizeOrgUnitPath(parent)
	}
	path := normalizeOrgUnitPath(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	ou, err := r.Directory.UpdateOrgUnit(ctx, profile, path, update)
	if err != nil {
		return explainAPICommandFailure(profile, "update ou", err)
	}
	return printOrgUnitResult(r.Stdout, ou, flags["json"] == "true", "Org unit updated")
}

func (r Runner) deleteOrgUnit(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("org unit path is required\n\nUsage: gws delete ou /Engineering --confirm")
	}
	if err := requireConfirm(flags, "delete ou"); err != nil {
		return err
	}
	path := normalizeOrgUnitPath(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteOrgUnit(ctx, profile, path); err != nil {
		return explainAPICommandFailure(profile, "delete ou", err)
	}
	output.New(r.Stdout).Printf("Org unit deleted: %s\n", path)
	return nil
}

func (r Runner) printGroups(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	opts, err := groupListOptionsFromFlags(flags)
	if err != nil {
		return err
	}
	groups, err := r.Directory.Groups(ctx, profile, opts)
	if err != nil {
		return explainAPICommandFailure(profile, "print groups", err)
	}
	fields, err := selectedGroupFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildGroupRows(groups, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws groups", rows); err != nil {
			return explainSheetExportFailure(profile, "print groups", err)
		}
		return nil
	}
	return renderGroups(r.Stdout, groups, fields, flags)
}

func (r Runner) printGroupMembers(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws print group-members group@example.com")
	}
	groupEmail := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	limit, err := limitFlag(flags["limit"], 100)
	if err != nil {
		return err
	}
	members, err := r.Directory.GroupMembers(ctx, profile, groupEmail, limit)
	if err != nil {
		return explainAPICommandFailure(profile, "print group-members", err)
	}
	fields, err := selectedMemberFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildMemberRows(members, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws group members", rows); err != nil {
			return explainSheetExportFailure(profile, "print group-members", err)
		}
		return nil
	}
	return renderMembers(r.Stdout, members, fields, flags)
}

func (r Runner) infoGroupMember(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("group email and member email are required\n\nUsage: gws info group-member group@example.com user@example.com")
	}
	groupEmail := strings.TrimSpace(args[2])
	memberEmail := strings.TrimSpace(args[3])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	member, err := r.Directory.GroupMember(ctx, profile, groupEmail, memberEmail)
	if err != nil {
		return explainAPICommandFailure(profile, "info group-member", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(member)
	}
	printMemberDetails(w, member)
	return nil
}

func (r Runner) addGroupMember(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("group email and member email are required\n\nUsage: gws add group-member group@example.com user@example.com")
	}
	groupEmail := strings.TrimSpace(args[2])
	memberEmail := strings.TrimSpace(args[3])
	role := groupMemberRole(flags["role"])
	if role == "" {
		return errors.New("--role must be OWNER, MANAGER, or MEMBER")
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	member, err := r.Directory.AddGroupMember(ctx, profile, groupEmail, memberEmail, role)
	if err != nil {
		return explainAPICommandFailure(profile, "add group-member", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(member)
	}
	w.Printf("Group member added: %s\n", member.Email)
	printStringField(w, "ID", member.ID)
	w.Printf("Role: %s\n", member.Role)
	printStringField(w, "Type", member.Type)
	printStringField(w, "Status", member.Status)
	printStringField(w, "Delivery settings", member.DeliverySettings)
	return nil
}

func (r Runner) updateGroupMember(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("group email and member email are required\n\nUsage: gws update group-member group@example.com user@example.com --role OWNER|MANAGER|MEMBER")
	}
	groupEmail := strings.TrimSpace(args[2])
	memberEmail := strings.TrimSpace(args[3])
	role := groupMemberRole(flags["role"])
	if role == "" {
		return errors.New("--role must be OWNER, MANAGER, or MEMBER")
	}
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	member, err := r.Directory.UpdateGroupMember(ctx, profile, groupEmail, memberEmail, role)
	if err != nil {
		return explainAPICommandFailure(profile, "update group-member", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(member)
	}
	w.Printf("Group member updated: %s\n", member.Email)
	printMemberDetails(w, member)
	return nil
}

func (r Runner) removeGroupMember(ctx context.Context, args []string, flags map[string]string) error {
	_ = flags
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("group email and member email are required\n\nUsage: gws remove group-member group@example.com user@example.com")
	}
	groupEmail := strings.TrimSpace(args[2])
	memberEmail := strings.TrimSpace(args[3])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.RemoveGroupMember(ctx, profile, groupEmail, memberEmail); err != nil {
		return explainAPICommandFailure(profile, "remove group-member", err)
	}
	output.New(r.Stdout).Printf("Group member removed: %s\n", memberEmail)
	return nil
}

func (r Runner) syncGroupMembers(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("group email is required\n\nUsage: gws sync group-members group@example.com (--members user1@example.com,user2@example.com | --members-file PATH | --members-csv PATH | --members-sheet SHEET_ID_OR_URL [--sheet-range RANGE]) [--role MEMBER] [--ignore-role] [--dry-run] [--confirm]")
	}
	groupEmail := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	source, err := r.groupMemberSyncSource(ctx, profile, flags)
	if err != nil {
		return err
	}
	ignoreRole, err := optionalBoolFlag(flags["ignore-role"], "--ignore-role")
	if err != nil {
		return err
	}
	if source.Structured {
		if ignoreRole {
			return errors.New("--ignore-role cannot be combined with --members-csv or --members-sheet")
		}
		if strings.TrimSpace(flags["role"]) != "" {
			return errors.New("--role cannot be combined with --members-csv or --members-sheet when roles are supplied per member")
		}
	} else if ignoreRole && strings.TrimSpace(flags["role"]) != "" {
		return errors.New("--ignore-role cannot be combined with --role")
	}
	role := "MEMBER"
	if !source.Structured && !ignoreRole {
		role = groupMemberRole(flags["role"])
		if role == "" {
			return errors.New("--role must be OWNER, MANAGER, or MEMBER")
		}
	}
	dryRun, err := optionalBoolFlag(flags["dry-run"], "--dry-run")
	if err != nil {
		return err
	}
	if !dryRun {
		if err := requireConfirm(flags, "sync group-members"); err != nil {
			return err
		}
	}
	currentMembers, err := r.Directory.GroupMembers(ctx, profile, groupEmail, 0)
	if err != nil {
		return explainAPICommandFailure(profile, "sync group-members", err)
	}
	w := output.New(r.Stdout)
	if source.Structured {
		plan := planStructuredGroupMemberSync(currentMembers, source.Targets)
		if dryRun {
			w.Printf("Group member sync plan: %s\n", groupEmail)
			w.Println("Mode: explicit roles")
			printSyncSummary(w, "Would add", len(plan.ToAdd))
			printSyncSummary(w, "Would remove", len(plan.ToRemove))
			printSyncSummary(w, "Would update roles", len(plan.ToUpdate))
			printSyncSummary(w, "Unchanged", len(plan.Unchanged))
			printRoleTargets(w, "Add members", plan.ToAdd)
			printRoleTargets(w, "Remove members", plan.ToRemove)
			printRoleChanges(w, "Update roles", plan.ToUpdate)
			printRoleTargets(w, "Unchanged members", plan.Unchanged)
			return nil
		}
		for _, target := range plan.ToAdd {
			if _, err := r.Directory.AddGroupMember(ctx, profile, groupEmail, target.Email, target.Role); err != nil {
				return explainAPICommandFailure(profile, "sync group-members", fmt.Errorf("add %s as %s: %w", target.Email, target.Role, err))
			}
		}
		for _, change := range plan.ToUpdate {
			if _, err := r.Directory.UpdateGroupMember(ctx, profile, groupEmail, change.Email, change.ToRole); err != nil {
				return explainAPICommandFailure(profile, "sync group-members", fmt.Errorf("update %s role %s->%s: %w", change.Email, change.FromRole, change.ToRole, err))
			}
		}
		for _, target := range plan.ToRemove {
			if err := r.Directory.RemoveGroupMember(ctx, profile, groupEmail, target.Email); err != nil {
				return explainAPICommandFailure(profile, "sync group-members", fmt.Errorf("remove %s: %w", target.Email, err))
			}
		}
		w.Printf("Group member sync complete: %s\n", groupEmail)
		w.Println("Mode: explicit roles")
		printSyncSummary(w, "Added", len(plan.ToAdd))
		printSyncSummary(w, "Removed", len(plan.ToRemove))
		printSyncSummary(w, "Updated roles", len(plan.ToUpdate))
		printSyncSummary(w, "Unchanged", len(plan.Unchanged))
		printRoleTargets(w, "Added members", plan.ToAdd)
		printRoleTargets(w, "Removed members", plan.ToRemove)
		printRoleChanges(w, "Updated role members", plan.ToUpdate)
		printRoleTargets(w, "Unchanged members", plan.Unchanged)
		return nil
	}
	plan := planGroupMemberSync(currentMembers, source.Emails, role, ignoreRole)
	if dryRun {
		w.Printf("Group member sync plan: %s\n", groupEmail)
		if ignoreRole {
			w.Println("Mode: ignore role")
		} else {
			w.Printf("Role: %s\n", role)
		}
		printSyncSummary(w, "Would add", len(plan.ToAdd))
		printSyncSummary(w, "Would remove", len(plan.ToRemove))
		printSyncSummary(w, "Would update roles", len(plan.ToUpdate))
		printSyncSummary(w, "Unchanged", len(plan.Unchanged))
		printStringSliceField(w, "Add members", plan.ToAdd)
		printStringSliceField(w, "Remove members", plan.ToRemove)
		printRoleChanges(w, "Update roles", plan.ToUpdate)
		return nil
	}
	for _, email := range plan.ToAdd {
		if _, err := r.Directory.AddGroupMember(ctx, profile, groupEmail, email, role); err != nil {
			return explainAPICommandFailure(profile, "sync group-members", fmt.Errorf("add %s: %w", email, err))
		}
	}
	for _, change := range plan.ToUpdate {
		if _, err := r.Directory.UpdateGroupMember(ctx, profile, groupEmail, change.Email, change.ToRole); err != nil {
			return explainAPICommandFailure(profile, "sync group-members", fmt.Errorf("update %s role %s->%s: %w", change.Email, change.FromRole, change.ToRole, err))
		}
	}
	for _, email := range plan.ToRemove {
		if err := r.Directory.RemoveGroupMember(ctx, profile, groupEmail, email); err != nil {
			return explainAPICommandFailure(profile, "sync group-members", fmt.Errorf("remove %s: %w", email, err))
		}
	}
	w.Printf("Group member sync complete: %s\n", groupEmail)
	if ignoreRole {
		w.Println("Mode: ignore role")
	} else {
		w.Printf("Role: %s\n", role)
	}
	printSyncSummary(w, "Added", len(plan.ToAdd))
	printSyncSummary(w, "Removed", len(plan.ToRemove))
	printSyncSummary(w, "Updated roles", len(plan.ToUpdate))
	printSyncSummary(w, "Unchanged", len(plan.Unchanged))
	printStringSliceField(w, "Added members", plan.ToAdd)
	printStringSliceField(w, "Removed members", plan.ToRemove)
	printRoleChanges(w, "Updated role members", plan.ToUpdate)
	return nil
}

func renderUsers(out io.Writer, users []google.UserInfo, fields []userFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(users)
		}
		records := make([]map[string]any, 0, len(users))
		for _, user := range users {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(user)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildUserRows(users, fields))
	default:
		return w.Table(buildUserRows(users, fields))
	}
}

func renderGroups(out io.Writer, groups []google.GroupInfo, fields []groupFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(groups)
		}
		records := make([]map[string]any, 0, len(groups))
		for _, group := range groups {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(group)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildGroupRows(groups, fields))
	default:
		return w.Table(buildGroupRows(groups, fields))
	}
}

func renderMembers(out io.Writer, members []google.MemberInfo, fields []memberFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(members)
		}
		records := make([]map[string]any, 0, len(members))
		for _, member := range members {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(member)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildMemberRows(members, fields))
	default:
		return w.Table(buildMemberRows(members, fields))
	}
}

func renderOrgUnits(out io.Writer, ous []google.OrgUnitInfo, fields []orgUnitFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(ous)
		}
		records := make([]map[string]any, 0, len(ous))
		for _, ou := range ous {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(ou)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildOrgUnitRows(ous, fields))
	default:
		return w.Table(buildOrgUnitRows(ous, fields))
	}
}

func renderAliases(out io.Writer, aliases []google.AliasInfo, fields []aliasFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(aliases)
		}
		records := make([]map[string]any, 0, len(aliases))
		for _, alias := range aliases {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(alias)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildAliasRows(aliases, fields))
	default:
		return w.Table(buildAliasRows(aliases, fields))
	}
}

func renderDomains(out io.Writer, domains []google.WorkspaceDomainInfo, fields []domainFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(domains)
		}
		records := make([]map[string]any, 0, len(domains))
		for _, domain := range domains {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(domain)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildDomainRows(domains, fields))
	default:
		return w.Table(buildDomainRows(domains, fields))
	}
}

func renderDomainAliases(out io.Writer, aliases []google.DomainAliasInfo, fields []domainAliasFieldSpec, flags map[string]string) error {
	format, err := outputFormatFromFlags(flags)
	if err != nil {
		return err
	}
	w := output.New(out)
	switch format {
	case formatJSON:
		if strings.TrimSpace(flags["fields"]) == "" {
			return w.JSON(aliases)
		}
		records := make([]map[string]any, 0, len(aliases))
		for _, alias := range aliases {
			record := map[string]any{}
			for _, field := range fields {
				record[field.key] = field.value(alias)
			}
			records = append(records, record)
		}
		return w.JSON(records)
	case formatCSV:
		return w.CSV(buildDomainAliasRows(aliases, fields))
	default:
		return w.Table(buildDomainAliasRows(aliases, fields))
	}
}

func printAliasResult(out io.Writer, alias google.AliasInfo, asJSON bool, label string) error {
	w := output.New(out)
	if asJSON {
		return w.JSON(alias)
	}
	w.Printf("%s: %s\n", label, alias.Alias)
	printStringField(w, "Primary email", alias.PrimaryEmail)
	printStringField(w, "ID", alias.ID)
	printStringField(w, "ETag", alias.Etag)
	printStringField(w, "Kind", alias.Kind)
	return nil
}

func printWorkspaceDomainResult(out io.Writer, domain google.WorkspaceDomainInfo, asJSON bool, label string) error {
	w := output.New(out)
	if asJSON {
		return w.JSON(domain)
	}
	w.Printf("%s: %s\n", label, domain.DomainName)
	printWorkspaceDomainFields(w, domain)
	return nil
}

func printGroupResult(out io.Writer, group google.GroupInfo, asJSON bool, label string) error {
	w := output.New(out)
	if asJSON {
		return w.JSON(group)
	}
	w.Printf("%s: %s\n", label, group.Email)
	printGroupDetails(w, group)
	return nil
}

func printOrgUnitResult(out io.Writer, ou google.OrgUnitInfo, asJSON bool, label string) error {
	w := output.New(out)
	if asJSON {
		return w.JSON(ou)
	}
	w.Printf("%s: %s\n", label, ou.OrgUnitPath)
	w.Printf("Name: %s\n", ou.Name)
	printStringField(w, "Description", ou.Description)
	printStringField(w, "Parent path", ou.ParentOrgUnitPath)
	printStringField(w, "ID", ou.OrgUnitID)
	printStringField(w, "Parent ID", ou.ParentOrgUnitID)
	w.Printf("Block inheritance: %t\n", ou.BlockInheritance)
	printStringField(w, "ETag", ou.Etag)
	printStringField(w, "Kind", ou.Kind)
	return nil
}

func (r Runner) printOrgUnits(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	ous, err := r.Directory.OrgUnits(ctx, profile)
	if err != nil {
		return explainAPICommandFailure(profile, "print ous", err)
	}
	fields, err := selectedOrgUnitFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildOrgUnitRows(ous, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws org units", rows); err != nil {
			return explainSheetExportFailure(profile, "print ous", err)
		}
		return nil
	}
	return renderOrgUnits(r.Stdout, ous, fields, flags)
}

func (r Runner) printUsers(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	opts, err := userListOptionsFromFlags(flags)
	if err != nil {
		return err
	}
	users, err := r.Directory.Users(ctx, profile, opts)
	if err != nil {
		return explainAPICommandFailure(profile, "print users", err)
	}
	fields, err := selectedUserFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildUserRows(users, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws users", rows); err != nil {
			return explainSheetExportFailure(profile, "print users", err)
		}
		return nil
	}
	return renderUsers(r.Stdout, users, fields, flags)
}

func (r Runner) setUserSuspended(ctx context.Context, args []string, flags map[string]string, suspended bool) error {
	action := "suspend"
	if !suspended {
		action = "unsuspend"
	}
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return fmt.Errorf("user email is required\n\nUsage: gws %s user user@example.com", action)
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	user, err := r.Directory.SetUserSuspended(ctx, profile, email, suspended)
	if err != nil {
		return explainAPICommandFailure(profile, action+" user", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(user)
	}
	if suspended {
		w.Printf("User suspended: %s\n", user.PrimaryEmail)
	} else {
		w.Printf("User unsuspended: %s\n", user.PrimaryEmail)
	}
	w.Printf("Suspended: %t\n", user.Suspended)
	return nil
}

func (r Runner) setUserAdmin(ctx context.Context, args []string, flags map[string]string, adminStatus bool) error {
	_ = flags
	action := "make admin"
	if !adminStatus {
		action = "revoke admin"
	}
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return fmt.Errorf("user email is required\n\nUsage: gws %s user@example.com", action)
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.SetUserAdmin(ctx, profile, email, adminStatus); err != nil {
		return explainAPICommandFailure(profile, action, err)
	}
	if adminStatus {
		output.New(r.Stdout).Printf("User made admin: %s\n", email)
	} else {
		output.New(r.Stdout).Printf("User admin revoked: %s\n", email)
	}
	return nil
}

func (r Runner) updateUser(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws update user user@example.com [--given-name NAME] [--family-name NAME] [--org-unit /PATH] [--recovery-email addr@example.com] [--recovery-phone +15551234567] [--change-password-at-next-login true|false] [--archived true|false] [--include-in-global-address-list true|false] [--phones-json JSON] [--addresses-json JSON] [--organizations-json JSON] [--locations-json JSON] [--relations-json JSON] [--external-ids-json JSON]")
	}
	givenName, hasGivenName := flags["given-name"]
	familyName, hasFamilyName := flags["family-name"]
	orgUnit, hasOrgUnit := flags["org-unit"]
	recoveryEmail, hasRecoveryEmail := flags["recovery-email"]
	recoveryPhone, hasRecoveryPhone := flags["recovery-phone"]
	changePassword, err := optionalBoolPointerFlag(flags, "change-password-at-next-login")
	if err != nil {
		return err
	}
	archived, err := optionalBoolPointerFlag(flags, "archived")
	if err != nil {
		return err
	}
	includeInGAL, err := optionalBoolPointerFlag(flags, "include-in-global-address-list")
	if err != nil {
		return err
	}
	phones, err := optionalJSONPointerFlag(flags, "phones-json")
	if err != nil {
		return err
	}
	addresses, err := optionalJSONPointerFlag(flags, "addresses-json")
	if err != nil {
		return err
	}
	organizations, err := optionalJSONPointerFlag(flags, "organizations-json")
	if err != nil {
		return err
	}
	locations, err := optionalJSONPointerFlag(flags, "locations-json")
	if err != nil {
		return err
	}
	relations, err := optionalJSONPointerFlag(flags, "relations-json")
	if err != nil {
		return err
	}
	externalIDs, err := optionalJSONPointerFlag(flags, "external-ids-json")
	if err != nil {
		return err
	}
	if !hasGivenName && !hasFamilyName && !hasOrgUnit && !hasRecoveryEmail && !hasRecoveryPhone &&
		changePassword == nil && archived == nil && includeInGAL == nil &&
		phones == nil && addresses == nil && organizations == nil && locations == nil &&
		relations == nil && externalIDs == nil {
		return errors.New("nothing to update; provide one or more user update flags")
	}
	update := google.UserUpdate{
		GivenName:                  strings.TrimSpace(givenName),
		FamilyName:                 strings.TrimSpace(familyName),
		ChangePasswordAtNextLogin:  changePassword,
		Archived:                   archived,
		IncludeInGlobalAddressList: includeInGAL,
		Phones:                     phones,
		Addresses:                  addresses,
		Organizations:              organizations,
		Locations:                  locations,
		Relations:                  relations,
		ExternalIDs:                externalIDs,
	}
	if hasOrgUnit {
		update.OrgUnitPath = normalizeOrgUnitPath(orgUnit)
	}
	if hasRecoveryEmail {
		value := strings.TrimSpace(recoveryEmail)
		update.RecoveryEmail = &value
	}
	if hasRecoveryPhone {
		value := strings.TrimSpace(recoveryPhone)
		update.RecoveryPhone = &value
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	user, err := r.Directory.UpdateUser(ctx, profile, email, update)
	if err != nil {
		return explainAPICommandFailure(profile, "update user", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(user)
	}
	w.Printf("User updated: %s\n", user.PrimaryEmail)
	printUserDetails(w, user)
	return nil
}

func (r Runner) deleteUser(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws delete user user@example.com --confirm")
	}
	if err := requireConfirm(flags, "delete user"); err != nil {
		return err
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteUser(ctx, profile, email); err != nil {
		return explainAPICommandFailure(profile, "delete user", err)
	}
	output.New(r.Stdout).Printf("User deleted: %s\n", email)
	return nil
}

func (r Runner) printUserAliases(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws print user-aliases user@example.com")
	}
	userEmail := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	aliases, err := r.Directory.UserAliases(ctx, profile, userEmail)
	if err != nil {
		return explainAPICommandFailure(profile, "print user-aliases", err)
	}
	fields, err := selectedAliasFields(flags["fields"])
	if err != nil {
		return err
	}
	if _, err := outputFormatFromFlags(flags); err != nil {
		return err
	}
	if flags["sheet"] == "true" {
		rows := buildAliasRows(aliases, fields)
		if err := r.exportRowsToSheet(ctx, profile, "gws user aliases", rows); err != nil {
			return explainSheetExportFailure(profile, "print user-aliases", err)
		}
		return nil
	}
	return renderAliases(r.Stdout, aliases, fields, flags)
}

func (r Runner) createUserAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("user email and alias email are required\n\nUsage: gws create user-alias user@example.com alias@example.com")
	}
	userEmail := strings.TrimSpace(args[2])
	aliasEmail := strings.TrimSpace(args[3])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	alias, err := r.Directory.CreateUserAlias(ctx, profile, userEmail, aliasEmail)
	if err != nil {
		return explainAPICommandFailure(profile, "create user-alias", err)
	}
	return printAliasResult(r.Stdout, alias, flags["json"] == "true", "User alias created")
}

func (r Runner) deleteUserAlias(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 4 || strings.TrimSpace(args[2]) == "" || strings.TrimSpace(args[3]) == "" {
		return errors.New("user email and alias email are required\n\nUsage: gws delete user-alias user@example.com alias@example.com --confirm")
	}
	if err := requireConfirm(flags, "delete user-alias"); err != nil {
		return err
	}
	userEmail := strings.TrimSpace(args[2])
	aliasEmail := strings.TrimSpace(args[3])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	if err := r.Directory.DeleteUserAlias(ctx, profile, userEmail, aliasEmail); err != nil {
		return explainAPICommandFailure(profile, "delete user-alias", err)
	}
	output.New(r.Stdout).Printf("User alias deleted: %s\n", aliasEmail)
	return nil
}

func printUserDetails(w output.Writer, user google.UserInfo) {
	w.Printf("Primary email: %s\n", user.PrimaryEmail)
	printStringField(w, "ID", user.ID)
	printStringField(w, "Customer ID", user.CustomerID)
	printStringField(w, "Name", user.Name)
	printStringField(w, "Given name", user.GivenName)
	printStringField(w, "Family name", user.FamilyName)
	printStringField(w, "Org unit", user.OrgUnitPath)
	w.Printf("Suspended: %t\n", user.Suspended)
	printStringField(w, "Suspension reason", user.SuspensionReason)
	w.Printf("Archived: %t\n", user.IsArchived)
	w.Printf("Admin: %t\n", user.IsAdmin)
	w.Printf("Delegated admin: %t\n", user.IsDelegatedAdmin)
	w.Printf("Enrolled in 2SV: %t\n", user.IsEnrolledIn2SV)
	w.Printf("Enforced in 2SV: %t\n", user.IsEnforcedIn2SV)
	w.Printf("Mailbox setup: %t\n", user.IsMailboxSetup)
	w.Printf("Guest user: %t\n", user.IsGuestUser)
	w.Printf("Directory visible: %t\n", user.IncludeInGlobalAddressList)
	w.Printf("Agreed to terms: %t\n", user.AgreedToTerms)
	w.Printf("Change password at next login: %t\n", user.ChangePasswordAtNextLogin)
	w.Printf("IP whitelisted: %t\n", user.IPWhitelisted)
	printStringField(w, "Recovery email", user.RecoveryEmail)
	printStringField(w, "Recovery phone", user.RecoveryPhone)
	printStringField(w, "Created", user.CreationTime)
	printStringField(w, "Last login", user.LastLoginTime)
	printStringField(w, "Deleted", user.DeletionTime)
	printStringField(w, "Thumbnail photo URL", user.ThumbnailPhotoURL)
	printStringField(w, "Thumbnail photo etag", user.ThumbnailPhotoEtag)
	printStringSliceField(w, "Aliases", user.Aliases)
	printStringSliceField(w, "Non-editable aliases", user.NonEditableAliases)
	printStructuredField(w, "Emails", user.Emails)
	printStructuredField(w, "Phones", user.Phones)
	printStructuredField(w, "Addresses", user.Addresses)
	printStructuredField(w, "Organizations", user.Organizations)
	printStructuredField(w, "Relations", user.Relations)
	printStructuredField(w, "External IDs", user.ExternalIDs)
	printStructuredField(w, "Locations", user.Locations)
	printStructuredField(w, "Custom schemas", user.CustomSchemas)
}

func printDomainDetails(w output.Writer, info google.DomainInfo) {
	w.Printf("Customer ID: %s\n", info.CustomerID)
	w.Printf("Primary domain: %s\n", info.PrimaryDomain)
	printStringField(w, "Alternate email", info.AlternateEmail)
	printStringField(w, "Customer created", info.CustomerCreationTime)
	printStringField(w, "Language", info.Language)
	printStringField(w, "Phone number", info.PhoneNumber)
	printStringField(w, "ETag", info.Etag)
	printStringField(w, "Kind", info.Kind)
}

func printWorkspaceDomainDetails(w output.Writer, domain google.WorkspaceDomainInfo) {
	w.Printf("Domain: %s\n", domain.DomainName)
	printWorkspaceDomainFields(w, domain)
}

func printWorkspaceDomainFields(w output.Writer, domain google.WorkspaceDomainInfo) {
	w.Printf("Primary: %t\n", domain.IsPrimary)
	w.Printf("Verified: %t\n", domain.Verified)
	if domain.CreationTime != 0 {
		w.Printf("Created: %d\n", domain.CreationTime)
	}
	if len(domain.DomainAliases) > 0 {
		w.Println("Domain aliases:")
		for _, alias := range domain.DomainAliases {
			w.Printf("  %s (verified: %t)\n", alias.DomainAliasName, alias.Verified)
		}
	}
	printStringField(w, "ETag", domain.Etag)
	printStringField(w, "Kind", domain.Kind)
}

func printDomainAliasResult(out io.Writer, alias google.DomainAliasInfo, asJSON bool, label string) error {
	w := output.New(out)
	if asJSON {
		return w.JSON(alias)
	}
	w.Printf("%s: %s\n", label, alias.DomainAliasName)
	printDomainAliasFields(w, alias)
	return nil
}

func printDomainAliasDetails(w output.Writer, alias google.DomainAliasInfo) {
	w.Printf("Domain alias: %s\n", alias.DomainAliasName)
	printDomainAliasFields(w, alias)
}

func printDomainAliasFields(w output.Writer, alias google.DomainAliasInfo) {
	printStringField(w, "Parent domain", alias.ParentDomainName)
	w.Printf("Verified: %t\n", alias.Verified)
	if alias.CreationTime != 0 {
		w.Printf("Created: %d\n", alias.CreationTime)
	}
	printStringField(w, "ETag", alias.Etag)
	printStringField(w, "Kind", alias.Kind)
}

func printGroupDetails(w output.Writer, group google.GroupInfo) {
	w.Printf("Email: %s\n", group.Email)
	printStringField(w, "ID", group.ID)
	printStringField(w, "Name", group.Name)
	printStringField(w, "Description", group.Description)
	w.Printf("Direct members: %d\n", group.DirectMembersCount)
	w.Printf("Admin created: %t\n", group.AdminCreated)
	printStringSliceField(w, "Aliases", group.Aliases)
	printStringSliceField(w, "Non-editable aliases", group.NonEditableAliases)
	printStringField(w, "ETag", group.Etag)
	printStringField(w, "Kind", group.Kind)
}

func printGmailDelegateDetails(w output.Writer, delegate google.DelegateInfo) {
	w.Printf("Delegate email: %s\n", delegate.DelegateEmail)
	printStringField(w, "Verification status", delegate.VerificationStatus)
}

func printMemberDetails(w output.Writer, member google.MemberInfo) {
	w.Printf("Email: %s\n", member.Email)
	printStringField(w, "ID", member.ID)
	printStringField(w, "Role", member.Role)
	printStringField(w, "Type", member.Type)
	printStringField(w, "Status", member.Status)
	printStringField(w, "Delivery settings", member.DeliverySettings)
	printStringField(w, "ETag", member.Etag)
	printStringField(w, "Kind", member.Kind)
}

func printOrgUnitDetails(w output.Writer, ou google.OrgUnitInfo) {
	w.Printf("Path: %s\n", ou.OrgUnitPath)
	w.Printf("Name: %s\n", ou.Name)
	printStringField(w, "Description", ou.Description)
	printStringField(w, "Parent path", ou.ParentOrgUnitPath)
	printStringField(w, "ID", ou.OrgUnitID)
	printStringField(w, "Parent ID", ou.ParentOrgUnitID)
	w.Printf("Block inheritance: %t\n", ou.BlockInheritance)
	printStringField(w, "ETag", ou.Etag)
	printStringField(w, "Kind", ou.Kind)
}

func printStringField(w output.Writer, label string, value string) {
	if value != "" {
		w.Printf("%s: %s\n", label, value)
	}
}

func printStringSliceField(w output.Writer, label string, values []string) {
	if len(values) > 0 {
		w.Printf("%s: %s\n", label, strings.Join(values, ", "))
	}
}

func printStructuredField(w output.Writer, label string, value any) {
	text := prettyJSON(value)
	if text == "" {
		return
	}
	lines := strings.Split(text, "\n")
	if len(lines) == 1 {
		w.Printf("%s: %s\n", label, lines[0])
		return
	}
	w.Printf("%s:\n", label)
	for _, line := range lines {
		w.Printf("  %s\n", line)
	}
}

func printSyncSummary(w output.Writer, label string, count int) {
	w.Printf("%s: %d\n", label, count)
}

func printRoleChanges(w output.Writer, label string, changes []memberRoleChange) {
	if len(changes) == 0 {
		return
	}
	parts := make([]string, 0, len(changes))
	for _, change := range changes {
		parts = append(parts, fmt.Sprintf("%s (%s -> %s)", change.Email, change.FromRole, change.ToRole))
	}
	w.Printf("%s: %s\n", label, strings.Join(parts, ", "))
}

func printRoleTargets(w output.Writer, label string, targets []memberRoleTarget) {
	if len(targets) == 0 {
		return
	}
	parts := make([]string, 0, len(targets))
	for _, target := range targets {
		parts = append(parts, fmt.Sprintf("%s (%s)", target.Email, target.Role))
	}
	w.Printf("%s: %s\n", label, strings.Join(parts, ", "))
}

func outputFormatFromFlags(flags map[string]string) (outputFormat, error) {
	if flags["sheet"] == "true" {
		if flags["json"] == "true" {
			return "", fmt.Errorf("--sheet cannot be combined with --json")
		}
		if format := strings.TrimSpace(flags["format"]); format != "" && format != string(formatText) {
			return "", fmt.Errorf("--sheet cannot be combined with --format=%s", format)
		}
		return formatText, nil
	}
	if flags["json"] == "true" {
		if format := strings.TrimSpace(flags["format"]); format != "" && format != string(formatJSON) {
			return "", fmt.Errorf("--json cannot be combined with --format=%s", format)
		}
		return formatJSON, nil
	}
	switch strings.ToLower(strings.TrimSpace(flags["format"])) {
	case "", string(formatText):
		return formatText, nil
	case string(formatJSON):
		return formatJSON, nil
	case string(formatCSV):
		return formatCSV, nil
	default:
		return "", fmt.Errorf("--format must be text, csv, or json")
	}
}

func defaultUserFields() []userFieldSpec {
	return []userFieldSpec{
		userFieldByKey("primaryEmail"),
		userFieldByKey("name"),
		userFieldByKey("suspended"),
		userFieldByKey("orgUnitPath"),
	}
}

func allUserFields() []userFieldSpec {
	return []userFieldSpec{
		userFieldByKey("primaryEmail"),
		userFieldByKey("name"),
		userFieldByKey("givenName"),
		userFieldByKey("familyName"),
		userFieldByKey("suspended"),
		userFieldByKey("archived"),
		userFieldByKey("orgUnitPath"),
		userFieldByKey("isAdmin"),
		userFieldByKey("isDelegatedAdmin"),
		userFieldByKey("isEnrolledIn2SV"),
		userFieldByKey("isEnforcedIn2SV"),
		userFieldByKey("isMailboxSetup"),
		userFieldByKey("includeInGlobalAddressList"),
		userFieldByKey("aliases"),
		userFieldByKey("nonEditableAliases"),
		userFieldByKey("recoveryEmail"),
		userFieldByKey("recoveryPhone"),
		userFieldByKey("suspensionReason"),
		userFieldByKey("isGuestUser"),
		userFieldByKey("agreedToTerms"),
		userFieldByKey("changePasswordAtNextLogin"),
		userFieldByKey("ipWhitelisted"),
		userFieldByKey("thumbnailPhotoURL"),
		userFieldByKey("emails"),
		userFieldByKey("phones"),
		userFieldByKey("addresses"),
		userFieldByKey("organizations"),
		userFieldByKey("relations"),
		userFieldByKey("externalIDs"),
		userFieldByKey("locations"),
		userFieldByKey("id"),
		userFieldByKey("creationTime"),
		userFieldByKey("lastLoginTime"),
		userFieldByKey("deletionTime"),
	}
}

func userFieldByKey(key string) userFieldSpec {
	switch key {
	case "primaryEmail":
		return userFieldSpec{key: key, header: "Primary Email", text: func(u google.UserInfo) string { return u.PrimaryEmail }, value: func(u google.UserInfo) any { return u.PrimaryEmail }}
	case "name":
		return userFieldSpec{key: key, header: "Name", text: func(u google.UserInfo) string { return u.Name }, value: func(u google.UserInfo) any { return u.Name }}
	case "givenName":
		return userFieldSpec{key: key, header: "Given Name", text: func(u google.UserInfo) string { return u.GivenName }, value: func(u google.UserInfo) any { return u.GivenName }}
	case "familyName":
		return userFieldSpec{key: key, header: "Family Name", text: func(u google.UserInfo) string { return u.FamilyName }, value: func(u google.UserInfo) any { return u.FamilyName }}
	case "suspended":
		return userFieldSpec{key: key, header: "Suspended", text: func(u google.UserInfo) string { return strconv.FormatBool(u.Suspended) }, value: func(u google.UserInfo) any { return u.Suspended }}
	case "archived":
		return userFieldSpec{key: key, header: "Archived", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsArchived) }, value: func(u google.UserInfo) any { return u.IsArchived }}
	case "orgUnitPath":
		return userFieldSpec{key: key, header: "Org Unit", text: func(u google.UserInfo) string { return u.OrgUnitPath }, value: func(u google.UserInfo) any { return u.OrgUnitPath }}
	case "isAdmin":
		return userFieldSpec{key: key, header: "Admin", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsAdmin) }, value: func(u google.UserInfo) any { return u.IsAdmin }}
	case "isDelegatedAdmin":
		return userFieldSpec{key: key, header: "Delegated Admin", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsDelegatedAdmin) }, value: func(u google.UserInfo) any { return u.IsDelegatedAdmin }}
	case "isEnrolledIn2SV":
		return userFieldSpec{key: key, header: "Enrolled In 2SV", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsEnrolledIn2SV) }, value: func(u google.UserInfo) any { return u.IsEnrolledIn2SV }}
	case "isEnforcedIn2SV":
		return userFieldSpec{key: key, header: "Enforced In 2SV", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsEnforcedIn2SV) }, value: func(u google.UserInfo) any { return u.IsEnforcedIn2SV }}
	case "isMailboxSetup":
		return userFieldSpec{key: key, header: "Mailbox Setup", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsMailboxSetup) }, value: func(u google.UserInfo) any { return u.IsMailboxSetup }}
	case "includeInGlobalAddressList":
		return userFieldSpec{key: key, header: "Directory Visible", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IncludeInGlobalAddressList) }, value: func(u google.UserInfo) any { return u.IncludeInGlobalAddressList }}
	case "aliases":
		return userFieldSpec{key: key, header: "Aliases", text: func(u google.UserInfo) string { return strings.Join(u.Aliases, ",") }, value: func(u google.UserInfo) any { return append([]string(nil), u.Aliases...) }}
	case "nonEditableAliases":
		return userFieldSpec{key: key, header: "Non-Editable Aliases", text: func(u google.UserInfo) string { return strings.Join(u.NonEditableAliases, ",") }, value: func(u google.UserInfo) any { return append([]string(nil), u.NonEditableAliases...) }}
	case "recoveryEmail":
		return userFieldSpec{key: key, header: "Recovery Email", text: func(u google.UserInfo) string { return u.RecoveryEmail }, value: func(u google.UserInfo) any { return u.RecoveryEmail }}
	case "recoveryPhone":
		return userFieldSpec{key: key, header: "Recovery Phone", text: func(u google.UserInfo) string { return u.RecoveryPhone }, value: func(u google.UserInfo) any { return u.RecoveryPhone }}
	case "suspensionReason":
		return userFieldSpec{key: key, header: "Suspension Reason", text: func(u google.UserInfo) string { return u.SuspensionReason }, value: func(u google.UserInfo) any { return u.SuspensionReason }}
	case "isGuestUser":
		return userFieldSpec{key: key, header: "Guest User", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IsGuestUser) }, value: func(u google.UserInfo) any { return u.IsGuestUser }}
	case "agreedToTerms":
		return userFieldSpec{key: key, header: "Agreed To Terms", text: func(u google.UserInfo) string { return strconv.FormatBool(u.AgreedToTerms) }, value: func(u google.UserInfo) any { return u.AgreedToTerms }}
	case "changePasswordAtNextLogin":
		return userFieldSpec{key: key, header: "Change Password Next Login", text: func(u google.UserInfo) string { return strconv.FormatBool(u.ChangePasswordAtNextLogin) }, value: func(u google.UserInfo) any { return u.ChangePasswordAtNextLogin }}
	case "ipWhitelisted":
		return userFieldSpec{key: key, header: "IP Whitelisted", text: func(u google.UserInfo) string { return strconv.FormatBool(u.IPWhitelisted) }, value: func(u google.UserInfo) any { return u.IPWhitelisted }}
	case "thumbnailPhotoURL":
		return userFieldSpec{key: key, header: "Thumbnail Photo URL", text: func(u google.UserInfo) string { return u.ThumbnailPhotoURL }, value: func(u google.UserInfo) any { return u.ThumbnailPhotoURL }}
	case "emails":
		return userFieldSpec{key: key, header: "Emails", text: func(u google.UserInfo) string { return compactJSON(u.Emails) }, value: func(u google.UserInfo) any { return u.Emails }}
	case "phones":
		return userFieldSpec{key: key, header: "Phones", text: func(u google.UserInfo) string { return compactJSON(u.Phones) }, value: func(u google.UserInfo) any { return u.Phones }}
	case "addresses":
		return userFieldSpec{key: key, header: "Addresses", text: func(u google.UserInfo) string { return compactJSON(u.Addresses) }, value: func(u google.UserInfo) any { return u.Addresses }}
	case "organizations":
		return userFieldSpec{key: key, header: "Organizations", text: func(u google.UserInfo) string { return compactJSON(u.Organizations) }, value: func(u google.UserInfo) any { return u.Organizations }}
	case "relations":
		return userFieldSpec{key: key, header: "Relations", text: func(u google.UserInfo) string { return compactJSON(u.Relations) }, value: func(u google.UserInfo) any { return u.Relations }}
	case "externalIDs":
		return userFieldSpec{key: key, header: "External IDs", text: func(u google.UserInfo) string { return compactJSON(u.ExternalIDs) }, value: func(u google.UserInfo) any { return u.ExternalIDs }}
	case "locations":
		return userFieldSpec{key: key, header: "Locations", text: func(u google.UserInfo) string { return compactJSON(u.Locations) }, value: func(u google.UserInfo) any { return u.Locations }}
	case "id":
		return userFieldSpec{key: key, header: "ID", text: func(u google.UserInfo) string { return u.ID }, value: func(u google.UserInfo) any { return u.ID }}
	case "creationTime":
		return userFieldSpec{key: key, header: "Created", text: func(u google.UserInfo) string { return u.CreationTime }, value: func(u google.UserInfo) any { return u.CreationTime }}
	case "lastLoginTime":
		return userFieldSpec{key: key, header: "Last Login", text: func(u google.UserInfo) string { return u.LastLoginTime }, value: func(u google.UserInfo) any { return u.LastLoginTime }}
	case "deletionTime":
		return userFieldSpec{key: key, header: "Deleted", text: func(u google.UserInfo) string { return u.DeletionTime }, value: func(u google.UserInfo) any { return u.DeletionTime }}
	default:
		return userFieldSpec{}
	}
}

func selectedUserFields(value string) ([]userFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultUserFields(), nil
	}
	fields := []userFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := userFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown user field %q; valid fields: %s", key, joinUserFieldKeys(allUserFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func defaultGroupFields() []groupFieldSpec {
	return []groupFieldSpec{
		groupFieldByKey("email"),
		groupFieldByKey("name"),
		groupFieldByKey("directMembersCount"),
		groupFieldByKey("adminCreated"),
	}
}

func defaultMemberFields() []memberFieldSpec {
	return []memberFieldSpec{
		memberFieldByKey("email"),
		memberFieldByKey("role"),
		memberFieldByKey("type"),
		memberFieldByKey("status"),
	}
}

func allMemberFields() []memberFieldSpec {
	return []memberFieldSpec{
		memberFieldByKey("email"),
		memberFieldByKey("role"),
		memberFieldByKey("type"),
		memberFieldByKey("status"),
		memberFieldByKey("deliverySettings"),
		memberFieldByKey("id"),
		memberFieldByKey("etag"),
		memberFieldByKey("kind"),
	}
}

func memberFieldByKey(key string) memberFieldSpec {
	switch key {
	case "email":
		return memberFieldSpec{key: key, header: "Email", text: func(m google.MemberInfo) string { return m.Email }, value: func(m google.MemberInfo) any { return m.Email }}
	case "role":
		return memberFieldSpec{key: key, header: "Role", text: func(m google.MemberInfo) string { return m.Role }, value: func(m google.MemberInfo) any { return m.Role }}
	case "type":
		return memberFieldSpec{key: key, header: "Type", text: func(m google.MemberInfo) string { return m.Type }, value: func(m google.MemberInfo) any { return m.Type }}
	case "status":
		return memberFieldSpec{key: key, header: "Status", text: func(m google.MemberInfo) string { return m.Status }, value: func(m google.MemberInfo) any { return m.Status }}
	case "deliverySettings":
		return memberFieldSpec{key: key, header: "Delivery Settings", text: func(m google.MemberInfo) string { return m.DeliverySettings }, value: func(m google.MemberInfo) any { return m.DeliverySettings }}
	case "id":
		return memberFieldSpec{key: key, header: "ID", text: func(m google.MemberInfo) string { return m.ID }, value: func(m google.MemberInfo) any { return m.ID }}
	case "etag":
		return memberFieldSpec{key: key, header: "ETag", text: func(m google.MemberInfo) string { return m.Etag }, value: func(m google.MemberInfo) any { return m.Etag }}
	case "kind":
		return memberFieldSpec{key: key, header: "Kind", text: func(m google.MemberInfo) string { return m.Kind }, value: func(m google.MemberInfo) any { return m.Kind }}
	default:
		return memberFieldSpec{}
	}
}

func selectedMemberFields(value string) ([]memberFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultMemberFields(), nil
	}
	fields := []memberFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := memberFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown group member field %q; valid fields: %s", key, joinMemberFieldKeys(allMemberFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func defaultOrgUnitFields() []orgUnitFieldSpec {
	return []orgUnitFieldSpec{
		orgUnitFieldByKey("orgUnitPath"),
		orgUnitFieldByKey("name"),
		orgUnitFieldByKey("parentOrgUnitPath"),
		orgUnitFieldByKey("description"),
	}
}

func allOrgUnitFields() []orgUnitFieldSpec {
	return []orgUnitFieldSpec{
		orgUnitFieldByKey("orgUnitPath"),
		orgUnitFieldByKey("name"),
		orgUnitFieldByKey("parentOrgUnitPath"),
		orgUnitFieldByKey("description"),
		orgUnitFieldByKey("orgUnitID"),
		orgUnitFieldByKey("parentOrgUnitID"),
		orgUnitFieldByKey("blockInheritance"),
		orgUnitFieldByKey("etag"),
		orgUnitFieldByKey("kind"),
	}
}

func orgUnitFieldByKey(key string) orgUnitFieldSpec {
	switch key {
	case "orgUnitPath":
		return orgUnitFieldSpec{key: key, header: "Path", text: func(o google.OrgUnitInfo) string { return o.OrgUnitPath }, value: func(o google.OrgUnitInfo) any { return o.OrgUnitPath }}
	case "name":
		return orgUnitFieldSpec{key: key, header: "Name", text: func(o google.OrgUnitInfo) string { return o.Name }, value: func(o google.OrgUnitInfo) any { return o.Name }}
	case "parentOrgUnitPath":
		return orgUnitFieldSpec{key: key, header: "Parent Path", text: func(o google.OrgUnitInfo) string { return o.ParentOrgUnitPath }, value: func(o google.OrgUnitInfo) any { return o.ParentOrgUnitPath }}
	case "description":
		return orgUnitFieldSpec{key: key, header: "Description", text: func(o google.OrgUnitInfo) string { return o.Description }, value: func(o google.OrgUnitInfo) any { return o.Description }}
	case "orgUnitID":
		return orgUnitFieldSpec{key: key, header: "ID", text: func(o google.OrgUnitInfo) string { return o.OrgUnitID }, value: func(o google.OrgUnitInfo) any { return o.OrgUnitID }}
	case "parentOrgUnitID":
		return orgUnitFieldSpec{key: key, header: "Parent ID", text: func(o google.OrgUnitInfo) string { return o.ParentOrgUnitID }, value: func(o google.OrgUnitInfo) any { return o.ParentOrgUnitID }}
	case "blockInheritance":
		return orgUnitFieldSpec{key: key, header: "Block Inheritance", text: func(o google.OrgUnitInfo) string { return strconv.FormatBool(o.BlockInheritance) }, value: func(o google.OrgUnitInfo) any { return o.BlockInheritance }}
	case "etag":
		return orgUnitFieldSpec{key: key, header: "ETag", text: func(o google.OrgUnitInfo) string { return o.Etag }, value: func(o google.OrgUnitInfo) any { return o.Etag }}
	case "kind":
		return orgUnitFieldSpec{key: key, header: "Kind", text: func(o google.OrgUnitInfo) string { return o.Kind }, value: func(o google.OrgUnitInfo) any { return o.Kind }}
	default:
		return orgUnitFieldSpec{}
	}
}

func selectedOrgUnitFields(value string) ([]orgUnitFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultOrgUnitFields(), nil
	}
	fields := []orgUnitFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := orgUnitFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown org unit field %q; valid fields: %s", key, joinOrgUnitFieldKeys(allOrgUnitFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func defaultAliasFields() []aliasFieldSpec {
	return []aliasFieldSpec{
		aliasFieldByKey("alias"),
		aliasFieldByKey("primaryEmail"),
		aliasFieldByKey("id"),
	}
}

func allAliasFields() []aliasFieldSpec {
	return []aliasFieldSpec{
		aliasFieldByKey("alias"),
		aliasFieldByKey("primaryEmail"),
		aliasFieldByKey("id"),
		aliasFieldByKey("etag"),
		aliasFieldByKey("kind"),
	}
}

func aliasFieldByKey(key string) aliasFieldSpec {
	switch key {
	case "alias":
		return aliasFieldSpec{key: key, header: "Alias", text: func(a google.AliasInfo) string { return a.Alias }, value: func(a google.AliasInfo) any { return a.Alias }}
	case "primaryEmail":
		return aliasFieldSpec{key: key, header: "Primary Email", text: func(a google.AliasInfo) string { return a.PrimaryEmail }, value: func(a google.AliasInfo) any { return a.PrimaryEmail }}
	case "id":
		return aliasFieldSpec{key: key, header: "ID", text: func(a google.AliasInfo) string { return a.ID }, value: func(a google.AliasInfo) any { return a.ID }}
	case "etag":
		return aliasFieldSpec{key: key, header: "ETag", text: func(a google.AliasInfo) string { return a.Etag }, value: func(a google.AliasInfo) any { return a.Etag }}
	case "kind":
		return aliasFieldSpec{key: key, header: "Kind", text: func(a google.AliasInfo) string { return a.Kind }, value: func(a google.AliasInfo) any { return a.Kind }}
	default:
		return aliasFieldSpec{}
	}
}

func selectedAliasFields(value string) ([]aliasFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultAliasFields(), nil
	}
	fields := []aliasFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := aliasFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown alias field %q; valid fields: %s", key, joinAliasFieldKeys(allAliasFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func defaultDomainFields() []domainFieldSpec {
	return []domainFieldSpec{
		domainFieldByKey("domainName"),
		domainFieldByKey("isPrimary"),
		domainFieldByKey("verified"),
		domainFieldByKey("aliasCount"),
	}
}

func allDomainFields() []domainFieldSpec {
	return []domainFieldSpec{
		domainFieldByKey("domainName"),
		domainFieldByKey("isPrimary"),
		domainFieldByKey("verified"),
		domainFieldByKey("aliasCount"),
		domainFieldByKey("creationTime"),
		domainFieldByKey("etag"),
		domainFieldByKey("kind"),
	}
}

func domainFieldByKey(key string) domainFieldSpec {
	switch key {
	case "domainName":
		return domainFieldSpec{key: key, header: "Domain", text: func(d google.WorkspaceDomainInfo) string { return d.DomainName }, value: func(d google.WorkspaceDomainInfo) any { return d.DomainName }}
	case "isPrimary":
		return domainFieldSpec{key: key, header: "Primary", text: func(d google.WorkspaceDomainInfo) string { return strconv.FormatBool(d.IsPrimary) }, value: func(d google.WorkspaceDomainInfo) any { return d.IsPrimary }}
	case "verified":
		return domainFieldSpec{key: key, header: "Verified", text: func(d google.WorkspaceDomainInfo) string { return strconv.FormatBool(d.Verified) }, value: func(d google.WorkspaceDomainInfo) any { return d.Verified }}
	case "aliasCount":
		return domainFieldSpec{key: key, header: "Aliases", text: func(d google.WorkspaceDomainInfo) string { return strconv.Itoa(len(d.DomainAliases)) }, value: func(d google.WorkspaceDomainInfo) any { return len(d.DomainAliases) }}
	case "creationTime":
		return domainFieldSpec{key: key, header: "Creation Time", text: func(d google.WorkspaceDomainInfo) string { return strconv.FormatInt(d.CreationTime, 10) }, value: func(d google.WorkspaceDomainInfo) any { return d.CreationTime }}
	case "etag":
		return domainFieldSpec{key: key, header: "ETag", text: func(d google.WorkspaceDomainInfo) string { return d.Etag }, value: func(d google.WorkspaceDomainInfo) any { return d.Etag }}
	case "kind":
		return domainFieldSpec{key: key, header: "Kind", text: func(d google.WorkspaceDomainInfo) string { return d.Kind }, value: func(d google.WorkspaceDomainInfo) any { return d.Kind }}
	default:
		return domainFieldSpec{}
	}
}

func selectedDomainFields(value string) ([]domainFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultDomainFields(), nil
	}
	fields := []domainFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := domainFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown domain field %q; valid fields: %s", key, joinDomainFieldKeys(allDomainFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func defaultDomainAliasFields() []domainAliasFieldSpec {
	return []domainAliasFieldSpec{
		domainAliasFieldByKey("domainAliasName"),
		domainAliasFieldByKey("parentDomainName"),
		domainAliasFieldByKey("verified"),
	}
}

func allDomainAliasFields() []domainAliasFieldSpec {
	return []domainAliasFieldSpec{
		domainAliasFieldByKey("domainAliasName"),
		domainAliasFieldByKey("parentDomainName"),
		domainAliasFieldByKey("verified"),
		domainAliasFieldByKey("creationTime"),
		domainAliasFieldByKey("etag"),
		domainAliasFieldByKey("kind"),
	}
}

func domainAliasFieldByKey(key string) domainAliasFieldSpec {
	switch key {
	case "domainAliasName":
		return domainAliasFieldSpec{key: key, header: "Alias", text: func(a google.DomainAliasInfo) string { return a.DomainAliasName }, value: func(a google.DomainAliasInfo) any { return a.DomainAliasName }}
	case "parentDomainName":
		return domainAliasFieldSpec{key: key, header: "Parent Domain", text: func(a google.DomainAliasInfo) string { return a.ParentDomainName }, value: func(a google.DomainAliasInfo) any { return a.ParentDomainName }}
	case "verified":
		return domainAliasFieldSpec{key: key, header: "Verified", text: func(a google.DomainAliasInfo) string { return strconv.FormatBool(a.Verified) }, value: func(a google.DomainAliasInfo) any { return a.Verified }}
	case "creationTime":
		return domainAliasFieldSpec{key: key, header: "Creation Time", text: func(a google.DomainAliasInfo) string { return strconv.FormatInt(a.CreationTime, 10) }, value: func(a google.DomainAliasInfo) any { return a.CreationTime }}
	case "etag":
		return domainAliasFieldSpec{key: key, header: "ETag", text: func(a google.DomainAliasInfo) string { return a.Etag }, value: func(a google.DomainAliasInfo) any { return a.Etag }}
	case "kind":
		return domainAliasFieldSpec{key: key, header: "Kind", text: func(a google.DomainAliasInfo) string { return a.Kind }, value: func(a google.DomainAliasInfo) any { return a.Kind }}
	default:
		return domainAliasFieldSpec{}
	}
}

func selectedDomainAliasFields(value string) ([]domainAliasFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultDomainAliasFields(), nil
	}
	fields := []domainAliasFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := domainAliasFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown domain alias field %q; valid fields: %s", key, joinDomainAliasFieldKeys(allDomainAliasFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func allGroupFields() []groupFieldSpec {
	return []groupFieldSpec{
		groupFieldByKey("email"),
		groupFieldByKey("name"),
		groupFieldByKey("description"),
		groupFieldByKey("directMembersCount"),
		groupFieldByKey("adminCreated"),
		groupFieldByKey("aliases"),
		groupFieldByKey("nonEditableAliases"),
		groupFieldByKey("id"),
		groupFieldByKey("etag"),
		groupFieldByKey("kind"),
	}
}

func groupFieldByKey(key string) groupFieldSpec {
	switch key {
	case "email":
		return groupFieldSpec{key: key, header: "Email", text: func(g google.GroupInfo) string { return g.Email }, value: func(g google.GroupInfo) any { return g.Email }}
	case "name":
		return groupFieldSpec{key: key, header: "Name", text: func(g google.GroupInfo) string { return g.Name }, value: func(g google.GroupInfo) any { return g.Name }}
	case "description":
		return groupFieldSpec{key: key, header: "Description", text: func(g google.GroupInfo) string { return g.Description }, value: func(g google.GroupInfo) any { return g.Description }}
	case "directMembersCount":
		return groupFieldSpec{key: key, header: "Direct Members", text: func(g google.GroupInfo) string { return strconv.FormatInt(g.DirectMembersCount, 10) }, value: func(g google.GroupInfo) any { return g.DirectMembersCount }}
	case "adminCreated":
		return groupFieldSpec{key: key, header: "Admin Created", text: func(g google.GroupInfo) string { return strconv.FormatBool(g.AdminCreated) }, value: func(g google.GroupInfo) any { return g.AdminCreated }}
	case "aliases":
		return groupFieldSpec{key: key, header: "Aliases", text: func(g google.GroupInfo) string { return strings.Join(g.Aliases, ",") }, value: func(g google.GroupInfo) any { return append([]string(nil), g.Aliases...) }}
	case "nonEditableAliases":
		return groupFieldSpec{key: key, header: "Non-Editable Aliases", text: func(g google.GroupInfo) string { return strings.Join(g.NonEditableAliases, ",") }, value: func(g google.GroupInfo) any { return append([]string(nil), g.NonEditableAliases...) }}
	case "id":
		return groupFieldSpec{key: key, header: "ID", text: func(g google.GroupInfo) string { return g.ID }, value: func(g google.GroupInfo) any { return g.ID }}
	case "etag":
		return groupFieldSpec{key: key, header: "ETag", text: func(g google.GroupInfo) string { return g.Etag }, value: func(g google.GroupInfo) any { return g.Etag }}
	case "kind":
		return groupFieldSpec{key: key, header: "Kind", text: func(g google.GroupInfo) string { return g.Kind }, value: func(g google.GroupInfo) any { return g.Kind }}
	default:
		return groupFieldSpec{}
	}
}

func selectedGroupFields(value string) ([]groupFieldSpec, error) {
	if strings.TrimSpace(value) == "" {
		return defaultGroupFields(), nil
	}
	fields := []groupFieldSpec{}
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		spec := groupFieldByKey(key)
		if spec.key == "" {
			return nil, fmt.Errorf("unknown group field %q; valid fields: %s", key, joinGroupFieldKeys(allGroupFields()))
		}
		fields = append(fields, spec)
	}
	return fields, nil
}

func headersForUserFields(fields []userFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForUser(user google.UserInfo, fields []userFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(user))
	}
	return row
}

func buildUserRows(users []google.UserInfo, fields []userFieldSpec) [][]string {
	rows := [][]string{headersForUserFields(fields)}
	for _, user := range users {
		rows = append(rows, rowForUser(user, fields))
	}
	return rows
}

func headersForMemberFields(fields []memberFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForMember(member google.MemberInfo, fields []memberFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(member))
	}
	return row
}

func buildMemberRows(members []google.MemberInfo, fields []memberFieldSpec) [][]string {
	rows := [][]string{headersForMemberFields(fields)}
	for _, member := range members {
		rows = append(rows, rowForMember(member, fields))
	}
	return rows
}

func headersForGroupFields(fields []groupFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForGroup(group google.GroupInfo, fields []groupFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(group))
	}
	return row
}

func buildGroupRows(groups []google.GroupInfo, fields []groupFieldSpec) [][]string {
	rows := [][]string{headersForGroupFields(fields)}
	for _, group := range groups {
		rows = append(rows, rowForGroup(group, fields))
	}
	return rows
}

func headersForOrgUnitFields(fields []orgUnitFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForOrgUnit(ou google.OrgUnitInfo, fields []orgUnitFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(ou))
	}
	return row
}

func buildOrgUnitRows(ous []google.OrgUnitInfo, fields []orgUnitFieldSpec) [][]string {
	rows := [][]string{headersForOrgUnitFields(fields)}
	for _, ou := range ous {
		rows = append(rows, rowForOrgUnit(ou, fields))
	}
	return rows
}

func headersForAliasFields(fields []aliasFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForAlias(alias google.AliasInfo, fields []aliasFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(alias))
	}
	return row
}

func buildAliasRows(aliases []google.AliasInfo, fields []aliasFieldSpec) [][]string {
	rows := [][]string{headersForAliasFields(fields)}
	for _, alias := range aliases {
		rows = append(rows, rowForAlias(alias, fields))
	}
	return rows
}

func headersForDomainFields(fields []domainFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForDomain(domain google.WorkspaceDomainInfo, fields []domainFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(domain))
	}
	return row
}

func buildDomainRows(domains []google.WorkspaceDomainInfo, fields []domainFieldSpec) [][]string {
	rows := [][]string{headersForDomainFields(fields)}
	for _, domain := range domains {
		rows = append(rows, rowForDomain(domain, fields))
	}
	return rows
}

func headersForDomainAliasFields(fields []domainAliasFieldSpec) []string {
	headers := make([]string, 0, len(fields))
	for _, field := range fields {
		headers = append(headers, field.header)
	}
	return headers
}

func rowForDomainAlias(alias google.DomainAliasInfo, fields []domainAliasFieldSpec) []string {
	row := make([]string, 0, len(fields))
	for _, field := range fields {
		row = append(row, field.text(alias))
	}
	return row
}

func buildDomainAliasRows(aliases []google.DomainAliasInfo, fields []domainAliasFieldSpec) [][]string {
	rows := [][]string{headersForDomainAliasFields(fields)}
	for _, alias := range aliases {
		rows = append(rows, rowForDomainAlias(alias, fields))
	}
	return rows
}

func joinUserFieldKeys(fields []userFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func joinGroupFieldKeys(fields []groupFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func joinMemberFieldKeys(fields []memberFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func joinOrgUnitFieldKeys(fields []orgUnitFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func joinAliasFieldKeys(fields []aliasFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func joinDomainFieldKeys(fields []domainFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func joinDomainAliasFieldKeys(fields []domainAliasFieldSpec) string {
	keys := make([]string, 0, len(fields))
	for _, field := range fields {
		keys = append(keys, field.key)
	}
	return strings.Join(keys, ", ")
}

func (r Runner) exportRowsToSheet(ctx context.Context, profile config.Profile, title string, rows [][]string) error {
	if r.Sheets == nil {
		return errors.New("Google Sheets export is not configured")
	}
	info, err := r.Sheets.CreateSheet(ctx, profile, title, rows)
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	w.Printf("Sheet created: %s\n", info.Title)
	w.Printf("URL: %s\n", info.SpreadsheetURL)
	return nil
}

func normalizeOrgUnitPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func requireConfirm(flags map[string]string, command string) error {
	if flags["confirm"] == "true" {
		return nil
	}
	return fmt.Errorf("%s requires --confirm", command)
}

func (r Runner) validateGoogle(ctx context.Context) (google.DomainInfo, error) {
	profile, err := r.activeProfile()
	if err != nil {
		return google.DomainInfo{}, err
	}
	return r.Directory.DomainInfo(ctx, profile)
}

func (r Runner) activeProfile() (config.Profile, error) {
	cfg, _, err := r.loadConfig()
	if err != nil {
		return config.Profile{}, err
	}
	profile, ok := cfg.Active()
	if !ok {
		return config.Profile{}, errors.New("gws is not configured yet; run `gws setup` first")
	}
	if profile.Domain == "" || profile.AdminSubject == "" || profile.CredentialsFile == "" {
		return config.Profile{}, errors.New("the active profile is incomplete; run `gws setup` to fill in domain, admin subject, and credentials")
	}
	if missing := auth.MissingRequiredScopes(profile.Scopes); len(missing) > 0 {
		return config.Profile{}, fmt.Errorf("the active profile is missing required scopes:\n  %s\n\nRebuild gws if needed, then rerun `gws setup` so the config and Google authorization include the current scope list.", strings.Join(missing, "\n  "))
	}
	credInfo, err := auth.ValidateCredentialsFile(profile.CredentialsFile)
	if err != nil {
		return config.Profile{}, fmt.Errorf("credentials are not ready: %w", err)
	}
	method := profile.AuthMethod
	if method == "" {
		method = auth.AuthMethod(credInfo)
	}
	if method != auth.MethodServiceAccount && !auth.TokenExists(profile.TokenFile) {
		return config.Profile{}, fmt.Errorf("OAuth token is missing: %s\n\nSetup created the profile and validated the credentials file, but gws still needs an OAuth token before it can call Google APIs.", profile.TokenFile)
	}
	return profile, nil
}

func explainValidationFailure(profile config.Profile, err error) error {
	if hint := explainGoogleAPIFailure(profile, "check connection", err, googleAPIAdminSDK); hint != "" {
		return fmt.Errorf("Admin SDK validation failed: %w\n\n%s", err, hint)
	}
	if profile.AuthMethod == auth.MethodServiceAccount {
		return fmt.Errorf("Admin SDK validation failed: %w\n\nFor service accounts, confirm that domain-wide delegation is enabled, the service account client ID is authorized in the Google Admin console, and this scope grant is authorized:\n  %s", err, strings.Join(profile.Scopes, "\n  "))
	}
	return fmt.Errorf("Admin SDK validation failed: %w\n\nFor OAuth, confirm that the Admin SDK API is enabled, the signed-in user is a Workspace admin, and the OAuth consent/token includes these scopes:\n  %s", err, strings.Join(profile.Scopes, "\n  "))
}

func explainAPICommandFailure(profile config.Profile, command string, err error) error {
	if hint := explainGoogleAPIFailure(profile, command, err, googleAPIAdminSDK); hint != "" {
		return fmt.Errorf("%s failed: %w\n\n%s", command, err, hint)
	}
	return fmt.Errorf("%s failed: %w\n\nRun `gws auth status` to inspect the active profile. If scopes changed recently, rerun `gws setup` so Google grants the current scope list:\n  %s", command, err, strings.Join(profile.Scopes, "\n  "))
}

func explainSheetExportFailure(profile config.Profile, command string, err error) error {
	if hint := explainGoogleAPIFailure(profile, command, err, googleAPISheets); hint != "" {
		return fmt.Errorf("%s sheet export failed: %w\n\n%s", command, err, hint)
	}
	return fmt.Errorf("%s sheet export failed: %w\n\nGoogle Sheets export requires the profile to include this scope and for the token to be reauthorized if scopes changed:\n  https://www.googleapis.com/auth/spreadsheets\n\nRun `gws auth status` to inspect the active profile, then rerun `gws setup` if needed.\n\nCurrent profile scopes:\n  %s", command, err, strings.Join(profile.Scopes, "\n  "))
}

func explainGmailCommandFailure(profile config.Profile, command string, err error) error {
	if hint := explainGoogleAPIFailure(profile, command, err, googleAPIGmail); hint != "" {
		return fmt.Errorf("%s failed: %w\n\n%s", command, err, hint)
	}
	return fmt.Errorf("%s failed: %w\n\nGmail delegation commands require the Gmail API to be enabled and a service account with domain-wide delegation. Run `gws auth status` to inspect the active profile.", command, err)
}

type googleAPIProduct string

const (
	googleAPIAdminSDK googleAPIProduct = "Admin SDK API"
	googleAPIGmail    googleAPIProduct = "Gmail API"
	googleAPISheets   googleAPIProduct = "Google Sheets API"
)

func explainGoogleAPIFailure(profile config.Profile, command string, err error, apiProduct googleAPIProduct) string {
	text := apiErrorText(err)
	switch {
	case isScopeFailure(text):
		return fmt.Sprintf("Google rejected the request because the active token or delegation grant does not have enough scopes.\n\nRun `gws auth status` to inspect the active profile, then rerun `gws setup` so Google grants the current scopes:\n  %s", strings.Join(profile.Scopes, "\n  "))
	case isAPIDisabledFailure(text):
		return fmt.Sprintf("The required Google API is not enabled for this project.\n\nEnable `%s` in Google Cloud Console, then retry the command.", apiProduct)
	case isDelegationFailure(text):
		if profile.AuthMethod == auth.MethodServiceAccount {
			return fmt.Sprintf("Google rejected the service account delegation.\n\nConfirm that domain-wide delegation is enabled, the service account client ID is authorized in the Google Admin console, and the admin subject `%s` is a real Workspace admin.", profile.AdminSubject)
		}
		return "Google rejected the current OAuth credentials.\n\nConfirm that the signed-in account is a Workspace admin, then rerun `gws setup` to refresh the token."
	case isInvalidInputFailure(text):
		if strings.Contains(command, "print users") || strings.Contains(command, "print groups") {
			return "Google rejected the request as invalid.\n\nCheck the query, sort, order, domain, and org-unit flags for this command."
		}
		if strings.Contains(command, "update user") || strings.Contains(command, "create user") {
			return "Google rejected the user payload as invalid.\n\nCheck the flag values you passed. Structured profile flags such as `--phones-json` and `--organizations-json` must be valid Admin SDK JSON."
		}
		if strings.Contains(command, "update group") || strings.Contains(command, "create group") {
			return "Google rejected the group payload as invalid.\n\nCheck the group email, name, and description values."
		}
		return "Google rejected the request as invalid.\n\nReview the command arguments and try again."
	default:
		return ""
	}
}

func apiErrorText(err error) string {
	parts := []string{strings.ToLower(err.Error())}
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		if apiErr.Message != "" {
			parts = append(parts, strings.ToLower(apiErr.Message))
		}
		if apiErr.Body != "" {
			parts = append(parts, strings.ToLower(apiErr.Body))
		}
		for _, item := range apiErr.Errors {
			if item.Reason != "" {
				parts = append(parts, strings.ToLower(item.Reason))
			}
			if item.Message != "" {
				parts = append(parts, strings.ToLower(item.Message))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func isScopeFailure(text string) bool {
	return strings.Contains(text, "access_token_scope_insufficient") ||
		strings.Contains(text, "insufficient authentication scopes") ||
		strings.Contains(text, "insufficientpermissions") ||
		strings.Contains(text, "insufficient permission")
}

func isAPIDisabledFailure(text string) bool {
	return strings.Contains(text, "service_disabled") ||
		strings.Contains(text, "accessnotconfigured") ||
		strings.Contains(text, "api has not been used in project") ||
		strings.Contains(text, "it is disabled")
}

func isDelegationFailure(text string) bool {
	return strings.Contains(text, "unauthorized_client") ||
		strings.Contains(text, "invalid_grant") ||
		strings.Contains(text, "not authorized to access this resource/api") ||
		strings.Contains(text, "admin subject is required for service account domain-wide delegation") ||
		strings.Contains(text, "delegation denied")
}

func isInvalidInputFailure(text string) bool {
	return strings.Contains(text, "invalid input") ||
		strings.Contains(text, "invalidinput") ||
		strings.Contains(text, "invalid argument") ||
		strings.Contains(text, "invalid_value") ||
		strings.Contains(text, "badrequest") ||
		strings.Contains(text, "invalid") && strings.Contains(text, "query")
}

func limitFlag(value string, fallback int64) (int64, error) {
	if value == "" {
		return fallback, nil
	}
	limit, err := strconv.ParseInt(value, 10, 64)
	if err != nil || limit <= 0 {
		return 0, fmt.Errorf("--limit must be a positive integer")
	}
	return limit, nil
}

func listLimitFlag(value string, fallback int64) (int64, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, false, nil
	}
	if strings.EqualFold(value, "all") {
		return 0, true, nil
	}
	limit, err := limitFlag(value, fallback)
	if err != nil {
		return 0, false, fmt.Errorf("--limit must be a positive integer or 'all'")
	}
	return limit, false, nil
}

func batchWorkersFlag(value string) (int, error) {
	if strings.TrimSpace(value) == "" {
		workers := runtime.NumCPU()
		if workers < 1 {
			return 1, nil
		}
		return workers, nil
	}
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || count <= 0 {
		return 0, fmt.Errorf("--workers must be a positive integer")
	}
	return count, nil
}

func batchTimeoutFlag(value string) (time.Duration, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("--timeout must be a positive duration like 5s or 1m")
	}
	return duration, nil
}

func userListOptionsFromFlags(flags map[string]string) (google.UserListOptions, error) {
	limit, fetchAll, err := listLimitFlag(flags["limit"], 100)
	if err != nil {
		return google.UserListOptions{}, err
	}
	orderBy, err := userSortFlag(flags["sort"])
	if err != nil {
		return google.UserListOptions{}, err
	}
	sortOrder, err := sortOrderFlag(flags["order"])
	if err != nil {
		return google.UserListOptions{}, err
	}
	showDeleted, err := optionalBoolFlag(flags["show-deleted"], "--show-deleted")
	if err != nil {
		return google.UserListOptions{}, err
	}
	query := strings.TrimSpace(flags["query"])
	if orgUnit := strings.TrimSpace(flags["org-unit"]); orgUnit != "" {
		orgQuery, err := orgUnitQuery(orgUnit)
		if err != nil {
			return google.UserListOptions{}, err
		}
		query = joinSearchClauses(query, orgQuery)
	}
	return google.UserListOptions{
		Limit:       limit,
		FetchAll:    fetchAll,
		Domain:      strings.TrimSpace(flags["domain"]),
		Query:       query,
		OrderBy:     orderBy,
		SortOrder:   sortOrder,
		ShowDeleted: showDeleted,
	}, nil
}

func groupListOptionsFromFlags(flags map[string]string) (google.GroupListOptions, error) {
	limit, fetchAll, err := listLimitFlag(flags["limit"], 100)
	if err != nil {
		return google.GroupListOptions{}, err
	}
	orderBy, err := groupSortFlag(flags["sort"])
	if err != nil {
		return google.GroupListOptions{}, err
	}
	sortOrder, err := sortOrderFlag(flags["order"])
	if err != nil {
		return google.GroupListOptions{}, err
	}
	return google.GroupListOptions{
		Limit:     limit,
		FetchAll:  fetchAll,
		Domain:    strings.TrimSpace(flags["domain"]),
		Query:     strings.TrimSpace(flags["query"]),
		UserKey:   strings.TrimSpace(flags["user"]),
		OrderBy:   orderBy,
		SortOrder: sortOrder,
	}, nil
}

func userSortFlag(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	switch value {
	case "email", "familyName", "givenName":
		return value, nil
	default:
		return "", fmt.Errorf("--sort for users must be email, familyName, or givenName")
	}
}

func groupSortFlag(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if value != "email" {
		return "", fmt.Errorf("--sort for groups must be email")
	}
	return value, nil
}

func sortOrderFlag(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "", nil
	case "asc", "ascending":
		return "ASCENDING", nil
	case "desc", "descending":
		return "DESCENDING", nil
	default:
		return "", fmt.Errorf("--order must be asc or desc")
	}
}

func optionalBoolFlag(value string, name string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return false, nil
	case "true", "yes", "1":
		return true, nil
	case "false", "no", "0":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be true or false", name)
	}
}

func orgUnitQuery(path string) (string, error) {
	literal, err := queryStringLiteral(normalizeOrgUnitPath(path))
	if err != nil {
		return "", fmt.Errorf("--org-unit cannot contain both single and double quotes")
	}
	return "orgUnitPath=" + literal, nil
}

func queryStringLiteral(value string) (string, error) {
	switch {
	case strings.Contains(value, "'") && strings.Contains(value, `"`):
		return "", errors.New("query value contains both quote styles")
	case strings.Contains(value, "'"):
		return `"` + value + `"`, nil
	default:
		return "'" + value + "'", nil
	}
}

func joinSearchClauses(first string, rest ...string) string {
	clauses := []string{}
	if strings.TrimSpace(first) != "" {
		clauses = append(clauses, strings.TrimSpace(first))
	}
	for _, clause := range rest {
		if strings.TrimSpace(clause) != "" {
			clauses = append(clauses, strings.TrimSpace(clause))
		}
	}
	return strings.Join(clauses, " ")
}

func desiredMemberEmails(flags map[string]string) ([]string, error) {
	inline := strings.TrimSpace(flags["members"])
	file := strings.TrimSpace(flags["members-file"])
	if inline == "" && file == "" {
		return nil, errors.New("sync group-members requires --members or --members-file")
	}
	if inline != "" && file != "" {
		return nil, errors.New("use either --members or --members-file, not both")
	}
	raw := inline
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("read members file: %w", err)
		}
		raw = string(data)
	}
	seen := map[string]bool{}
	members := []string{}
	for _, part := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		email := normalizeEmailToken(part)
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		members = append(members, email)
	}
	sort.Strings(members)
	return members, nil
}

type groupMemberSyncSource struct {
	Emails     []string
	Targets    []memberRoleTarget
	Structured bool
}

func (r Runner) groupMemberSyncSource(ctx context.Context, profile config.Profile, flags map[string]string) (groupMemberSyncSource, error) {
	inline := strings.TrimSpace(flags["members"])
	file := strings.TrimSpace(flags["members-file"])
	csvPath := strings.TrimSpace(flags["members-csv"])
	sheetRef := strings.TrimSpace(flags["members-sheet"])
	sources := 0
	for _, value := range []string{inline, file, csvPath, sheetRef} {
		if value != "" {
			sources++
		}
	}
	if sources == 0 {
		return groupMemberSyncSource{}, errors.New("sync group-members requires --members, --members-file, --members-csv, or --members-sheet")
	}
	if sources > 1 {
		return groupMemberSyncSource{}, errors.New("use only one of --members, --members-file, --members-csv, or --members-sheet")
	}
	if csvPath != "" {
		rows, err := readCSVRows(csvPath)
		if err != nil {
			return groupMemberSyncSource{}, err
		}
		targets, err := memberRoleTargetsFromRows(rows)
		if err != nil {
			return groupMemberSyncSource{}, err
		}
		return groupMemberSyncSource{Targets: targets, Structured: true}, nil
	}
	if sheetRef != "" {
		if r.Sheets == nil {
			return groupMemberSyncSource{}, errors.New("Google Sheets input is not configured")
		}
		spreadsheetID, err := spreadsheetIDFromValue(sheetRef)
		if err != nil {
			return groupMemberSyncSource{}, err
		}
		rows, err := r.Sheets.ReadRows(ctx, profile, spreadsheetID, strings.TrimSpace(flags["sheet-range"]))
		if err != nil {
			return groupMemberSyncSource{}, err
		}
		targets, err := memberRoleTargetsFromRows(rows)
		if err != nil {
			return groupMemberSyncSource{}, err
		}
		return groupMemberSyncSource{Targets: targets, Structured: true}, nil
	}
	emails, err := desiredMemberEmails(flags)
	if err != nil {
		return groupMemberSyncSource{}, err
	}
	return groupMemberSyncSource{Emails: emails}, nil
}

func readCSVRows(path string) ([][]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read members csv: %w", err)
	}
	reader := csv.NewReader(strings.NewReader(string(data)))
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read members csv: %w", err)
	}
	return rows, nil
}

func memberRoleTargetsFromRows(rows [][]string) ([]memberRoleTarget, error) {
	if len(rows) == 0 {
		return nil, errors.New("member sync input is empty")
	}
	header := map[string]int{}
	for i, cell := range rows[0] {
		key := strings.ToLower(strings.TrimSpace(cell))
		if key != "" {
			header[key] = i
		}
	}
	emailIndex, ok := firstHeaderIndex(header, "email", "member", "user")
	if !ok {
		return nil, errors.New("member sync input must include an email column")
	}
	roleIndex, hasRole := firstHeaderIndex(header, "role")
	seen := map[string]bool{}
	targets := []memberRoleTarget{}
	for rowNum, row := range rows[1:] {
		email := normalizeEmailToken(cellAt(row, emailIndex))
		if email == "" {
			continue
		}
		role := "MEMBER"
		if hasRole {
			role = groupMemberRole(cellAt(row, roleIndex))
			if role == "" {
				return nil, fmt.Errorf("invalid role on row %d; expected OWNER, MANAGER, or MEMBER", rowNum+2)
			}
		}
		if seen[email] {
			return nil, fmt.Errorf("duplicate member %s in structured sync input", email)
		}
		seen[email] = true
		targets = append(targets, memberRoleTarget{Email: email, Role: role})
	}
	if len(targets) == 0 {
		return nil, errors.New("member sync input did not contain any members")
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Email < targets[j].Email
	})
	return targets, nil
}

func firstHeaderIndex(header map[string]int, names ...string) (int, bool) {
	for _, name := range names {
		if index, ok := header[name]; ok {
			return index, true
		}
	}
	return 0, false
}

func cellAt(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func spreadsheetIDFromValue(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("spreadsheet ID or URL is required")
	}
	if !strings.Contains(value, "://") {
		return value, nil
	}
	marker := "/spreadsheets/d/"
	index := strings.Index(value, marker)
	if index < 0 {
		return "", fmt.Errorf("could not parse spreadsheet ID from %q", value)
	}
	rest := value[index+len(marker):]
	if slash := strings.Index(rest, "/"); slash >= 0 {
		rest = rest[:slash]
	}
	if rest == "" {
		return "", fmt.Errorf("could not parse spreadsheet ID from %q", value)
	}
	return rest, nil
}

type memberRoleChange struct {
	Email    string
	FromRole string
	ToRole   string
}

type memberRoleTarget struct {
	Email string
	Role  string
}

type groupMemberSyncPlan struct {
	ToAdd     []string
	ToRemove  []string
	ToUpdate  []memberRoleChange
	Unchanged []string
}

type structuredGroupMemberSyncPlan struct {
	ToAdd     []memberRoleTarget
	ToRemove  []memberRoleTarget
	ToUpdate  []memberRoleChange
	Unchanged []memberRoleTarget
}

func planGroupMemberSync(current []google.MemberInfo, desired []string, role string, ignoreRole bool) groupMemberSyncPlan {
	currentByEmail := map[string]google.MemberInfo{}
	for _, member := range current {
		email := normalizeEmailToken(member.Email)
		if email == "" {
			continue
		}
		currentByEmail[email] = member
	}
	desiredSet := map[string]bool{}
	plan := groupMemberSyncPlan{}
	for _, email := range desired {
		desiredSet[email] = true
		member, ok := currentByEmail[email]
		if !ok {
			plan.ToAdd = append(plan.ToAdd, email)
			continue
		}
		currentRole := strings.ToUpper(strings.TrimSpace(member.Role))
		if ignoreRole || currentRole == role {
			plan.Unchanged = append(plan.Unchanged, email)
			continue
		}
		plan.ToUpdate = append(plan.ToUpdate, memberRoleChange{
			Email:    email,
			FromRole: currentRole,
			ToRole:   role,
		})
	}
	for email, member := range currentByEmail {
		if desiredSet[email] {
			continue
		}
		if ignoreRole || strings.ToUpper(strings.TrimSpace(member.Role)) == role {
			plan.ToRemove = append(plan.ToRemove, email)
		}
	}
	sort.Strings(plan.ToAdd)
	sort.Strings(plan.ToRemove)
	sort.Strings(plan.Unchanged)
	sort.Slice(plan.ToUpdate, func(i, j int) bool {
		return plan.ToUpdate[i].Email < plan.ToUpdate[j].Email
	})
	return plan
}

func planStructuredGroupMemberSync(current []google.MemberInfo, desired []memberRoleTarget) structuredGroupMemberSyncPlan {
	currentByEmail := map[string]google.MemberInfo{}
	for _, member := range current {
		email := normalizeEmailToken(member.Email)
		if email == "" {
			continue
		}
		currentByEmail[email] = member
	}
	desiredByEmail := map[string]memberRoleTarget{}
	plan := structuredGroupMemberSyncPlan{}
	for _, target := range desired {
		desiredByEmail[target.Email] = target
		member, ok := currentByEmail[target.Email]
		if !ok {
			plan.ToAdd = append(plan.ToAdd, target)
			continue
		}
		currentRole := strings.ToUpper(strings.TrimSpace(member.Role))
		if currentRole == target.Role {
			plan.Unchanged = append(plan.Unchanged, target)
			continue
		}
		plan.ToUpdate = append(plan.ToUpdate, memberRoleChange{
			Email:    target.Email,
			FromRole: currentRole,
			ToRole:   target.Role,
		})
	}
	for email, member := range currentByEmail {
		if _, ok := desiredByEmail[email]; ok {
			continue
		}
		plan.ToRemove = append(plan.ToRemove, memberRoleTarget{
			Email: email,
			Role:  strings.ToUpper(strings.TrimSpace(member.Role)),
		})
	}
	sort.Slice(plan.ToAdd, func(i, j int) bool { return plan.ToAdd[i].Email < plan.ToAdd[j].Email })
	sort.Slice(plan.ToRemove, func(i, j int) bool { return plan.ToRemove[i].Email < plan.ToRemove[j].Email })
	sort.Slice(plan.Unchanged, func(i, j int) bool { return plan.Unchanged[i].Email < plan.Unchanged[j].Email })
	sort.Slice(plan.ToUpdate, func(i, j int) bool { return plan.ToUpdate[i].Email < plan.ToUpdate[j].Email })
	return plan
}

func normalizeEmailToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func requireServiceAccountProfile(profile config.Profile, command string) error {
	if profile.AuthMethod != auth.MethodServiceAccount {
		return fmt.Errorf("%s requires a service account profile with domain-wide delegation. OAuth user tokens cannot manage Gmail delegates", command)
	}
	return nil
}

func optionalBoolPointerFlag(flags map[string]string, name string) (*bool, error) {
	value, ok := flags[name]
	if !ok {
		return nil, nil
	}
	parsed, err := optionalBoolFlag(value, "--"+name)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func optionalJSONPointerFlag(flags map[string]string, name string) (*any, error) {
	value, ok := flags[name]
	if !ok {
		return nil, nil
	}
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil, fmt.Errorf("--%s requires a JSON value", name)
	}
	var decoded any
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, fmt.Errorf("--%s must be valid JSON: %w", name, err)
	}
	return &decoded, nil
}

func compactJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	switch string(data) {
	case "null", "[]", "{}", `""`:
		return ""
	default:
		return string(data)
	}
}

func prettyJSON(value any) string {
	if value == nil {
		return ""
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	switch string(data) {
	case "null", "[]", "{}", `""`:
		return ""
	default:
		return string(data)
	}
}

func groupMemberRole(value string) string {
	if value == "" {
		return "MEMBER"
	}
	role := strings.ToUpper(strings.TrimSpace(value))
	switch role {
	case "OWNER", "MANAGER", "MEMBER":
		return role
	default:
		return ""
	}
}

func passwordFromFlags(flags map[string]string) (string, error) {
	password := strings.TrimSpace(flags["password"])
	passwordFile := strings.TrimSpace(flags["password-file"])
	if password != "" && passwordFile != "" {
		return "", errors.New("use either --password or --password-file, not both")
	}
	if passwordFile != "" {
		data, err := os.ReadFile(passwordFile)
		if err != nil {
			return "", fmt.Errorf("read password file: %w", err)
		}
		password = strings.TrimRight(string(data), "\r\n")
	}
	if password == "" {
		return "", errors.New("password is required; use --password-file PATH to avoid storing it in shell history")
	}
	return password, nil
}

func boolFlag(value string, fallback bool) (bool, error) {
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y":
		return true, nil
	case "false", "0", "no", "n":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean value %q", value)
	}
}

func (r Runner) localAuthStatus() authStatus {
	path, err := r.configPath()
	if err != nil {
		return authStatus{Message: err.Error()}
	}
	cfg, err := config.Load(path)
	if err != nil {
		return authStatus{ConfigFile: path, Message: err.Error()}
	}
	cfg = config.ApplyEnv(cfg)
	status := authStatus{
		ConfigFile:    path,
		ActiveProfile: cfg.ActiveProfile,
	}
	profile, ok := cfg.Active()
	if !ok {
		status.Message = "Run `gws setup` to create a profile."
		return status
	}
	status.Configured = true
	status.Domain = profile.Domain
	status.AdminSubject = profile.AdminSubject
	status.CredentialsFile = profile.CredentialsFile
	status.TokenFile = profile.TokenFile
	status.AuthMethod = profile.AuthMethod
	status.Scopes = profile.Scopes
	status.MissingScopes = auth.MissingRequiredScopes(profile.Scopes)
	credInfo, err := auth.ValidateCredentialsFile(profile.CredentialsFile)
	if err != nil {
		status.Message = fmt.Sprintf("Credentials are not ready: %v", err)
		return status
	}
	status.CredentialType = credInfo.Type
	if status.AuthMethod == "" {
		status.AuthMethod = auth.AuthMethod(credInfo)
	}
	if status.Domain == "" || status.AdminSubject == "" {
		status.Message = "Profile is missing domain or admin subject. Run `gws setup` again."
		return status
	}
	if len(status.MissingScopes) > 0 {
		status.Message = "Profile is missing required scopes. Rebuild gws if needed, then rerun `gws setup` to update config and authorization."
		return status
	}
	if status.AuthMethod != auth.MethodServiceAccount {
		status.TokenPresent = auth.TokenExists(profile.TokenFile)
		if !status.TokenPresent {
			status.Message = "OAuth token is missing. Run `gws setup` again to authorize this profile."
			return status
		}
	}
	status.Ready = true
	status.Message = "Local auth files are present."
	return status
}

func (r Runner) loadConfig() (config.File, string, error) {
	path, err := r.configPath()
	if err != nil {
		return config.File{}, "", err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.File{}, "", err
	}
	return config.ApplyEnv(cfg), path, nil
}

func (r Runner) configPath() (string, error) {
	if r.Config != "" {
		return r.Config, nil
	}
	if env := os.Getenv("GWS_CONFIG"); env != "" {
		return env, nil
	}
	return config.DefaultPath()
}
