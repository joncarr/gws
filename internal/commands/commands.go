package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
	case "config show":
		return r.configShow(parsed.Flags)
	case "check connection":
		return r.checkConnection(ctx, parsed.Flags)
	case "info domain":
		return r.infoDomain(ctx, parsed.Flags)
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
	w.Println("  gws check connection")
	w.Println("  gws info domain")
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
	w.Println("This setup connects gws to Google Workspace in small, visible steps.")
	w.Println("You need a Google Cloud project with the Admin SDK API enabled.")
	w.Println("You can use either a Desktop OAuth client JSON or a service account JSON configured for domain-wide delegation.")
	w.Println("")
	w.Println("Required Admin SDK scope for this first validation slice:")
	for _, scope := range auth.RequiredScopes {
		w.Printf("  %s\n", scope)
	}
	w.Println("")
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

func (r Runner) validateGoogle(ctx context.Context) (google.DomainInfo, error) {
	cfg, _, err := r.loadConfig()
	if err != nil {
		return google.DomainInfo{}, err
	}
	profile, ok := cfg.Active()
	if !ok {
		return google.DomainInfo{}, errors.New("gws is not configured yet; run `gws setup` first")
	}
	if profile.Domain == "" || profile.AdminSubject == "" || profile.CredentialsFile == "" {
		return google.DomainInfo{}, errors.New("the active profile is incomplete; run `gws setup` to fill in domain, admin subject, and credentials")
	}
	credInfo, err := auth.ValidateCredentialsFile(profile.CredentialsFile)
	if err != nil {
		return google.DomainInfo{}, fmt.Errorf("credentials are not ready: %w", err)
	}
	method := profile.AuthMethod
	if method == "" {
		method = auth.AuthMethod(credInfo)
	}
	if method != auth.MethodServiceAccount && !auth.TokenExists(profile.TokenFile) {
		return google.DomainInfo{}, fmt.Errorf("OAuth token is missing: %s\n\nSetup created the profile and validated the credentials file, but gws still needs an OAuth token before it can call Google APIs.", profile.TokenFile)
	}
	return r.Directory.DomainInfo(ctx, profile)
}

func explainValidationFailure(profile config.Profile, err error) error {
	if profile.AuthMethod == auth.MethodServiceAccount {
		return fmt.Errorf("Admin SDK validation failed: %w\n\nFor service accounts, confirm that domain-wide delegation is enabled, the service account client ID is authorized in the Google Admin console, and this scope is included:\n  %s", err, strings.Join(profile.Scopes, "\n  "))
	}
	return fmt.Errorf("Admin SDK validation failed: %w\n\nFor OAuth, confirm that the Admin SDK API is enabled, the signed-in user is a Workspace admin, and the OAuth consent/token includes this scope:\n  %s", err, strings.Join(profile.Scopes, "\n  "))
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
