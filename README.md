# gws

`gws` is a Go command-line tool for administering Google Workspace environments.
It borrows familiar command wording from GAMADV-XTD3 where that helps, but it is
not a line-by-line port. The goal is a clean, idiomatic Go codebase with a
strong setup experience, a reusable execution core, and incremental feature
growth.

The current project state is focused on:

- guided authentication and configuration
- core Admin SDK directory administration
- structured output for terminal, JSON, CSV, and Google Sheets
- bounded-concurrency batch execution
- first-pass Gmail delegate administration

## Status

`gws` is usable now for a meaningful set of Google Workspace admin workflows,
but it is still early-stage software.

What is solid today:

- guided `gws setup` flow
- profile/config management
- Admin SDK connectivity validation
- users, groups, org units, domains, aliases
- group membership sync, including role-aware sync
- list filtering, pagination, field selection, CSV/JSON output, Sheets export
- batch execution groundwork
- Gmail delegate create/list/delete/info

What is still incomplete:

- no full GAM grammar compatibility
- no full Google Workspace product coverage
- no live integration test suite checked into the repo yet
- setup still requires manual Google Cloud and Admin Console work
- Gmail support is limited to mailbox delegate settings, not message operations

## Build

```sh
go build ./cmd/gws
./gws help
```

During development, keep config isolated:

```sh
./gws --config ./dev-gws-config.json setup
./gws --config ./dev-gws-config.json config show
```

## Quick Start

If you want the fastest path to a working `gws` install:

1. Create or choose a Google Cloud project.
2. Enable:
   - `Admin SDK API`
   - `Google Sheets API`
   - `Gmail API` if you want Gmail delegate commands
3. Create a Desktop OAuth client.
4. Download the client JSON.
5. Run:

```sh
go build -o gws ./cmd/gws
./gws setup
./gws check connection
./gws print users --limit 5
```

If you later want Gmail delegate management, create a service account profile
with domain-wide delegation and rerun `gws setup` using the service account key.

## Setup and Authentication

`gws` currently supports two credential types:

- Desktop OAuth client JSON
- service account JSON with domain-wide delegation

OAuth is the easiest way to get started for Admin SDK commands. Service accounts
are required for Gmail delegate management.

### APIs to enable

Enable these APIs in Google Cloud Console:

- `Admin SDK API`: required for core Workspace admin commands
- `Google Sheets API`: required for `--sheet`
- `Gmail API`: required for Gmail delegate commands

### Required scopes

`gws setup` asks Google for the current scope set used by the implemented
commands:

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

If scopes change after you already authorized `gws`, rerun:

```sh
./gws setup
```

### What `gws setup` asks for

`gws setup` guides you through:

1. requirements and required scopes
2. profile details
3. credential validation
4. authorization or delegation expectations
5. a real Admin SDK validation call

The prompts ask for:

- profile name
- primary Workspace domain
- Workspace admin email used for validation and impersonation
- path to OAuth client JSON or service account JSON

### OAuth setup

Use OAuth if you want the easiest path to get `gws` working for Admin SDK
commands.

You need:

1. a Google Cloud project
2. `Admin SDK API` enabled
3. a configured OAuth consent screen
4. a Desktop OAuth client
5. the downloaded OAuth client JSON

Then run:

```sh
./gws setup
```

Setup will:

- validate the JSON file
- print an authorization URL
- start a temporary localhost callback listener
- save an OAuth token
- validate the resulting Admin SDK access

When setup prompts with:

```text
Path to OAuth client or service account JSON:
```

provide the path to the downloaded OAuth client JSON file.

### Service account setup

Use a service account if you want domain-wide delegation or Gmail delegate
commands.

You need:

1. a Google Cloud project
2. `Admin SDK API` enabled
3. `Gmail API` enabled for Gmail delegate commands
4. `Google Sheets API` enabled for `--sheet`
5. a service account with domain-wide delegation enabled
6. a JSON key for that service account
7. the service account client ID authorized in Google Admin Console under
   domain-wide delegation

Authorize these scopes in the Google Admin Console for the service account
client ID:

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

Then rerun:

```sh
./gws setup
```

using the service account JSON path when prompted.

### Config files and tokens

`gws setup` writes a config profile that records:

- active profile
- Workspace domain
- admin subject
- credentials file
- auth method
- token file for OAuth profiles
- scope list
- default output mode

OAuth token files are written with `0600` permissions. Service account profiles
do not use a token file.

### First commands after setup

These are the quickest sanity checks:

```sh
./gws auth status
./gws check connection
./gws info domain
./gws print users --limit 5
./gws print groups --limit 5
```

## Command Surface

The easiest way to inspect the live command surface is:

```sh
./gws help
```

Current commands are grouped below by area.

### Getting started

```text
gws version
gws help
gws setup [--profile default] [--domain example.com] [--admin admin@example.com] [--credentials client.json]
```

### Configuration and auth

```text
gws config show [--json]
gws auth status [--json]
gws check connection
```

### Domain commands

```text
gws info domain [example.com]
gws info domain-alias alias.example.com [--json]
gws print domains [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws print domain-aliases [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws create domain example.com [--json]
gws create domain-alias alias.example.com --parent example.com [--json]
gws delete domain example.com --confirm
gws delete domain-alias alias.example.com --confirm
```

Examples:

```sh
./gws print domains
./gws print domains --fields domainName,isPrimary --format csv
./gws print domains --fields domainName,verified --sheet
./gws info domain example.com
./gws print domain-aliases --fields domainAliasName,parentDomainName --format csv
```

### User commands

```text
gws info user user@example.com [--json]
gws print users [--limit 100|all] [--domain example.com] [--org-unit /PATH] [--query QUERY] [--show-deleted] [--sort email|familyName|givenName] [--order asc|desc] [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws print user-aliases user@example.com [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws create user user@example.com --given-name NAME --family-name NAME --password-file PATH [--org-unit /PATH] [--json]
gws create user-alias user@example.com alias@example.com [--json]
gws update user user@example.com [--given-name NAME] [--family-name NAME] [--org-unit /PATH] [--recovery-email addr@example.com] [--recovery-phone +15551234567] [--change-password-at-next-login true|false] [--archived true|false] [--include-in-global-address-list true|false] [--phones-json JSON] [--addresses-json JSON] [--organizations-json JSON] [--locations-json JSON] [--relations-json JSON] [--external-ids-json JSON] [--json]
gws suspend user user@example.com [--json]
gws unsuspend user user@example.com [--json]
gws make admin user@example.com
gws revoke admin user@example.com
gws delete user user@example.com --confirm
gws delete user-alias user@example.com alias@example.com --confirm
```

Examples:

```sh
./gws print users --limit 25
./gws print users --limit all --query "isSuspended=false"
./gws print users --org-unit /Engineering --sort familyName --order asc
./gws print users --fields primaryEmail,orgUnitPath --format csv
./gws print users --fields primaryEmail,name --sheet
./gws info user user@example.com
./gws create user user@example.com --given-name Ada --family-name Lovelace --password-file ./password.txt
./gws update user user@example.com --recovery-email recover@example.com --recovery-phone +15551234567
./gws update user user@example.com --phones-json '[{"value":"+15551234567","type":"work","primary":true}]'
./gws suspend user user@example.com
./gws make admin user@example.com
```

Supported `print users --fields` keys:

```text
primaryEmail, name, givenName, familyName, suspended, archived,
orgUnitPath, isAdmin, isDelegatedAdmin, isEnrolledIn2SV, isEnforcedIn2SV,
isMailboxSetup, includeInGlobalAddressList, aliases, nonEditableAliases,
recoveryEmail, recoveryPhone, suspensionReason, isGuestUser, agreedToTerms,
changePasswordAtNextLogin, ipWhitelisted, thumbnailPhotoURL, emails, phones,
addresses, organizations, relations, externalIDs, locations, id,
creationTime, lastLoginTime, deletionTime
```

### Group commands

```text
gws info group group@example.com [--json]
gws info group-member group@example.com member@example.com [--json]
gws print groups [--limit 100|all] [--domain example.com] [--user user@example.com] [--query QUERY] [--sort email] [--order asc|desc] [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws print group-members group@example.com [--limit 100] [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws print group-aliases group@example.com [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws create group group@example.com --name NAME [--description TEXT] [--json]
gws create group-alias group@example.com alias@example.com [--json]
gws update group group@example.com [--email new-group@example.com] [--name NAME] [--description TEXT] [--json]
gws add group-member group@example.com user@example.com [--role MEMBER] [--json]
gws update group-member group@example.com member@example.com --role OWNER|MANAGER|MEMBER [--json]
gws remove group-member group@example.com user@example.com
gws sync group-members group@example.com (--members user1@example.com,user2@example.com | --members-file PATH | --members-csv PATH | --members-sheet SHEET_ID_OR_URL [--sheet-range RANGE]) [--role MEMBER] [--ignore-role] [--dry-run] [--confirm]
gws delete group group@example.com --confirm
gws delete group-alias group@example.com alias@example.com --confirm
```

Examples:

```sh
./gws print groups --limit all
./gws print groups --user user@example.com
./gws print groups --query "email:engineering" --sort email --order desc
./gws print groups --fields email,directMembersCount --format csv
./gws print group-members group@example.com --fields email,role,status --sheet
./gws create group group@example.com --name "Engineering"
./gws update group group@example.com --email engineering@example.com
./gws add group-member group@example.com user@example.com --role MANAGER
./gws update group-member group@example.com user@example.com --role OWNER
./gws sync group-members group@example.com --members ada@example.com,grace@example.com --dry-run
./gws sync group-members group@example.com --members ada@example.com --role MANAGER --dry-run
./gws sync group-members group@example.com --members-file ./members.txt --ignore-role --confirm
./gws sync group-members group@example.com --members-csv ./members.csv --dry-run
./gws sync group-members group@example.com --members-sheet https://docs.google.com/spreadsheets/d/SPREADSHEET_ID/edit --sheet-range "Members!A:B" --confirm
```

Supported `print groups --fields` keys:

```text
email, name, description, directMembersCount, adminCreated, aliases,
nonEditableAliases, id, etag, kind
```

`sync group-members` behavior:

- `--members` or `--members-file`: sync against a flat list of emails
- `--role`: target a specific role such as `MEMBER`, `MANAGER`, or `OWNER`
- `--ignore-role`: treat the group as a flat membership set and remove extras
  regardless of role
- `--members-csv` or `--members-sheet`: treat the input as the full desired
  role map for the group
- `--dry-run`: preview changes
- `--confirm`: required for real changes unless `--dry-run` is set

Structured CSV or Sheet input uses at least an `email` column and optionally a
`role` column:

```text
email,role
ada@example.com,OWNER
grace@example.com,MEMBER
linus@example.com,MANAGER
```

### Org unit commands

```text
gws info ou /Engineering [--json]
gws print ous [--fields ...] [--format text|csv|json] [--sheet] [--json]
gws create ou --name NAME --parent /PATH [--description TEXT] [--json]
gws update ou /PATH [--name NAME] [--parent /PATH] [--description TEXT] [--json]
gws delete ou /PATH --confirm
```

Examples:

```sh
./gws print ous
./gws print ous --fields orgUnitPath,name,parentOrgUnitPath --sheet
./gws info ou /Engineering
./gws create ou --name Engineering --parent / --description "Engineering users"
./gws update ou /Engineering --parent /Departments
./gws delete ou /Engineering --confirm
```

### Gmail delegate commands

```text
gws print gmail-delegates user@example.com [--json]
gws info gmail-delegate user@example.com delegate@example.com [--json]
gws create gmail-delegate user@example.com delegate@example.com [--json]
gws delete gmail-delegate user@example.com delegate@example.com --confirm
```

Examples:

```sh
./gws print gmail-delegates user@example.com
./gws info gmail-delegate user@example.com delegate@example.com
./gws create gmail-delegate user@example.com delegate@example.com
./gws delete gmail-delegate user@example.com delegate@example.com --confirm
```

Important constraints:

- these commands require a service account profile
- domain-wide delegation must be configured
- `Gmail API` must be enabled
- OAuth user-token profiles cannot manage Gmail delegates

### Batch commands

```text
gws batch run --file PATH [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]
gws batch csv --file PATH --command TEMPLATE [--workers N] [--timeout 30s] [--dry-run] [--fail-fast]
gws batch template WORKFLOW [--example]
```

Examples:

```sh
./gws batch run --file ./commands.txt
./gws batch run --file ./commands.txt --workers 8 --timeout 30s
./gws batch run --file ./commands.txt --dry-run
./gws batch run --file ./commands.txt --fail-fast
./gws batch csv --file ./users.csv --command 'update user "{{email}}" --org-unit "{{orgUnit}}"'
./gws batch template user-update --example
./gws batch template group-member-sync --example
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
- the `gws` prefix is optional inside batch files
- simple single-quoted and double-quoted arguments are supported
- execution uses a worker pool
- output is captured per command and printed deterministically in input order
- `--timeout` limits each command individually
- `--fail-fast` stops scheduling new work after the first failure

Built-in batch template workflows:

- `user-update`
- `user-suspend`
- `user-unsuspend`
- `user-make-admin`
- `user-revoke-admin`
- `group-create`
- `group-member-add`
- `group-member-remove`
- `group-member-sync`
- `ou-create`
- `ou-update`

## Output Modes

`gws` supports:

- human-readable text
- JSON via `--json` or `--format json`
- CSV via `--format csv`
- Google Sheets export via `--sheet` on supported list commands

List-style commands support combinations of:

- `--fields`
- `--format text|csv|json`
- `--json`
- `--sheet`

Sheets export is currently implemented for these list commands:

- `print users`
- `print groups`
- `print group-members`
- `print ous`
- `print user-aliases`
- `print group-aliases`
- `print domains`
- `print domain-aliases`

`--sheet` creates a new spreadsheet, writes headers plus rows, and prints the
URL.

## Error Handling

`gws` tries to translate common Google failures into actionable guidance. The
current implementation distinguishes between:

- insufficient scopes
- disabled APIs
- invalid query or payload errors
- OAuth credential problems
- service account / domain-wide delegation problems

If Google rejects a request after new features were added, rerun:

```sh
./gws setup
```

## Environment overrides

These environment variables are supported:

```text
GWS_CONFIG
GWS_PROFILE
GWS_DOMAIN
GWS_ADMIN_SUBJECT
GWS_CREDENTIALS_FILE
```

## Current limitations

Current limitations are important because they affect operator expectations:

- `gws setup` does not yet create Google Cloud projects, service accounts, OAuth
  clients, or Admin Console domain-wide delegation entries for you
- Gmail support currently covers delegate administration only, not messages,
  labels, forwarding, filters, or mailbox content operations
- Google Sheets export currently creates new spreadsheets; it does not append to
  an existing sheet
- batch execution runs commands concurrently, but it does not yet model
  dependencies between commands
- there is no built-in live integration test harness for a real Workspace
  tenant yet
- command wording is GAM-inspired, but exact GAM grammar compatibility is not a
  goal in this phase
- areas like Reports, Drive ownership transfer, Calendar, Chrome management,
  Vault, and device management are not implemented yet

## What’s next

Near-term work is expected to stay focused on operational depth rather than a
huge jump in surface area. Likely next slices:

1. stronger setup automation for Google Cloud-side bootstrap, while still being
   explicit about the Admin Console steps that cannot be skipped
2. live smoke and integration testing against a real Workspace tenant
3. more Gmail admin workflows and batch helpers built on the current auth model
4. continued expansion into additional Google Workspace admin domains only after
   setup, auth, output, and test coverage stay coherent

## Additional docs

- [Architecture](docs/architecture.md)
- [Compatibility notes](docs/compatibility.md)
