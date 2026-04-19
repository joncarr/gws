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
gws check connection
gws info domain
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
domain-wide delegation for this initial readonly Admin SDK scope:

```text
https://www.googleapis.com/auth/admin.directory.customer.readonly
```

## Migration Expectations

Early users should treat `gws` as a new tool with familiar command language.
Scripts written for GAMADV-XTD3 should not be expected to run unchanged.

Compatibility will grow command by command after the setup, auth, config, output,
and Google API boundaries are stable.
