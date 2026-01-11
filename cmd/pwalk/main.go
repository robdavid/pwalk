package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"runtime"

	"github.com/robdavid/pwalk"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/workpool"
)

func main() {
	var quiet bool
	var usage bool
	var size uint64
	var threads int
	flag.BoolVar(&quiet, "quiet", false, "Doesn't print file names, only totals")
	flag.BoolVar(&usage, "usage", false, "Total file usage")
	flag.IntVar(&threads, "threads", runtime.NumCPU(), "Set number of threads")
	flag.Parse()
	for _, file := range flag.Args() {
		count := 0
		fn := func(fpath path.RootedPath, err error, ent fs.DirEntry) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", fpath, err)
			} else {
				count++
				if !quiet {
					fmt.Println(fpath)
				}
				if usage {
					if info, err := ent.Info(); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %s\n", fpath, err)
					} else {
						size += uint64(info.Size())
					}
				}
			}
		}
		func() {
			wp := workpool.New(threads, threads)
			defer wp.Stop()
			wp.Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))
			pwalk.Walk(file, fn, wp)
		}()
		fmt.Printf("Total: %d\n", count)
		if usage {
			fmt.Printf("Size:  %d\n", size)
		}
	}
}
