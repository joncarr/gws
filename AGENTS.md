# AGENTS.md

## Project

Build **gws**, a Go-based command-line tool for administering Google Workspace environments.

This project is inspired by GAMADV-XTD3, but it is **not** a line-by-line port and should not attempt full compatibility in one pass. The goal is a clean, maintainable, idiomatic Go codebase that can grow into a practical Google Workspace administration tool over time.

The executable name for this project is **`gws`**.

---

## Core Principles

1. **Keep dependencies minimal**

   * Prefer the Go standard library whenever practical.
   * Only introduce a third-party dependency when it clearly reduces complexity or risk.
   * Avoid framework-heavy designs.
   * Do not add dependencies just for convenience if the functionality is straightforward to implement in-house.

2. **Favor maintainability over feature count**

   * Build a solid command engine, config layer, auth layer, and Google API client abstractions first.
   * Add features in small, testable increments.
   * Do not try to recreate the entire GAM command surface in the first implementation.

3. **Idiomatic Go first**

   * Use clear package boundaries.
   * Prefer small interfaces and concrete types.
   * Use `context.Context` throughout execution paths.
   * Make use of Go concurrency for performance where possible.
   * Keep business logic out of `main.go`.

4. **Compatibility is staged, not absolute**

   * Use GAMADV-XTD3 (https://github.com/taers232c/GAMADV-XTD3) as inspiration and as a reference for command semantics.
   * Preserve familiar command wording where practical.
   * Prefer semantic compatibility over exact syntactic duplication.
   * Document unsupported or partially supported behaviors clearly.

5. **CLI and library separation**

   * The CLI should be a thin wrapper around reusable internal execution logic.
   * The core command execution engine should be usable programmatically.

---

## Naming

* Project name: **gws**
* Executable: **gws**
* Module/package names should use `gws` consistently.
* Do not use `gamgo` in code, docs, comments, examples, or package naming.

---

## Scope

### Initial scope

Focus on a small, high-value subset of Google Workspace administration tasks:

* version/help
* config/profile management
* authentication setup
* basic domain info
* users

  * print users
  * info user
  * create user
  * update user
  * suspend / unsuspend user
  * delete user
* groups

  * print groups
  * info group
  * create/update/delete group
  * add/remove/sync members
* org units

  * print ous
  * info ou
  * create/update/delete ou

### Deferred scope

These areas should be designed for, but not fully implemented initially:

* full GAM grammar compatibility
* batch execution
* CSV-driven execution
* Gmail message manipulation
* Drive ownership transfer flows
* Calendar event management
* Chrome / device management
* Reports
* Vault

---

## Dependency Guidance

### Preferred approach

Use only:

* Go standard library
* Official Google Go client libraries where needed
* OAuth/auth packages required for Google API access

### Avoid unless justified

* Large CLI frameworks
* Reflection-heavy command systems
* Overly abstract plugin systems
* Code generation unless it clearly improves maintainability
* Multiple competing config/logging libraries

### CLI framework guidance

Because this project values simplicity and minimal dependencies:

* Do **not** assume a heavy CLI framework is required.
* A small in-house command dispatcher built with the standard library is acceptable and preferred if it remains clean.
* If a CLI library is introduced later, it must be justified in the README and architecture docs.

---

## Architecture

Organize the project roughly like this:

```text
cmd/gws/main.go
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
pkg/gws/
docs/
```

### Package responsibilities

#### `cmd/gws/`

* CLI entrypoint only
* bootstrap config, logging, and app execution
* no business logic

#### `internal/app/`

* application bootstrap
* runtime wiring
* dependency construction
* execution orchestration

#### `internal/cli/`

* token handling
* command dispatch
* argument parsing for the supported syntax
* help/usage generation

#### `internal/commands/`

* one package or subpackages for command handlers
* command metadata
* validation and execution logic

#### `internal/config/`

* config file loading/saving
* profile/section support
* environment variable overrides
* runtime option resolution

#### `internal/auth/`

* OAuth and token handling
* service account support
* domain-wide delegation scaffolding
* credential storage abstraction

#### `internal/google/`

* thin wrappers around Google API clients
* isolate API-specific logic from command handlers
* keep API clients mockable for tests

#### `internal/output/`

* text output
* JSON output
* CSV output
* redirection support

#### `internal/logging/`

* logging initialization
* structured internal logging if needed
* user-facing error rendering kept separate from internal diagnostics

#### `internal/selectors/`

* entity selection logic such as user, group, org unit, and file selectors
* keep selectors independent from output formatting

#### `internal/batch/`

* future support for batch and CSV command execution
* leave stubs/interfaces if needed, but do not overbuild early

#### `pkg/gws/`

* reusable public API for programmatic execution
* keep the external surface small and stable

---

## Execution Model

The system should revolve around a central execution function, for example:

```go
func Execute(ctx context.Context, args []string) (int, error)
```

The command-line binary should do little more than:

1. create context
2. initialize config/logging
3. call the execution engine
4. exit with the returned status code

This mirrors the idea of a reusable command processor without copying the original implementation.

---

## Command Design

### General rules

* Commands should be explicit and readable.
* Prefer a stable internal representation of commands over ad hoc parsing.
* Do not over-engineer a full grammar engine on day one.
* Build a parser that is simple now but extensible later.

### Command style

Supported commands should initially follow a pragmatic structure such as:

```text
gws version
gws help
gws config show
gws config set key value
gws info domain
gws print users
gws info user user@example.com
gws create user user@example.com ...
gws update user user@example.com ...
```

### Compatibility guidance

* GAM-like wording such as `print users`, `info user`, `create user` is encouraged.
* Exact parity with the original grammar is not required initially.
* Do not implement obscure syntax variants before core admin workflows work reliably.

---

## Configuration

The config system should support:

* a default config file
* multiple profiles/sections
* credential/token paths
* selected admin subject / impersonation settings
* output defaults
* environment variable overrides

Configuration rules:

* prefer explicit defaults
* avoid hidden magic
* resolve values in a documented order
* make it easy to inspect the active configuration

---

## Output

Support these output modes:

* human-readable text
* JSON
* CSV

Output rules:

* keep user-facing output deterministic
* make JSON stable enough for scripting
* CSV should be consistent and easy to test with golden files
* separate output formatting from API fetching and business logic

---

## Errors and Exit Codes

* Return meaningful exit codes.
* User errors should be concise and actionable.
* Internal errors should preserve enough context for debugging.
* Do not swallow API errors.
* Avoid panic-based control flow.

---

## Testing Standards

Write tests from the start.

### Required testing approach

* table-driven unit tests
* parser tests
* command handler tests
* mock Google client tests
* golden tests for CLI output where helpful

### Testing rules

* every new command should include tests
* avoid untestable global state
* keep filesystem and network access abstracted when practical
* prefer deterministic tests over broad integration tests early on

---

## Documentation

Maintain at least these files:

### `README.md`

Should explain:

* what `gws` is
* current scope
* why the project exists
* how to build and run it
* how authentication/configuration work
* what is and is not implemented yet

### `docs/compatibility.md`

Should explain:

* which GAM-inspired command patterns are targeted
* which areas are intentionally deferred
* syntax differences from GAMADV-XTD3
* migration expectations for users

### `docs/architecture.md`

Should explain:

* package layout
* execution flow
* auth/config/output boundaries
* how to add a new command

---

## Implementation Phases

### Phase 1: foundation

Build:

* repo structure
* `gws version`
* `gws help`
* config loader/saver
* logging bootstrap
* central execution engine
* basic output abstraction

### Phase 2: auth and connectivity

Build:

* OAuth/token scaffolding
* service account scaffolding
* basic Google API client setup
* `gws info domain`
* config inspection commands

### Phase 3: core admin entities

Build:

* users subset
* groups subset
* org units subset

### Phase 4: batch and compatibility growth

Build:

* batch execution
* CSV execution
* additional GAM-like selectors and aliases
* more Google Workspace domains as needed

---

## Coding Style

* Use standard Go formatting.
* Keep functions short and focused.
* Prefer explicit code over clever abstractions.
* Avoid premature generalization.
* Comment why, not what.
* Preserve readability over terseness.

---

## What Agents Should Do

When working in this repository:

* preserve the minimal dependency philosophy
* keep `gws` as the only product name
* add features incrementally
* update docs when behavior changes
* add or update tests with each meaningful feature
* avoid broad refactors unless necessary
* prefer finishing a thin vertical slice over scattering unfinished scaffolding everywhere

---

## What Agents Should Not Do

* Do not rename the project back to `gamgo`.
* Do not introduce multiple unnecessary third-party libraries.
* Do not build full GAM compatibility before the foundation is stable.
* Do not put command logic directly in `main.go`.
* Do not tightly couple parsing, execution, output, and Google API calls.
* Do not add complex abstractions without a present need.

---

## Decision Heuristics

When uncertain, choose the option that is:

1. simpler
2. easier to test
3. easier to explain
4. easier to extend later
5. lower in dependency weight

If two designs are both valid, prefer the one with fewer moving parts.

---

## Immediate Objective

The immediate objective is to create a clean Go foundation for `gws` with:

* minimal dependencies
* reusable execution engine
* config/auth scaffolding
* a small but working set of Google Workspace admin commands
* clear docs and tests
