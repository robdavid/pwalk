package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/robdavid/pwalk"
	"github.com/robdavid/pwalk/pkgs/dir"
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
		fn := func(d dir.Dir, err error) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s", d.Path.Path())
			}
			for _, ent := range d.Entries {
				count++
				if !quiet {
					fmt.Println(d.Path.Append(ent.Name()))
				}
				if usage {
					if info, err := ent.Info(); err != nil {
						fmt.Fprintf(os.Stderr, "%s: %s", d.Path.Append(ent.Name()).Path(), err)
					} else {
						size += uint64(info.Size())
					}
				}
			}
		}
		func() {
			wp := workpool.New(threads, 1+threads/4)
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
