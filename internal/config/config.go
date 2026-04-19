package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joncarr/gws/internal/auth"
)

const (
	DefaultProfileName = "default"
	DefaultOutput      = "text"
)

type File struct {
	ActiveProfile string             `json:"active_profile"`
	Profiles      map[string]Profile `json:"profiles"`
}

type Profile struct {
	Domain          string   `json:"domain"`
	AdminSubject    string   `json:"admin_subject"`
	CredentialsFile string   `json:"credentials_file"`
	TokenFile       string   `json:"token_file"`
	AuthMethod      string   `json:"auth_method"`
	Scopes          []string `json:"scopes"`
	Output          string   `json:"output"`
}

func (p Profile) GetCredentialsFile() string { return p.CredentialsFile }
func (p Profile) GetTokenFile() string       { return p.TokenFile }
func (p Profile) GetAdminSubject() string    { return p.AdminSubject }
func (p Profile) GetScopes() []string        { return p.Scopes }

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gws", "config.json"), nil
}

func DefaultTokenPath(profile string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gws", "tokens", profile+"-token.json"), nil
}

func Empty() File {
	return File{
		ActiveProfile: DefaultProfileName,
		Profiles:      map[string]Profile{},
	}
}

func NewProfile(profile string) (Profile, error) {
	tokenPath, err := DefaultTokenPath(profile)
	if err != nil {
		return Profile{}, err
	}
	return Profile{
		TokenFile:  tokenPath,
		AuthMethod: "oauth",
		Scopes:     append([]string(nil), auth.RequiredScopes...),
		Output:     DefaultOutput,
	}, nil
}

func Load(path string) (File, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return File{}, err
		}
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Empty(), nil
	}
	if err != nil {
		return File{}, err
	}
	var cfg File
	if err := json.Unmarshal(data, &cfg); err != nil {
		return File{}, fmt.Errorf("read config %s: %w", path, err)
	}
	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = DefaultProfileName
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return cfg, nil
}

func Save(path string, cfg File) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}
	if cfg.ActiveProfile == "" {
		cfg.ActiveProfile = DefaultProfileName
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (f File) Active() (Profile, bool) {
	profile, ok := f.Profiles[f.ActiveProfile]
	return profile, ok
}

func ApplyEnv(f File) File {
	if v := os.Getenv("GWS_PROFILE"); v != "" {
		f.ActiveProfile = v
	}
	profile, ok := f.Profiles[f.ActiveProfile]
	if !ok {
		return f
	}
	if v := os.Getenv("GWS_DOMAIN"); v != "" {
		profile.Domain = v
	}
	if v := os.Getenv("GWS_ADMIN_SUBJECT"); v != "" {
		profile.AdminSubject = v
	}
	if v := os.Getenv("GWS_CREDENTIALS_FILE"); v != "" {
		profile.CredentialsFile = v
	}
	f.Profiles[f.ActiveProfile] = profile
	return f
}
