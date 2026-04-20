package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/joncarr/gws/internal/auth"
	"github.com/joncarr/gws/internal/cli"
	"github.com/joncarr/gws/internal/config"
	"github.com/joncarr/gws/internal/google"
	"github.com/joncarr/gws/internal/output"
)

const Version = "0.1.0-dev"

type Runner struct {
	Stdin     io.Reader
	Stdout    io.Writer
	Stderr    io.Writer
	Config    string
	Directory google.DirectoryClient
}

func (r Runner) Run(ctx context.Context, parsed cli.Parsed) error {
	switch commandKey(parsed.Positionals) {
	case "", "help":
		r.help()
		return nil
	case "version":
		output.New(r.Stdout).Printf("gws %s\n", Version)
		return nil
	case "setup":
		return r.setup(ctx, parsed.Flags)
	case "add group-member":
		return r.addGroupMember(ctx, parsed.Positionals, parsed.Flags)
	case "create group":
		return r.createGroup(ctx, parsed.Positionals, parsed.Flags)
	case "create user":
		return r.createUser(ctx, parsed.Positionals, parsed.Flags)
	case "remove group-member":
		return r.removeGroupMember(ctx, parsed.Positionals, parsed.Flags)
	case "suspend user":
		return r.setUserSuspended(ctx, parsed.Positionals, parsed.Flags, true)
	case "unsuspend user":
		return r.setUserSuspended(ctx, parsed.Positionals, parsed.Flags, false)
	case "update group":
		return r.updateGroup(ctx, parsed.Positionals, parsed.Flags)
	case "update user":
		return r.updateUser(ctx, parsed.Positionals, parsed.Flags)
	case "config show":
		return r.configShow(parsed.Flags)
	case "auth status":
		return r.authStatus(ctx, parsed.Flags)
	case "check connection":
		return r.checkConnection(ctx, parsed.Flags)
	case "info domain":
		return r.infoDomain(ctx, parsed.Flags)
	case "info group":
		return r.infoGroup(ctx, parsed.Positionals, parsed.Flags)
	case "info ou":
		return r.infoOrgUnit(ctx, parsed.Positionals, parsed.Flags)
	case "info user":
		return r.infoUser(ctx, parsed.Positionals, parsed.Flags)
	case "print groups":
		return r.printGroups(ctx, parsed.Flags)
	case "print group-members":
		return r.printGroupMembers(ctx, parsed.Positionals, parsed.Flags)
	case "print ous":
		return r.printOrgUnits(ctx, parsed.Flags)
	case "print users":
		return r.printUsers(ctx, parsed.Flags)
	default:
		return fmt.Errorf("unknown command %q\n\nRun `gws help` to see supported commands.", strings.Join(parsed.Positionals, " "))
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
	w.Println("  gws version")
	w.Println("  gws help")
	w.Println("  gws setup [--profile default] [--domain example.com] [--admin admin@example.com] [--credentials client.json]")
	w.Println("  gws config show [--json]")
	w.Println("  gws auth status [--json]")
	w.Println("  gws check connection")
	w.Println("  gws info domain")
	w.Println("  gws info group group@example.com [--json]")
	w.Println("  gws info ou /Engineering [--json]")
	w.Println("  gws info user user@example.com [--json]")
	w.Println("  gws print groups [--limit 100] [--json]")
	w.Println("  gws print group-members group@example.com [--limit 100] [--json]")
	w.Println("  gws print ous [--json]")
	w.Println("  gws print users [--limit 100] [--json]")
	w.Println("  gws add group-member group@example.com user@example.com [--role MEMBER] [--json]")
	w.Println("  gws create group group@example.com --name NAME [--description TEXT] [--json]")
	w.Println("  gws create user user@example.com --given-name NAME --family-name NAME --password-file PATH [--org-unit /PATH] [--json]")
	w.Println("  gws remove group-member group@example.com user@example.com")
	w.Println("  gws suspend user user@example.com [--json]")
	w.Println("  gws unsuspend user user@example.com [--json]")
	w.Println("  gws update group group@example.com [--name NAME] [--description TEXT] [--json]")
	w.Println("  gws update user user@example.com [--given-name NAME] [--family-name NAME] [--org-unit /PATH] [--json]")
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
	w.Println("You can use either a Desktop OAuth client JSON or a service account JSON configured for domain-wide delegation.")
	w.Println("")
	w.Println("Required Admin SDK scope for this first validation slice:")
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
	w.Printf("Customer ID: %s\n", info.CustomerID)
	w.Printf("Primary domain: %s\n", info.PrimaryDomain)
	return nil
}

func (r Runner) infoDomain(ctx context.Context, flags map[string]string) error {
	info, err := r.validateGoogle(ctx)
	if err != nil {
		return err
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(info)
	}
	w.Printf("Customer ID: %s\n", info.CustomerID)
	w.Printf("Primary domain: %s\n", info.PrimaryDomain)
	if info.VerifiedDomainName != "" {
		w.Printf("Verified domain: %s\n", info.VerifiedDomainName)
	}
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
	w.Printf("Primary email: %s\n", user.PrimaryEmail)
	if user.Name != "" {
		w.Printf("Name: %s\n", user.Name)
	}
	w.Printf("Suspended: %t\n", user.Suspended)
	w.Printf("Org unit: %s\n", user.OrgUnitPath)
	w.Printf("Admin: %t\n", user.IsAdmin)
	w.Printf("Delegated admin: %t\n", user.IsDelegatedAdmin)
	w.Printf("Directory visible: %t\n", user.IncludeInGlobalAddressList)
	if user.CreationTime != "" {
		w.Printf("Created: %s\n", user.CreationTime)
	}
	if user.LastLoginTime != "" {
		w.Printf("Last login: %s\n", user.LastLoginTime)
	}
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
	w.Printf("Email: %s\n", group.Email)
	if group.Name != "" {
		w.Printf("Name: %s\n", group.Name)
	}
	if group.Description != "" {
		w.Printf("Description: %s\n", group.Description)
	}
	w.Printf("Direct members: %d\n", group.DirectMembersCount)
	w.Printf("Admin created: %t\n", group.AdminCreated)
	if len(group.Aliases) > 0 {
		w.Printf("Aliases: %s\n", strings.Join(group.Aliases, ", "))
	}
	if len(group.NonEditableAliases) > 0 {
		w.Printf("Non-editable aliases: %s\n", strings.Join(group.NonEditableAliases, ", "))
	}
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
		return errors.New("group email is required\n\nUsage: gws update group group@example.com [--name NAME] [--description TEXT]")
	}
	name, hasName := flags["name"]
	description, hasDescription := flags["description"]
	if !hasName && !hasDescription {
		return errors.New("nothing to update; provide --name or --description")
	}
	email := strings.TrimSpace(args[2])
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	group, err := r.Directory.UpdateGroup(ctx, profile, email, google.GroupInfo{
		Name:        strings.TrimSpace(name),
		Description: strings.TrimSpace(description),
	})
	if err != nil {
		return explainAPICommandFailure(profile, "update group", err)
	}
	return printGroupResult(r.Stdout, group, flags["json"] == "true", "Group updated")
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
	w.Printf("Path: %s\n", ou.OrgUnitPath)
	w.Printf("Name: %s\n", ou.Name)
	if ou.Description != "" {
		w.Printf("Description: %s\n", ou.Description)
	}
	if ou.ParentOrgUnitPath != "" {
		w.Printf("Parent path: %s\n", ou.ParentOrgUnitPath)
	}
	if ou.OrgUnitID != "" {
		w.Printf("ID: %s\n", ou.OrgUnitID)
	}
	return nil
}

func (r Runner) printGroups(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	limit, err := limitFlag(flags["limit"], 100)
	if err != nil {
		return err
	}
	groups, err := r.Directory.Groups(ctx, profile, limit)
	if err != nil {
		return explainAPICommandFailure(profile, "print groups", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(groups)
	}
	w.Println("Email\tName\tDirect Members\tAdmin Created")
	for _, group := range groups {
		w.Printf("%s\t%s\t%d\t%t\n", group.Email, group.Name, group.DirectMembersCount, group.AdminCreated)
	}
	return nil
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
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(members)
	}
	w.Println("Email\tRole\tType\tStatus")
	for _, member := range members {
		w.Printf("%s\t%s\t%s\t%s\n", member.Email, member.Role, member.Type, member.Status)
	}
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
	w.Printf("Role: %s\n", member.Role)
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

func printGroupResult(out io.Writer, group google.GroupInfo, asJSON bool, label string) error {
	w := output.New(out)
	if asJSON {
		return w.JSON(group)
	}
	w.Printf("%s: %s\n", label, group.Email)
	if group.Name != "" {
		w.Printf("Name: %s\n", group.Name)
	}
	if group.Description != "" {
		w.Printf("Description: %s\n", group.Description)
	}
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
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(ous)
	}
	w.Println("Path\tName\tParent Path\tDescription")
	for _, ou := range ous {
		w.Printf("%s\t%s\t%s\t%s\n", ou.OrgUnitPath, ou.Name, ou.ParentOrgUnitPath, ou.Description)
	}
	return nil
}

func (r Runner) printUsers(ctx context.Context, flags map[string]string) error {
	profile, err := r.activeProfile()
	if err != nil {
		return err
	}
	limit, err := limitFlag(flags["limit"], 100)
	if err != nil {
		return err
	}
	users, err := r.Directory.Users(ctx, profile, limit)
	if err != nil {
		return explainAPICommandFailure(profile, "print users", err)
	}
	w := output.New(r.Stdout)
	if flags["json"] == "true" {
		return w.JSON(users)
	}
	w.Println("Primary Email\tName\tSuspended\tOrg Unit")
	for _, user := range users {
		w.Printf("%s\t%s\t%t\t%s\n", user.PrimaryEmail, user.Name, user.Suspended, user.OrgUnitPath)
	}
	return nil
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

func (r Runner) updateUser(ctx context.Context, args []string, flags map[string]string) error {
	if len(args) < 3 || strings.TrimSpace(args[2]) == "" {
		return errors.New("user email is required\n\nUsage: gws update user user@example.com [--given-name NAME] [--family-name NAME] [--org-unit /PATH]")
	}
	givenName, hasGivenName := flags["given-name"]
	familyName, hasFamilyName := flags["family-name"]
	orgUnit, hasOrgUnit := flags["org-unit"]
	if !hasGivenName && !hasFamilyName && !hasOrgUnit {
		return errors.New("nothing to update; provide --given-name, --family-name, or --org-unit")
	}
	update := google.UserUpdate{
		GivenName:  strings.TrimSpace(givenName),
		FamilyName: strings.TrimSpace(familyName),
	}
	if hasOrgUnit {
		update.OrgUnitPath = normalizeOrgUnitPath(orgUnit)
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
	if user.Name != "" {
		w.Printf("Name: %s\n", user.Name)
	}
	w.Printf("Org unit: %s\n", user.OrgUnitPath)
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
	if profile.AuthMethod == auth.MethodServiceAccount {
		return fmt.Errorf("Admin SDK validation failed: %w\n\nFor service accounts, confirm that domain-wide delegation is enabled, the service account client ID is authorized in the Google Admin console, and this scope is included:\n  %s", err, strings.Join(profile.Scopes, "\n  "))
	}
	return fmt.Errorf("Admin SDK validation failed: %w\n\nFor OAuth, confirm that the Admin SDK API is enabled, the signed-in user is a Workspace admin, and the OAuth consent/token includes this scope:\n  %s", err, strings.Join(profile.Scopes, "\n  "))
}

func explainAPICommandFailure(profile config.Profile, command string, err error) error {
	return fmt.Errorf("%s failed: %w\n\nRun `gws auth status` to inspect the active profile. If scopes changed recently, rerun `gws setup` so Google grants the current scope list:\n  %s", command, err, strings.Join(profile.Scopes, "\n  "))
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
