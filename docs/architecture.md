# Architecture

`gws` is organized around a reusable execution engine with a thin binary
entrypoint.

## Package Layout

```text
cmd/gws/main.go
pkg/gws/
internal/app/
internal/cli/
internal/commands/
internal/config/
internal/auth/
internal/google/
internal/output/
internal/logging/
internal/selectors/
internal/batch/
```

## Execution Flow

1. `cmd/gws/main.go` creates a context and calls `pkg/gws.Execute`.
2. `pkg/gws` exposes the small public API for programmatic execution.
3. `internal/app` parses arguments, resolves global options, and wires
   command dependencies.
4. `internal/commands` validates command intent and calls config, auth, output,
   and Google client boundaries.

The central call shape is:

```go
func Execute(ctx context.Context, args []string, opts Options) (int, error)
```

## Config Boundary

`internal/config` owns JSON config load/save, default paths, profiles, token
paths, and environment overrides. It does not call Google APIs.

## Auth Boundary

`internal/auth` owns credential detection, required scopes, OAuth token
acquisition, token persistence, and service-account domain-wide delegation HTTP
clients. It keeps setup mechanics separate from command parsing and Google API
resource calls.

The first OAuth flow uses a temporary localhost callback listener. Setup prints
the Google authorization URL, waits for the callback, exchanges the code, and
writes the token file with `0600` permissions.

Service account profiles use the official JWT config path with the configured
admin subject as the delegation subject. Setup explains that the service account
client ID and required scope must be authorized in the Google Admin console.

## Google Boundary

`internal/google` wraps official Google clients behind a small interface. Command
tests use mocks and do not touch the network.

The first real validation call uses Admin SDK Directory API customer lookup:

```text
customers.get("my_customer")
```

That call powers both `gws check connection` and `gws info domain`.

The first readonly entity command uses Admin SDK Directory API users list:

```text
users.list(domain="<configured-domain>", query="<optional-query>")
```

That call powers `gws print users`. User list flags are translated into a
small `google.UserListOptions` value before reaching the Directory client, so
command parsing stays separate from Google API parameter mapping. When
`--limit all` is used, the Directory client follows `nextPageToken` until every
page has been collected. Output shaping stays in `internal/commands`: the
command selects explicit user field specs and renders text tables, CSV, or JSON
without teaching the Google client about presentation concerns. Google Sheets
export reuses those same row builders and goes through a separate thin
`internal/google` Sheets client. The same command layer also translates common
Google API failures into clearer user-facing explanations for missing scopes,
disabled APIs, delegation/auth problems, and invalid request payloads.

Batch execution lives in `internal/batch/` as a small reusable engine. It
parses line-based command files into argument slices, then runs them through a
bounded worker pool via an executor callback. That keeps concurrency policy and
batch-file parsing out of `main.go` and out of the normal CLI parser. The same
package now also expands CSV rows into commands through a lightweight
`{{column}}` template renderer before handing those commands to the same worker
pool. Per-command timeout handling stays in the command layer so the reusable
batch engine does not need to know about CLI duration flags or child runner
construction. Common CSV workflows are exposed as small command-layer helpers
that print recommended headers plus a matching `gws batch csv` template.

Single-user inspection uses:

```text
users.get("user@example.com")
```

That call powers `gws info user user@example.com`.

User creation uses:

```text
users.insert(user)
```

That call powers `gws create user user@example.com`.

User suspension uses:

```text
users.patch("user@example.com", suspended=true|false)
```

That call powers `gws suspend user user@example.com` and
`gws unsuspend user user@example.com`.

User profile updates also use patch semantics:

```text
users.patch("user@example.com", name/orgUnitPath/recovery/profile fields)
```

That call powers `gws update user user@example.com`. The CLI maps simple scalar
flags directly and uses explicit `*-json` flags for nested Admin SDK profile
arrays like phones or organizations.

User deletion uses:

```text
users.delete("user@example.com")
```

That call powers `gws delete user user@example.com --confirm`.

Readonly group commands use:

```text
groups.list(domain="<configured-domain>", query="<optional-query>")
groups.get("group@example.com")
```

Those calls power `gws print groups` and `gws info group group@example.com`.
Group list flags are translated into `google.GroupListOptions`, and `print
groups --limit all` follows `nextPageToken` until the full result set is read.
Like user lists, group list output uses explicit field specs instead of
reflection so column ordering and field names stay stable. The same field and
row definitions are reused for CSV and Google Sheets export.

Group create/update commands use:

```text
groups.insert(group)
groups.patch("group@example.com", group)
```

Those calls power `gws create group` and `gws update group`.

Group deletion uses:

```text
groups.delete("group@example.com")
```

That call powers `gws delete group group@example.com --confirm`.

Group membership commands use:

```text
members.list("group@example.com")
members.insert("group@example.com", member)
members.delete("group@example.com", "user@example.com")
```

Those calls power `gws print group-members`, `gws add group-member`, and
`gws remove group-member`. `gws sync group-members` uses the full
`members.list` result to compute a desired/current difference, then applies
`members.insert`, `members.patch`, and `members.delete` for adds, role
changes, and removals. The sync command can build that desired state either
from flat email input (`--members` / `--members-file`) or from explicit-role
CSV/Sheets input (`--members-csv` / `--members-sheet`).

Readonly org-unit commands use:

```text
orgunits.list("my_customer", orgUnitPath="/", type="allIncludingParent")
orgunits.get("my_customer", "/Engineering")
```

Those calls power `gws print ous` and `gws info ou /Engineering`.

Org-unit create/update commands use:

```text
orgunits.insert("my_customer", orgUnit)
orgunits.patch("my_customer", "/Engineering", orgUnit)
```

Those calls power `gws create ou` and `gws update ou`.

Org-unit deletion uses:

```text
orgunits.delete("my_customer", "/Engineering")
```

That call powers `gws delete ou /Engineering --confirm`.

## Adding a Command

Add a command by:

1. Extending dispatch in `internal/commands`.
2. Keeping parsing small and explicit in `internal/cli` if new flag behavior is
   needed.
3. Adding tests with mocked Google/config dependencies.
4. Updating README or compatibility docs when user-visible behavior changes.
