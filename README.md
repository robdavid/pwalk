# pwalk

pwalk is a small, concurrent, depth-first directory walker written in Go. It reads directories in parallel, assembles the results into an ordered tree, and streams file and directory paths to a user-supplied callback.

## Features
- Parallel directory reading using a bounded worker pool
- Deterministic, depth-first traversal order
- Pluggable callback (`WalkFn`) that receives each path and any read error
- Simple CLI in `cmd/pwalk` for quick usage

## Build

```bash
go build ./cmd/pwalk
```

## Run (CLI)

Basic usage (prints files under a path):

```bash
./pwalk -threads 4 /some/path
```

Flags (from `cmd/pwalk/main.go`):
- `-quiet` : don't print file names, only totals
- `-usage` : sum file sizes (requires file info)
- `-threads N` : set number of worker threads (defaults to `runtime.NumCPU()`)

## Library usage

Use the library by calling `pwalk.Walk(root, fn, opts...)`.
- `root` is a string root path
- `fn` is an `assemble.WalkFn` (signature: `func(path.RootedPath, error, fs.DirEntry)`) 
- `opts` is an optional variadic list of configuration functions (for example to supply a custom workpool or change thread counts). If you don't supply any options, `pwalk` will create a suitable workpool internally.

Simple example (uses the default internal workpool):

```go
package main

import (
    "fmt"
    "io/fs"

    "github.com/robdavid/pwalk"
    "github.com/robdavid/pwalk/pkgs/path"
)

func main() {
    fn := func(p path.RootedPath, err error, ent fs.DirEntry) {
        if err != nil {
            fmt.Printf("%s: %v\n", p, err)
            return
        }
        fmt.Println(p.String())
    }

    pwalk.Walk(".", fn)
}
```

## Tests

Run all tests:

```bash
go test ./...
```

Test a package with coverage:

```bash
go test ./pkgs/assemble -cover
```

## Notes for contributors
- `pkgs/dir` expects entries to be sorted; tests that construct synthetic `dir.Dir` values must sort names accordingly.
- Use `RootedPath.Append()` to build subpaths, not string concatenation.
- Use `slog` for structured logging; components expose a `Log *slog.Logger` field for runtime diagnostics.

For design details, see `walk.go`, `pkgs/assemble/assemble.go`, and `pkgs/workpool/workpool.go`.
