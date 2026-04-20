package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	goauth "golang.org/x/oauth2/google"
)

var RequiredScopes = []string{
	"https://www.googleapis.com/auth/admin.directory.customer.readonly",
	"https://www.googleapis.com/auth/admin.directory.group",
	"https://www.googleapis.com/auth/admin.directory.group.member",
	"https://www.googleapis.com/auth/admin.directory.orgunit",
	"https://www.googleapis.com/auth/admin.directory.user",
}

const (
	MethodOAuth          = "oauth"
	MethodServiceAccount = "service_account"
)

type CredentialInfo struct {
	Type     string
	ClientID string
}

type Profile interface {
	GetCredentialsFile() string
	GetTokenFile() string
	GetAdminSubject() string
	GetScopes() []string
}

func ValidateCredentialsFile(path string) (CredentialInfo, error) {
	if path == "" {
		return CredentialInfo{}, errors.New("no credentials file is configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return CredentialInfo{}, fmt.Errorf("read credentials file: %w", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return CredentialInfo{}, fmt.Errorf("credentials file is not valid JSON: %w", err)
	}
	if info, ok := parseOAuthClient(raw, "installed"); ok {
		return info, nil
	}
	if info, ok := parseOAuthClient(raw, "web"); ok {
		return info, nil
	}
	if info, ok := parseServiceAccount(raw); ok {
		return info, nil
	}
	return CredentialInfo{}, errors.New("credentials file is not a supported OAuth client or service account JSON file")
}

func IsOAuth(info CredentialInfo) bool {
	return info.Type == "oauth_installed" || info.Type == "oauth_web"
}

func AuthMethod(info CredentialInfo) string {
	if info.Type == MethodServiceAccount {
		return MethodServiceAccount
	}
	return MethodOAuth
}

func MissingRequiredScopes(scopes []string) []string {
	have := map[string]bool{}
	for _, scope := range scopes {
		have[scope] = true
	}
	var missing []string
	for _, scope := range RequiredScopes {
		if !have[scope] {
			missing = append(missing, scope)
		}
	}
	return missing
}

func parseOAuthClient(raw map[string]json.RawMessage, key string) (CredentialInfo, bool) {
	var body struct {
		ClientID string `json:"client_id"`
	}
	if data, ok := raw[key]; ok && json.Unmarshal(data, &body) == nil && body.ClientID != "" {
		return CredentialInfo{Type: "oauth_" + key, ClientID: body.ClientID}, true
	}
	return CredentialInfo{}, false
}

func parseServiceAccount(raw map[string]json.RawMessage) (CredentialInfo, bool) {
	var typ string
	if data, ok := raw["type"]; !ok || json.Unmarshal(data, &typ) != nil || typ != "service_account" {
		return CredentialInfo{}, false
	}
	var clientID string
	if data, ok := raw["client_id"]; ok {
		_ = json.Unmarshal(data, &clientID)
	}
	var clientEmail string
	if data, ok := raw["client_email"]; ok {
		_ = json.Unmarshal(data, &clientEmail)
	}
	if clientID == "" {
		clientID = clientEmail
	}
	return CredentialInfo{Type: "service_account", ClientID: clientID}, true
}

func TokenExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func HTTPClient(ctx context.Context, profile Profile) (*http.Client, error) {
	credentials, err := os.ReadFile(profile.GetCredentialsFile())
	if err != nil {
		return nil, fmt.Errorf("read credentials file: %w", err)
	}
	info, err := ValidateCredentialsFile(profile.GetCredentialsFile())
	if err != nil {
		return nil, err
	}
	if info.Type == MethodServiceAccount {
		conf, err := goauth.JWTConfigFromJSON(credentials, profile.GetScopes()...)
		if err != nil {
			return nil, fmt.Errorf("parse service account credentials: %w", err)
		}
		if profile.GetAdminSubject() == "" {
			return nil, errors.New("admin subject is required for service account domain-wide delegation")
		}
		conf.Subject = profile.GetAdminSubject()
		return conf.Client(ctx), nil
	}
	conf, err := goauth.ConfigFromJSON(credentials, profile.GetScopes()...)
	if err != nil {
		return nil, fmt.Errorf("parse OAuth client credentials: %w", err)
	}
	token, err := ReadToken(profile.GetTokenFile())
	if err != nil {
		return nil, err
	}
	return conf.Client(ctx, token), nil
}

func RunOAuthLocalFlow(ctx context.Context, credentialsPath string, scopes []string, tokenPath string, out io.Writer) error {
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		return fmt.Errorf("read OAuth credentials file: %w", err)
	}
	conf, err := goauth.ConfigFromJSON(credentials, scopes...)
	if err != nil {
		return fmt.Errorf("parse OAuth client credentials: %w", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start local OAuth callback listener: %w", err)
	}
	defer ln.Close()

	state, err := randomState()
	if err != nil {
		return err
	}
	conf.RedirectURL = "http://" + ln.Addr().String() + "/oauth2callback"

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
	}
	server.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/oauth2callback" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("state") != state {
			http.Error(w, "OAuth state did not match. Return to your terminal and rerun gws setup.", http.StatusBadRequest)
			errCh <- errors.New("OAuth callback state did not match; rerun setup and use the newest authorization URL")
			return
		}
		if msg := r.URL.Query().Get("error"); msg != "" {
			http.Error(w, "Google returned an OAuth error. Return to your terminal for details.", http.StatusBadRequest)
			errCh <- fmt.Errorf("Google OAuth returned %q; check that the OAuth client and requested scopes are allowed", msg)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No authorization code was returned. Return to your terminal and rerun gws setup.", http.StatusBadRequest)
			errCh <- errors.New("Google did not return an authorization code")
			return
		}
		fmt.Fprintln(w, "gws authorization complete. You can close this browser tab and return to your terminal.")
		codeCh <- code
	})

	go func() {
		if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	defer server.Shutdown(context.Background())

	authURL := conf.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Fprintf(out, "Open this URL in a browser and sign in as a Workspace admin:\n\n%s\n\n", authURL)
	fmt.Fprintf(out, "Waiting for Google to redirect back to %s ...\n", conf.RedirectURL)

	var code string
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	case code = <-codeCh:
	}

	token, err := conf.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange OAuth authorization code: %w", err)
	}
	if err := SaveToken(tokenPath, token); err != nil {
		return err
	}
	return nil
}

func ReadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read OAuth token file: %w", err)
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse OAuth token file: %w", err)
	}
	if !token.Valid() && token.RefreshToken == "" {
		return nil, errors.New("OAuth token file does not contain a usable access or refresh token")
	}
	return &token, nil
}

func SaveToken(path string, token *oauth2.Token) error {
	if path == "" {
		return errors.New("no OAuth token file path is configured")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create token directory: %w", err)
	}
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write OAuth token file: %w", err)
	}
	return nil
}

func randomState() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("create OAuth state: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}
