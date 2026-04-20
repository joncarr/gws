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
gws info group group@example.com
gws info ou /Engineering
gws info user user@example.com
gws print group-members group@example.com
gws print groups
gws print ous
gws print users
gws add group-member group@example.com user@example.com
gws create group group@example.com --name "Engineering"
gws create user user@example.com --given-name Ada --family-name Lovelace --password-file ./password.txt
gws remove group-member group@example.com user@example.com
gws suspend user user@example.com
gws unsuspend user user@example.com
gws update group group@example.com --name "Engineering Team"
gws update user user@example.com --given-name Ada --family-name Byron
```

## Intentional Differences

- No broad GAM grammar parser yet.
- No obscure aliases or syntax variants yet.
- No batch or CSV execution yet.
- Setup is designed as a guided `gws` experience, not a clone of another tool's
  file layout.
- The first validation target is Admin SDK customer/domain access, not broad
  user, group, Gmail, Drive, Calendar, Reports, or Vault coverage.

## Setup Compatibility Direction

GAMADV-XTD3 is a strong reference for admin onboarding quality: it teaches the
operator what credentials, APIs, scopes, and delegation steps are required.
`gws` follows that product goal, but the file layout and prompts are native to
this project.

Current setup supports Desktop OAuth clients and service accounts with
domain-wide delegation for the planned initial users/groups/org-unit admin
surface:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
https://www.googleapis.com/auth/admin.directory.group
https://www.googleapis.com/auth/admin.directory.group.member
https://www.googleapis.com/auth/admin.directory.orgunit
https://www.googleapis.com/auth/admin.directory.user
```

These scopes intentionally stop short of deferred areas such as Gmail, Drive,
Calendar, Chrome/device management, Reports, and Vault.

## Migration Expectations

Early users should treat `gws` as a new tool with familiar command language.
Scripts written for GAMADV-XTD3 should not be expected to run unchanged.

Compatibility will grow command by command after the setup, auth, config, output,
and Google API boundaries are stable.
