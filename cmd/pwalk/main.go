package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/robdavid/pwalk"
	"github.com/robdavid/pwalk/pkgs/dir"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/workpool"
)

func main() {
	var quiet bool
	var usage bool
	var size uint64
	flag.BoolVar(&quiet, "quiet", false, "Doesn't print file names, only totals")
	flag.BoolVar(&usage, "usage", false, "Total file usage")
	flag.Parse()
	for _, file := range flag.Args() {
		count := 0
		fn := func(p path.RootedPath, ent dir.DirEntry) {
			if !quiet {
				fmt.Println(p.Path())
			}
			if usage {
				if info, err := ent.Info(); err != nil {
					fmt.Fprintf(os.Stderr, "%s: %s", p.Path(), err)
				} else {
					size += uint64(info.Size())
				}
			}
			count++
		}
		func() {
			wp := workpool.New(12, 12)
			defer wp.Stop()
			wp.Log = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			pwalk.Walk(path.NewAt(file), fn, wp)
		}()
		fmt.Printf("Total: %d\n", count)
		if usage {
			fmt.Printf("Size:  %d\n", size)
		}
	}
}
