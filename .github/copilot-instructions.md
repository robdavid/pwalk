# AI Coding Instructions for pwalk

## Architecture Overview
pwalk is a parallel directory walker in Go that traverses file systems concurrently using a worker pool. The core flow: `Walk(root, fn, workpool)` spawns goroutines to read directories in parallel, assembles results into a tree structure, and streams files depth-first via the `WalkFn` callback.

Key components:
- **Profile**: Use `-threads` flag in main.go for performance tuning (defaults to `runtime.NumCPU()`)
- **Testing**: Use `testify/assert` for assertions; test concurrency with sync primitives (e.g., `workpool_test.go`).
## AI Coding Instructions for pwalk

Overview
 - Short: pwalk is a parallel, depth-first directory walker implemented in Go. It reads directories concurrently, assembles a tree representation, and streams paths in deterministic order to a callback.

Key components (files to inspect)
 - `pkgs/workpool`: Worker pool that accepts `Run(func())` and maintains worker goroutines. See `workpool.go` for lifecycle (`New`, `Run`, `Stop`, `Wait`).
 - `pkgs/assemble`: Receives `*dir.Dir` values and builds an in-memory assembly that `Next()` traverses in-order. Check `assemble.go` and `traverse.go` for `Add`, `Next`, and the `Location` type.
 - `pkgs/dir`: Wraps `fs.DirEntry` with directory-child links and implements `EntryIndex` used by `assemble` to attach children.
 - `pkgs/path`: Immutable path helpers: `Path` (use `Append()` not string joins).
 - `walk.go` / `cmd/pwalk/main.go`: shows how components compose: walker reads directories with `dir.Read`, pushes into an `assemble.Assembly`, and uses a `workpool` to parallelize directory reads.

Developer workflows (commands)
 - Build CLI: `go build ./cmd/pwalk` (produces `pwalk` binary).
 - Run tests: `go test ./...` (unit tests live under `pkgs/*` and top-level `walk_test.go`).
 - Run a package's tests and coverage: `go test ./pkgs/assemble -cover`.
 - Debug logging: set the `Log` field on `workpool.Workpool` or `assemble.Assembly` to a `slog` logger at `slog.LevelDebug`.

Project-specific conventions / patterns
 - Immutable Path objects: always use `Path.Append()` to construct subpaths.
 - Directory entries in `dir.Dir.Entries` are sorted lexicographically; test helpers must sort entries before creating synthetic `dir.Dir` objects (this repo uses `slices.BinarySearchFunc` in `EntryIndex`).
 - `assemble.Assembly` expects callers to stream `*dir.Dir` items via `Sink()`; `Add()` is internal to assembly threads but tests may call `Add()` directly when simulating arrival of directory reads.
 - Logging fields: many structs include `Log *slog.Logger` — default level is `Error`. Tests often set `Log` to discard output.

Testing notes
 - Tests use `github.com/stretchr/testify/assert` (declared in `go.mod`).
 - Concurrency tests use synchronization primitives (e.g., `sync.WaitGroup`, `time.Sleep`) and validate `workpool` activity counts.
 - For `assemble`, unit tests create synthetic `dir.Dir` objects; see `pkgs/assemble/assemble_test.go` for examples of constructing `DirEntryDir`/`DirEntryFile` test values.

Integration points
 - External deps: only `testify` is used (dev/test only). All other behaviour uses the Go stdlib (notably `io/fs`, `os`, `log/slog`).
 - CLI uses `flag` package and demonstrates real-world usage in `cmd/pwalk/main.go` (flags: `-quiet`, `-usage`, `-threads`).

When editing code
 - Preserve the `Path` and `Dir` invariants: entries must be sorted; `WithChild` is used to attach children in `assemble.Add`.
 - Prefer small, focused changes. Avoid modifying public API surface unless required.

Files to open first when onboarding
 - `walk.go`, `cmd/pwalk/main.go`, `pkgs/assemble/assemble.go`, `pkgs/dir/dir.go`, `pkgs/workpool/workpool.go`, `pkgs/path/path.go`.

If you need help
 - Ask for the specific package or function you want to modify and whether you need test scaffolding. The test harness uses `go test` and the `testify` helpers.

This file is intentionally concise: open the referenced files for deeper behavior. The repository's small size means reading those 6 files gives a complete mental model.

Additional: README has been added to the project root with usage examples.

- **Logging**: Components have `Log *slog.Logger` fields; default to `slog.LevelError` unless debugging.

## Integration Points
- External deps: Only `testify` for testing; pure Go stdlib otherwise.
- Cross-component: Workpool injects concurrency; Assemble manages state; Dir/Path handle data structures.
- CLI: `cmd/pwalk/main.go` demonstrates usage with flags for threads, quiet mode, usage calculation.

Reference: `walk.go` for API, `workpool.go` for concurrency patterns, `assemble.go` for streaming logic.