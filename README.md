# gws

`gws` is a Go command-line tool for administering Google Workspace environments.
It is inspired by GAMADV-XTD3 command wording, but it is not a line-by-line port.
The project favors a clean, idiomatic Go foundation that can grow in small,
testable slices.

The first priority is connection and authorization. Broad command coverage comes
later.

## Current Scope

Implemented first-pass commands:

```text
gws version
gws help
gws setup
gws config show
gws check connection
gws info domain
```

`gws setup` creates a local profile, validates the Google credentials file,
guides authorization, explains the required Admin SDK scope, shows the files it
creates, and finishes with a real Admin SDK connection check.

## Build and Run

```sh
go build ./cmd/gws
./gws help
./gws setup
```

Use a custom config path during development:

```sh
gws --config ./gws-config.json setup
gws --config ./gws-config.json config show
```

## Setup and Authentication

This first slice supports:

- a Desktop OAuth client JSON from Google Cloud Console
- a service account JSON configured for Google Workspace domain-wide delegation

The Google Cloud project must have the Admin SDK API enabled.

The initial validation scope is:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
```

During setup, `gws` asks for:

- profile name
- primary Workspace domain
- Workspace admin email used for validation
- path to the OAuth client JSON or service account JSON file

For OAuth, setup prints an authorization URL, starts a temporary localhost
callback listener, saves the token file, and then validates Admin SDK access.

For service accounts, setup explains the required domain-wide delegation step,
shows the client ID to authorize in the Google Admin console, and validates by
impersonating the configured admin subject.

It then writes a config file like:

```json
{
  "active_profile": "default",
  "profiles": {
    "default": {
      "domain": "example.com",
      "admin_subject": "admin@example.com",
      "credentials_file": "/path/to/client.json",
      "token_file": "/home/user/.config/gws/tokens/default-token.json",
      "auth_method": "oauth",
      "scopes": [
        "https://www.googleapis.com/auth/admin.directory.customer.readonly"
      ],
      "output": "text"
    }
  }
}
```

OAuth tokens are stored with `0600` permissions at the configured token path.
Service account profiles do not use a token file.

After setup, run either validation command at any time:

```sh
gws check connection
gws info domain
```

## Environment Overrides

The config path and active profile can be overridden with:

```text
GWS_CONFIG
GWS_PROFILE
GWS_DOMAIN
GWS_ADMIN_SUBJECT
GWS_CREDENTIALS_FILE
```

## Not Implemented Yet

- users, groups, and org-unit mutation commands
- GAM grammar compatibility beyond the small first command set
- batch and CSV execution
