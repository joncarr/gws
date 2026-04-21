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
gws info domain example.com
gws info domain-alias alias.example.com
gws info group group@example.com
gws info group-member group@example.com user@example.com
gws info ou /Engineering
gws info user user@example.com
gws print group-aliases group@example.com
gws print group-members group@example.com
gws print groups
gws print ous
gws print user-aliases user@example.com
gws print users
gws print domains
gws print domain-aliases
gws add group-member group@example.com user@example.com
gws create domain example.com
gws create domain-alias alias.example.com --parent example.com
gws create group-alias group@example.com alias@example.com
gws create group group@example.com
gws create user-alias user@example.com alias@example.com
gws create user user@example.com
gws create ou
gws delete domain example.com
gws delete domain-alias alias.example.com
gws delete group-alias group@example.com alias@example.com
gws delete group group@example.com
gws delete ou /Engineering
gws delete user-alias user@example.com alias@example.com
gws delete user user@example.com
gws remove group-member group@example.com user@example.com
gws sync group-members group@example.com
gws make admin user@example.com
gws revoke admin user@example.com
gws suspend user user@example.com
gws unsuspend user user@example.com
gws update group group@example.com
gws update group-member group@example.com user@example.com
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

### Fastest Path

If you want to get `gws` working quickly:

1. In **Google Cloud Console**, create or choose a project.
2. Enable these APIs:
   - **Admin SDK API**
   - **Google Sheets API** if you want to use `--sheet`
3. Create a **Desktop app** OAuth client.
4. Download the OAuth client JSON file.
5. Run:

```sh
go build -o gws ./cmd/gws
./gws setup
./gws check connection
```

Use a Google Workspace admin account when Google asks you to authorize.

### What To Enable In Google Cloud

Your Google Cloud project should enable these APIs:

- **Admin SDK API**: required for core Google Workspace admin commands
- **Gmail API**: required for Gmail delegate management commands
- **Google Sheets API**: required for `print users --sheet` and
  `print groups --sheet`

If you do not plan to use Gmail delegation or `--sheet` yet, `gws` still works
best when all current scopes are authorized up front. If you add a feature
later and your token or delegation grant is older, rerun `gws setup`.

Setup asks for the scopes needed for the current command set:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
https://www.googleapis.com/auth/admin.directory.domain
https://www.googleapis.com/auth/admin.directory.group
https://www.googleapis.com/auth/admin.directory.group.member
https://www.googleapis.com/auth/admin.directory.orgunit
https://www.googleapis.com/auth/admin.directory.user
https://www.googleapis.com/auth/admin.directory.user.alias
https://www.googleapis.com/auth/gmail.settings.basic
https://www.googleapis.com/auth/gmail.settings.sharing
https://www.googleapis.com/auth/spreadsheets
```

These cover the current Admin SDK commands, Google Sheets export, and Gmail
delegate management. Gmail delegate commands specifically require a service
account profile with domain-wide delegation; OAuth user tokens cannot manage
delegates through the Gmail API.

### What `gws setup` Needs From You

`gws setup` will ask for:

- profile name
- primary Google Workspace domain
- Google Workspace admin email used for validation
- path to the OAuth client JSON or service account JSON file

For OAuth, setup will:

- validate the credentials JSON
- print an authorization URL
- start a temporary localhost callback listener
- save the OAuth token
- validate real Admin SDK access

For service accounts, setup will:

- validate the service account JSON
- show the service account client ID
- remind you to authorize domain-wide delegation scopes in the Google Admin console
- validate by impersonating the configured admin subject

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
5. Enable **Google Sheets API** if you want `--sheet` export.
6. Go to **APIs & Services > OAuth consent screen** and complete the required
   consent screen setup for your organization.
7. Go to **APIs & Services > Credentials**.
8. Select **Create credentials > OAuth client ID**.
9. Choose **Desktop app** as the application type.
10. Download the JSON credentials file.

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

### Service Account Setup

Use a service account only if you specifically want domain-wide delegation.

1. In Google Cloud Console, create a service account.
2. Enable domain-wide delegation for that service account.
3. Enable **Admin SDK API** in the Google Cloud project.
4. Enable **Gmail API** if you want Gmail delegate management.
5. Enable **Google Sheets API** if you want `--sheet` export.
6. Create and download a JSON key for the service account.
7. Copy the service account's numeric client ID.
8. In Google Admin Console, go to **Security > Access and data control > API
   controls > Domain-wide delegation**.
9. Add the service account client ID with these scopes:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
https://www.googleapis.com/auth/admin.directory.domain
https://www.googleapis.com/auth/admin.directory.group
https://www.googleapis.com/auth/admin.directory.group.member
https://www.googleapis.com/auth/admin.directory.orgunit
https://www.googleapis.com/auth/admin.directory.user
https://www.googleapis.com/auth/admin.directory.user.alias
https://www.googleapis.com/auth/gmail.settings.basic
https://www.googleapis.com/auth/gmail.settings.sharing
https://www.googleapis.com/auth/spreadsheets
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
        "https://www.googleapis.com/auth/admin.directory.domain",
        "https://www.googleapis.com/auth/admin.directory.group",
        "https://www.googleapis.com/auth/admin.directory.group.member",
        "https://www.googleapis.com/auth/admin.directory.orgunit",
        "https://www.googleapis.com/auth/admin.directory.user",
        "https://www.googleapis.com/auth/admin.directory.user.alias",
        "https://www.googleapis.com/auth/gmail.settings.basic",
        "https://www.googleapis.com/auth/gmail.settings.sharing",
        "https://www.googleapis.com/auth/spreadsheets"
      ],
      "output": "text"
    }
  }
}
```

OAuth tokens are stored with `0600` permissions at the configured token path.
Service account profiles do not use a token file.

### First Commands To Run After Setup

These are the quickest checks that `gws` is really connected:

```sh
gws auth status
gws check connection
gws info domain
gws print users --limit 5
gws print groups --limit 5
```

If `gws` reports missing scopes or insufficient permissions after you upgrade to
a newer version, rerun:

```sh
gws setup
```

That refreshes the saved config and makes Google issue a token with the current
scope list.

After setup, run either validation command at any time:

```sh
gws auth status
gws check connection
gws info domain
```

To list and inspect Workspace domains and domain aliases:

```sh
gws print domains
gws print domains --fields domainName,isPrimary --format csv
gws print domains --fields domainName,verified --sheet
gws info domain example.com
gws create domain secondary.example.com
gws delete domain secondary.example.com --confirm
gws print domain-aliases
gws print domain-aliases --fields domainAliasName,parentDomainName --format csv
gws info domain-alias alias.example.com
gws create domain-alias alias.example.com --parent example.com
gws delete domain-alias alias.example.com --confirm
```

To list users from the configured Workspace domain:

```sh
gws print users
gws print users --limit 25
gws print users --limit all
gws print users --domain example.com
gws print users --query "isSuspended=false"
gws print users --org-unit /Engineering
gws print users --sort familyName --order asc
gws print users --show-deleted
gws print users --fields primaryEmail,name,suspended
gws print users --fields primaryEmail,orgUnitPath --format csv
gws print users --fields primaryEmail,orgUnitPath --format json
gws print users --fields primaryEmail,name --sheet
gws print users --json
```

`--query` is passed to the Admin SDK Directory API user search parameter. You
can combine it with `--org-unit`; `gws` appends an `orgUnitPath='/PATH'` search
clause for you.

Supported user fields for `--fields`:

```text
primaryEmail, name, givenName, familyName, suspended, archived,
orgUnitPath, isAdmin, isDelegatedAdmin, isEnrolledIn2SV, isEnforcedIn2SV,
isMailboxSetup, includeInGlobalAddressList, aliases, nonEditableAliases,
recoveryEmail, recoveryPhone, suspensionReason, isGuestUser, agreedToTerms,
changePasswordAtNextLogin, ipWhitelisted, thumbnailPhotoURL, emails, phones,
addresses, organizations, relations, externalIDs, locations, id,
creationTime, lastLoginTime, deletionTime
```

`--sheet` creates a new Google Sheet owned by the authenticated account, writes
the selected columns, and prints the spreadsheet URL. If your profile was set
up before Sheets export existed, rerun `gws setup` so the saved token includes
the Google Sheets scope.

When Google rejects a command, `gws` tries to explain whether the problem is
missing scopes, a disabled API, service-account delegation, or an invalid query
or update payload.

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

To grant or revoke super administrator status:

```sh
gws make admin user@example.com
gws revoke admin user@example.com
```

To update a user:

```sh
gws update user user@example.com --given-name Ada
gws update user user@example.com --family-name Byron
gws update user user@example.com --org-unit /Engineering
gws update user user@example.com --given-name Ada --family-name Byron --org-unit /Engineering
gws update user user@example.com --recovery-email recover@example.com --recovery-phone +15551234567
gws update user user@example.com --change-password-at-next-login true --include-in-global-address-list false
gws update user user@example.com --phones-json '[{"value":"+15551234567","type":"work","primary":true}]'
gws update user user@example.com --organizations-json '[{"name":"Example Corp","title":"Staff Engineer","primary":true}]'
gws update user user@example.com --archived true
```

Structured profile flags use raw JSON because the Admin SDK accepts nested
arrays/objects for values like phones, addresses, organizations, locations,
relations, and external IDs.

To delete a user:

```sh
gws delete user user@example.com --confirm
```

To manage user aliases:

```sh
gws print user-aliases user@example.com
gws print user-aliases user@example.com --fields alias,primaryEmail --format csv
gws create user-alias user@example.com alias@example.com
gws delete user-alias user@example.com alias@example.com --confirm
```

To list and inspect groups from the configured Workspace domain:

```sh
gws print groups
gws print groups --limit 25
gws print groups --limit all
gws print groups --domain example.com
gws print groups --user user@example.com
gws print groups --query "email:engineering"
gws print groups --sort email --order desc
gws print groups --fields email,name,directMembersCount
gws print groups --fields email,directMembersCount --format csv
gws print groups --fields email,directMembersCount --format json
gws print groups --fields email,name --sheet
gws print groups --json
gws info group group@example.com
gws info group group@example.com --json
```

`--user` lists groups for a specific member. `--query` is passed to the Admin
SDK Directory API group search parameter.

Supported group fields for `--fields`:

```text
email, name, description, directMembersCount, adminCreated, aliases,
nonEditableAliases, id, etag, kind
```

To create or update a group:

```sh
gws create group group@example.com --name "Engineering"
gws create group group@example.com --name "Engineering" --description "Engineering team"
gws update group group@example.com --name "Engineering Team"
gws update group group@example.com --description "Primary engineering discussion group"
gws update group group@example.com --email engineering@example.com
```

To delete a group:

```sh
gws delete group group@example.com --confirm
```

To manage group aliases:

```sh
gws print group-aliases group@example.com
gws print group-aliases group@example.com --fields alias,primaryEmail --format csv
gws create group-alias group@example.com alias@example.com
gws delete group-alias group@example.com alias@example.com --confirm
```

To list, inspect, add, update, or remove group members:

```sh
gws print group-members group@example.com
gws print group-members group@example.com --limit 25
gws print group-members group@example.com --fields email,role --format csv
gws print group-members group@example.com --fields email,role,status --sheet
gws info group-member group@example.com user@example.com
gws add group-member group@example.com user@example.com
gws add group-member group@example.com user@example.com --role MANAGER
gws update group-member group@example.com user@example.com --role OWNER
gws sync group-members group@example.com --members ada@example.com,grace@example.com --dry-run
gws sync group-members group@example.com --members ada@example.com --role MANAGER --dry-run
gws sync group-members group@example.com --members-file ./members.txt --ignore-role --confirm
gws sync group-members group@example.com --members-file ./members.txt --confirm
gws sync group-members group@example.com --members-csv ./members.csv --dry-run
gws sync group-members group@example.com --members-sheet https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit --sheet-range "Members!A:B" --confirm
gws remove group-member group@example.com user@example.com
```

`sync group-members` compares the current direct membership against the desired
email list. By default it is role-aware for the target role, which means it can
add missing members, remove extra members with that role, and update existing
members to the requested role. Use `--role OWNER|MANAGER|MEMBER` to target a
specific role. Use `--ignore-role` to treat the group as a flat membership set
and remove extras regardless of role. Use `--dry-run` first to inspect the
plan. Unless `--dry-run` is set, `--confirm` is required because the command
can remove existing members.

For multi-role sync, use a CSV file or Google Sheet with at least an `email`
column and, optionally, a `role` column:

```text
email,role
ada@example.com,OWNER
grace@example.com,MEMBER
linus@example.com,MANAGER
```

When you use `--members-csv` or `--members-sheet`, `gws` treats that input as
the full desired membership map for the group and reconciles adds, removals,
and role changes in one pass.

## Batch execution

`gws` now has initial batch groundwork for running multiple commands from a file
with bounded concurrency:

```sh
gws batch run --file ./commands.txt
gws batch csv --file ./users.csv --command 'update user "{{email}}" --org-unit "{{orgUnit}}"'
gws batch run --file ./commands.txt --workers 8
gws batch run --file ./commands.txt --timeout 30s
gws batch run --file ./commands.txt --dry-run
gws batch run --file ./commands.txt --fail-fast
```

Batch files are line-based:

```text
# comments are ignored
gws version
print users --limit 10
update group eng@example.com --description "Primary engineering group"
```

Rules:
- blank lines and `#` comments are ignored
- each non-comment line is one command
- lines may start with `gws`, but do not have to
- simple single-quoted and double-quoted arguments are supported
- execution uses a worker pool so independent commands can run concurrently
- `--timeout` limits each command individually
- each command's output is captured and printed in input order
- results are summarized after the batch completes

For CSV-driven expansion, use `gws batch csv` with `{{column}}` placeholders:

```text
email,orgUnit
ada@example.com,/Engineering
grace@example.com,/Sales
```

```sh
gws batch csv --file ./users.csv --command 'update user "{{email}}" --org-unit "{{orgUnit}}"'
```

For common admin workflows, `gws` can print starter CSV helpers:

```sh
gws batch template user-update
gws batch template user-update --example
gws batch template group-member-sync --example
gws batch template user-suspend --example
gws batch template user-unsuspend --example
gws batch template user-make-admin --example
gws batch template user-revoke-admin --example
gws batch template group-create --example
gws batch template group-member-add --example
gws batch template group-member-remove --example
gws batch template ou-create --example
gws batch template ou-update --example
```

Current helper workflows:
- `user-update`
- `user-unsuspend`
- `user-make-admin`
- `user-revoke-admin`
- `group-member-sync`
- `user-suspend`
- `group-create`
- `group-member-add`
- `group-member-remove`
- `ou-create`
- `ou-update`

To list and inspect organizational units:

```sh
gws print ous
gws print ous --fields orgUnitPath,name --format csv
gws print ous --fields orgUnitPath,name,parentOrgUnitPath --sheet
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

To delete an organizational unit:

```sh
gws delete ou /Engineering --confirm
```

To manage Gmail delegates, use a service account profile with domain-wide
delegation:

```sh
gws print gmail-delegates user@example.com
gws info gmail-delegate user@example.com delegate@example.com
gws create gmail-delegate user@example.com delegate@example.com
gws delete gmail-delegate user@example.com delegate@example.com --confirm
```

These commands require the Gmail API plus the Gmail delegation scopes and are
not available through OAuth user-token profiles.

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

- group membership sync command
- GAM grammar compatibility beyond the small first command set
- batch and CSV execution
