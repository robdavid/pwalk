package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"runtime"
	"strings"

	"github.com/robdavid/pwalk"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
	"github.com/robdavid/pwalk/pkgs/workpool"
)

func main() {
	var quiet bool
	var usage bool
	var size uint64
	var threads int
	var symlinks bool
	flag.BoolVar(&quiet, "quiet", false, "Doesn't print file names, only totals")
	flag.BoolVar(&usage, "usage", false, "Total file usage")
	flag.IntVar(&threads, "threads", runtime.NumCPU(), "Set number of threads")
	flag.BoolVar(&symlinks, "symlinks", false, "Follow symlinks to directories")
	flag.Parse()
	for _, file := range flag.Args() {
		count := 0
		var prevPath path.RootedPath
		fn := func(fpath path.RootedPath, ent fs.DirEntry, err error, void walk.Void) {
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %s\n", fpath, err)
			} else {
				if cmpPaths(fpath, prevPath) <= 0 {
					fmt.Fprintf(os.Stderr, "Error: paths not in order: %s < %s\n", fpath, prevPath)
				}
				prevPath = fpath
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
		wp := workpool.New(threads, 0)
		defer wp.Stop()
		var filter walk.PreProcessFn[walk.Void]
		if !symlinks {
			filter = walk.FilterDirSymLinks
		}
		pwalk.GenWalk(file, filter, fn, walk.ConfigWorkpool(wp))
		fmt.Printf("Total: %d\n", count)
		fmt.Printf("Max parallelism: %d\n", wp.MaxActive)
		if usage {
			fmt.Printf("Size:  %d\n", size)
		}
	}
}

func cmpPaths(a, b path.RootedPath) int {
	var i int
	if c := strings.Compare(a.Root(), b.Root()); c != 0 {
		return c
	}
	for i = range min(a.Len(), b.Len()) {
		if a.Get(i) < b.Get(i) {
			return -1
		} else if a.Get(i) > b.Get(i) {
			return 1
		}
	}
	if a.Len() < b.Len() {
		return -1
	} else if a.Len() > b.Len() {
		return 1
	} else {
		return 0
	}
}
