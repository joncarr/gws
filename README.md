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
gws auth status
gws check connection
gws info domain
gws info group group@example.com
gws info ou /Engineering
gws info user user@example.com
gws print group-members group@example.com
gws print groups
gws print ous
gws print users
gws add group-member group@example.com user@example.com
gws create group group@example.com
gws create user user@example.com
gws create ou
gws remove group-member group@example.com user@example.com
gws suspend user user@example.com
gws unsuspend user user@example.com
gws update group group@example.com
gws update ou /Engineering
gws update user user@example.com
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

Setup asks for the Admin SDK scopes needed for the planned initial
users/groups/org-unit admin surface:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
https://www.googleapis.com/auth/admin.directory.group
https://www.googleapis.com/auth/admin.directory.group.member
https://www.googleapis.com/auth/admin.directory.orgunit
https://www.googleapis.com/auth/admin.directory.user
```

These cover the current readonly validation/list/info commands and the next
planned users, groups, group membership, and org-unit mutation commands. Deferred
areas such as Gmail, Drive, Calendar, Reports, Vault, Chrome, and device
management are intentionally not requested yet.

### Recommended First-Time Setup

The easiest way to test `gws` today is with a **Desktop OAuth client**.

Google Workspace and Google Cloud are separate admin surfaces:

- **Google Workspace Admin Console** is where your Workspace domain, users,
  groups, and admin permissions live.
- **Google Cloud Console** is where you create API credentials so a tool like
  `gws` can call Google Workspace APIs.

For this first setup, you create credentials in Google Cloud Console, then sign
in with a Google Workspace admin account during OAuth authorization.

1. Open Google Cloud Console.
2. Create or choose a Google Cloud project.
3. Go to **APIs & Services > Library**.
4. Enable **Admin SDK API**.
5. Go to **APIs & Services > OAuth consent screen** and complete the required
   consent screen setup for your organization.
6. Go to **APIs & Services > Credentials**.
7. Select **Create credentials > OAuth client ID**.
8. Choose **Desktop app** as the application type.
9. Download the JSON credentials file.

When `gws setup` asks:

```text
Path to OAuth client or service account JSON:
```

enter the path to that downloaded file, for example:

```text
/home/username/Downloads/client_secret_1234567890.apps.googleusercontent.com.json
```

Then run setup:

```sh
go build -o gws ./cmd/gws
./gws setup
```

For development, use a local config file so you do not overwrite your normal
profile:

```sh
./gws --config ./dev-gws-config.json setup
./gws --config ./dev-gws-config.json check connection
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

### Service Account Setup

Use a service account only if you specifically want domain-wide delegation.

1. In Google Cloud Console, create a service account.
2. Enable domain-wide delegation for that service account.
3. Create and download a JSON key for the service account.
4. Copy the service account's numeric client ID.
5. In Google Admin Console, go to **Security > Access and data control > API
   controls > Domain-wide delegation**.
6. Add the service account client ID with these scopes:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
https://www.googleapis.com/auth/admin.directory.group
https://www.googleapis.com/auth/admin.directory.group.member
https://www.googleapis.com/auth/admin.directory.orgunit
https://www.googleapis.com/auth/admin.directory.user
```

When `gws setup` asks for the credentials path, enter the downloaded service
account JSON key path.

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
        "https://www.googleapis.com/auth/admin.directory.customer.readonly",
        "https://www.googleapis.com/auth/admin.directory.group",
        "https://www.googleapis.com/auth/admin.directory.group.member",
        "https://www.googleapis.com/auth/admin.directory.orgunit",
        "https://www.googleapis.com/auth/admin.directory.user"
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
gws auth status
gws check connection
gws info domain
```

To list users from the configured Workspace domain:

```sh
gws print users
gws print users --limit 25
gws print users --json
```

To inspect one user:

```sh
gws info user user@example.com
gws info user user@example.com --json
```

To create a user, prefer `--password-file` so the initial password is not stored
in shell history:

```sh
gws create user user@example.com --given-name Ada --family-name Lovelace --password-file ./password.txt
gws create user user@example.com --given-name Ada --family-name Lovelace --password-file ./password.txt --org-unit /Engineering
gws create user user@example.com --given-name Ada --family-name Lovelace --password-file ./password.txt --change-password-at-next-login false
```

To suspend or unsuspend a user:

```sh
gws suspend user user@example.com
gws unsuspend user user@example.com
```

To update a user:

```sh
gws update user user@example.com --given-name Ada
gws update user user@example.com --family-name Byron
gws update user user@example.com --org-unit /Engineering
gws update user user@example.com --given-name Ada --family-name Byron --org-unit /Engineering
```

To list and inspect groups from the configured Workspace domain:

```sh
gws print groups
gws print groups --limit 25
gws print groups --json
gws info group group@example.com
gws info group group@example.com --json
```

To create or update a group:

```sh
gws create group group@example.com --name "Engineering"
gws create group group@example.com --name "Engineering" --description "Engineering team"
gws update group group@example.com --name "Engineering Team"
gws update group group@example.com --description "Primary engineering discussion group"
```

To list, add, or remove group members:

```sh
gws print group-members group@example.com
gws print group-members group@example.com --limit 25
gws add group-member group@example.com user@example.com
gws add group-member group@example.com user@example.com --role MANAGER
gws remove group-member group@example.com user@example.com
```

To list and inspect organizational units:

```sh
gws print ous
gws print ous --json
gws info ou /
gws info ou /Engineering
gws info ou /Engineering --json
```

To create or update organizational units:

```sh
gws create ou --name Engineering --parent /
gws create ou --name Engineering --parent / --description "Engineering users"
gws update ou /Engineering --name ProductEngineering
gws update ou /Engineering --parent /Departments
gws update ou /Engineering --description "Engineering users"
```

If you authorized `gws` before a new command or scope existed, rerun `gws setup`
so the OAuth token or domain-wide delegation grant includes the current scope
list.

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

- user delete command
- group delete command
- group membership sync command
- org-unit delete command
- GAM grammar compatibility beyond the small first command set
- batch and CSV execution
