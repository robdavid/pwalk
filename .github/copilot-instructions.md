# AI Coding Instructions for pwalk

## Architecture Overview
pwalk is a parallel directory walker in Go that traverses file systems concurrently using a worker pool. The core flow: `Walk(root, fn, workpool)` spawns goroutines to read directories in parallel, assembles results into a tree structure, and streams files depth-first via the `WalkFn` callback.

Key components:
- **Workpool** (`pkgs/workpool`): Manages concurrent workers with bounded queue; implements `Workpool` interface with `Run(func())` and `Wait()`.
- **Assemble** (`pkgs/assemble`): Collects concurrent directory reads into a tree, then iterates files in order; uses channels for streaming.
- **Dir** (`pkgs/dir`): Wraps `fs.DirEntry` with child directory links; `Read()` populates entries.
- **Path** (`pkgs/path`): Immutable path handling with `RootedPath` (root + subpath) and `Path` ([]string).

## Developer Workflows
- **Build binary**: `go build ./cmd/pwalk` (outputs `pwalk` executable)
- **Run tests**: `go test ./...` (covers main, path, workpool packages)
- **Debug concurrency**: Set workpool `Log` to `slog.LevelDebug` to trace worker activity
- **Profile**: Use `-threads` flag in main.go for performance tuning (defaults to `runtime.NumCPU()`)

## Code Patterns
- **Concurrency**: Always call `wp.Wait()` after `Walk()` to ensure completion; use `defer wp.Stop()` in main.
- **Error handling**: WalkFn receives errors per file; log via `slog` at appropriate levels.
- **Path operations**: Use `RootedPath.Append()` for immutable path building; avoid direct string concatenation.
- **Testing**: Use `testify/assert` for assertions; test concurrency with sync primitives (e.g., `workpool_test.go`).
- **Logging**: Components have `Log *slog.Logger` fields; default to `slog.LevelError` unless debugging.

## Integration Points
- External deps: Only `testify` for testing; pure Go stdlib otherwise.
- Cross-component: Workpool injects concurrency; Assemble manages state; Dir/Path handle data structures.
- CLI: `cmd/pwalk/main.go` demonstrates usage with flags for threads, quiet mode, usage calculation.

Reference: `walk.go` for API, `workpool.go` for concurrency patterns, `assemble.go` for streaming logic.