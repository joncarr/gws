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

## Adding a Command

Add a command by:

1. Extending dispatch in `internal/commands`.
2. Keeping parsing small and explicit in `internal/cli` if new flag behavior is
   needed.
3. Adding tests with mocked Google/config dependencies.
4. Updating README or compatibility docs when user-visible behavior changes.
