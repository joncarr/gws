# Compatibility

`gws` uses GAMADV-XTD3 as inspiration for command wording and administrator
workflows, but it does not attempt full grammar compatibility in the initial
implementation.

## Targeted Command Style

The first command set favors readable, GAM-like phrases:

```text
gws print users
gws info user user@example.com
gws create user user@example.com
gws info domain
```

Only this initial subset exists today:

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
gws print groups --limit all
gws print groups --query "email:engineering" --sort email --order asc
gws print groups --user user@example.com
gws print groups --fields email,directMembersCount --format csv
gws print groups --fields email,name --sheet
gws print group-members group@example.com --fields email,role --format csv
gws print ous
gws print ous --fields orgUnitPath,name --format csv
gws print user-aliases user@example.com
gws print user-aliases user@example.com --fields alias,primaryEmail --format csv
gws print users
gws print users --limit all
gws print users --query "isSuspended=false" --org-unit /Engineering --sort familyName --order asc
gws print users --fields primaryEmail,orgUnitPath --format csv
gws print users --fields primaryEmail,name --sheet
gws print domains
gws print domains --fields domainName,isPrimary --format csv
gws print domain-aliases
gws print domain-aliases --fields domainAliasName,parentDomainName --format csv
gws add group-member group@example.com user@example.com
gws sync group-members group@example.com --members user1@example.com,user2@example.com --dry-run
gws sync group-members group@example.com --members user1@example.com,user2@example.com --role MANAGER --dry-run
gws sync group-members group@example.com --members-csv ./members.csv --dry-run
gws create domain example.com
gws create domain-alias alias.example.com --parent example.com
gws create group-alias group@example.com alias@example.com
gws create group group@example.com --name "Engineering"
gws create ou --name Engineering --parent /
gws create user-alias user@example.com alias@example.com
gws create user user@example.com --given-name Ada --family-name Lovelace --password-file ./password.txt
gws delete domain example.com --confirm
gws delete domain-alias alias.example.com --confirm
gws delete group-alias group@example.com alias@example.com --confirm
gws delete group group@example.com --confirm
gws delete ou /Engineering --confirm
gws delete user-alias user@example.com alias@example.com --confirm
gws delete user user@example.com --confirm
gws remove group-member group@example.com user@example.com
gws make admin user@example.com
gws revoke admin user@example.com
gws suspend user user@example.com
gws unsuspend user user@example.com
gws update group group@example.com --name "Engineering Team"
gws update group group@example.com --email engineering@example.com
gws update group-member group@example.com user@example.com --role OWNER
gws update ou /Engineering --description "Engineering users"
gws update user user@example.com --given-name Ada --family-name Byron
gws update user user@example.com --recovery-email recover@example.com --phones-json '[{"value":"+15551234567","type":"work"}]'
```

## Intentional Differences

- No broad GAM grammar parser yet.
- List filtering uses explicit flags such as `--query`, `--org-unit`,
  `--domain`, `--sort`, and `--order` instead of attempting to support every
  GAM selector variant.
- List field selection and CSV export use explicit flags such as `--fields` and
  `--format csv` instead of GAM's `todrive` or print-field grammar.
- Google Sheets export currently uses an explicit `--sheet` flag on supported
  list commands instead of a trailing GAM `todrive` token.
- Group membership sync uses explicit `--members` or `--members-file` input
  instead of broader GAM selector and CSV grammar. `gws` now supports
  role-aware sync through `--role`, a flat-membership mode through
  `--ignore-role`, and explicit-role reconciliation through `--members-csv` or
  `--members-sheet`.
- No obscure aliases or syntax variants yet.
- Batch execution now supports line-based command files through
  `gws batch run --file PATH`. `gws batch csv --file PATH --command TEMPLATE`
  now covers simple CSV-driven expansion, but broader GAM CSV grammar is still
  deferred. `gws batch template WORKFLOW` provides starter CSV headers and
  command templates for common admin tasks, but it is intentionally smaller in
  scope than GAM's broader CSV command language.
- Gmail delegation is now supported through explicit `print/info/create/delete
  gmail-delegate` commands. This slice is intentionally limited to mailbox
  delegation and requires a service account with domain-wide delegation.
- Setup is designed as a guided `gws` experience, not a clone of another tool's
  file layout.
- The first validation target is Admin SDK customer/domain access, not broad
  Drive, Calendar, Reports, or Vault coverage.

## Setup Compatibility Direction

GAMADV-XTD3 is a strong reference for admin onboarding quality: it teaches the
operator what credentials, APIs, scopes, and delegation steps are required.
`gws` follows that product goal, but the file layout and prompts are native to
this project.

Current setup supports Desktop OAuth clients and service accounts with
domain-wide delegation for the current Admin SDK, Gmail delegate, and Sheets
surface:

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

These scopes still stop short of broader Drive, Calendar, Chrome/device
management, Reports, and Vault areas.

## Migration Expectations

Early users should treat `gws` as a new tool with familiar command language.
Scripts written for GAMADV-XTD3 should not be expected to run unchanged.

Compatibility will grow command by command after the setup, auth, config, output,
and Google API boundaries are stable.
