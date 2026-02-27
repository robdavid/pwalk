package assemble_test

import (
	"io/fs"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/robdavid/pwalk/pkgs/assemble"
	"github.com/robdavid/pwalk/pkgs/path"
	"github.com/robdavid/pwalk/pkgs/walk"
	"github.com/stretchr/testify/assert"
)

// testDirEntry implements fs.DirEntry for testing
type testDirEntry struct {
	name  string
	isDir bool
	size  int64
}

func (t testDirEntry) Name() string      { return t.name }
func (t testDirEntry) IsDir() bool       { return t.isDir }
func (t testDirEntry) Type() fs.FileMode { return 0 }
func (t testDirEntry) Info() (fs.FileInfo, error) {
	return testFileInfo{t}, nil
}

type testFileInfo struct {
	testDirEntry
}

func (t testFileInfo) Size() int64        { return t.size }
func (t testFileInfo) Mode() fs.FileMode  { return 0 }
func (t testFileInfo) ModTime() time.Time { return time.Time{} }
func (t testFileInfo) IsDir() bool        { return t.isDir }
func (t testFileInfo) Sys() any           { return nil }

func createTestDir(p path.Path, entries []testDirEntry, err error) *walk.Dir[walk.Void] {
	// Sort entries by name for binary search
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})
	dirEntries := make([]walk.DirEntry[walk.Void], len(entries))
	for i, ent := range entries {
		if ent.isDir {
			dirEntries[i] = walk.MakeUnmappedDirEntryDir[walk.Void](ent)
		} else {
			dirEntries[i] = walk.MakeUnmappedDirEntryFile[walk.Void](ent)
		}
	}
	return &walk.Dir[walk.Void]{Path: p, Entries: dirEntries, Error: err}
}

// TestAddAndNextSimple tests basic addition and traversal of a simple tree.
func TestAddAndNextSimple(t *testing.T) {
	as := assemble.New[walk.Void](walk.NewWalkConfig[walk.Void](), walk.Nil, nil)

	// Create root dir
	rootPath := path.New(rootPathStr)
	rootDir := createTestDir(rootPath, []testDirEntry{
		{"file1.txt", false, 100},
		{"subdir", true, 0},
	}, nil)

	// Add root
	as.Add(rootDir)

	// Create subdir
	subPath := rootPath.Append("subdir")
	subDir := createTestDir(subPath, []testDirEntry{
		{"file2.txt", false, 200},
	}, nil)

	// Add subdir
	as.Add(subDir)

	// Now call Next repeatedly
	var results []string
	for {
		fnext, _, direrr, more, blocked := as.Next()
		if !more {
			break
		}
		if blocked {
			t.Fatal("Should not be blocked")
		}
		assert.NoError(t, direrr)
		results = append(results, fnext.String())
	}

	expected := []string{
		rootPathStr,
		filepath.Join(rootPathStr, "file1.txt"),
		filepath.Join(rootPathStr, "subdir"),
		filepath.Join(rootPathStr, "subdir", "file2.txt"),
	}
	assert.Equal(t, expected, results)
}

// TestAddOutOfOrder simulates concurrent addition by adding directories incrementally,
// verifying that Next blocks when child directories aren't available yet, and resumes correctly after they're added.
func TestAddOutOfOrder(t *testing.T) {
	as := assemble.New[walk.Void](walk.NewWalkConfig[walk.Void](), walk.Nil, nil)

	rootPath := path.New(rootPathStr)

	// Add root with 'a'
	rootDir := createTestDir(rootPath, []testDirEntry{
		{"a", true, 0},
		{"file1.txt", false, 100},
	}, nil)
	as.Add(rootDir)

	// Call Next, should get root, then block on 'a'
	var results []string
	blockedCount := 0

	// First call
	fnext, _, direrr, more, blocked := as.Next()
	assert.True(t, more)
	assert.False(t, blocked)
	assert.NoError(t, direrr)
	results = append(results, fnext.String())

	// Second call, should block on 'a'
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.True(t, blocked)
	blockedCount++

	// Now add 'a'
	interPath := rootPath.Append("a")
	interDir := createTestDir(interPath, []testDirEntry{
		{"b", true, 0},
		{"file_a.txt", false, 150},
	}, nil)
	as.Add(interDir)

	// Third call, should get 'a'
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.False(t, blocked)
	assert.NoError(t, direrr)
	results = append(results, fnext.String())

	// Fourth call, should block on 'b'
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.True(t, blocked)
	blockedCount++

	// Now add 'b'
	deepPath := interPath.Append("b")
	deepDir := createTestDir(deepPath, []testDirEntry{
		{"file3.txt", false, 300},
	}, nil)
	as.Add(deepDir)

	// Fifth call, should get 'b'
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.False(t, blocked)
	assert.NoError(t, direrr)
	results = append(results, fnext.String())

	// Sixth call, file3
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.False(t, blocked)
	assert.NoError(t, direrr)
	results = append(results, fnext.String())

	// Seventh call, file_a
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.False(t, blocked)
	assert.NoError(t, direrr)
	results = append(results, fnext.String())

	// Eighth call, file1
	fnext, _, direrr, more, blocked = as.Next()
	assert.True(t, more)
	assert.False(t, blocked)
	assert.NoError(t, direrr)
	results = append(results, fnext.String())

	// No more
	fnext, _, direrr, more, blocked = as.Next()
	assert.False(t, more)

	assert.Greater(t, blockedCount, 0)

	expected := []string{
		rootPathStr,
		filepath.Join(rootPathStr, "a"),
		filepath.Join(rootPathStr, "a", "b"),
		filepath.Join(rootPathStr, "a", "b", "file3.txt"),
		filepath.Join(rootPathStr, "a", "file_a.txt"),
		filepath.Join(rootPathStr, "file1.txt"),
	}
	assert.Equal(t, expected, results)
}

// TestNextTraversalOrder ensures depth-first traversal order across multiple subdirectories.
func TestNextTraversalOrder(t *testing.T) {
	as := assemble.New[walk.Void](walk.NewWalkConfig[walk.Void](), walk.Nil, nil)

	rootPath := path.New(rootPathStr)

	// Create a more complex tree
	rootDir := createTestDir(rootPath, []testDirEntry{
		{"dir1", true, 0},
		{"dir2", true, 0},
		{"file1.txt", false, 100},
	}, nil)
	as.Add(rootDir)

	dir1Path := rootPath.Append("dir1")
	dir1Dir := createTestDir(dir1Path, []testDirEntry{
		{"file2.txt", false, 200},
		{"subdir", true, 0},
	}, nil)
	as.Add(dir1Dir)

	subdirPath := dir1Path.Append("subdir")
	subdirDir := createTestDir(subdirPath, []testDirEntry{
		{"file3.txt", false, 300},
	}, nil)
	as.Add(subdirDir)

	dir2Path := rootPath.Append("dir2")
	dir2Dir := createTestDir(dir2Path, []testDirEntry{
		{"file4.txt", false, 400},
	}, nil)
	as.Add(dir2Dir)

	// Traverse
	var results []string
	for {
		fnext, _, direrr, more, blocked := as.Next()
		if !more {
			break
		}
		if blocked {
			t.Fatal("All dirs added, should not block")
		}
		assert.NoError(t, direrr)
		results = append(results, fnext.String())
	}

	expected := []string{
		rootPathStr,
		filepath.Join(rootPathStr, "dir1"),
		filepath.Join(rootPathStr, "dir1", "file2.txt"),
		filepath.Join(rootPathStr, "dir1", "subdir"),
		filepath.Join(rootPathStr, "dir1", "subdir", "file3.txt"),
		filepath.Join(rootPathStr, "dir2"),
		filepath.Join(rootPathStr, "dir2", "file4.txt"),
		filepath.Join(rootPathStr, "file1.txt"),
	}
	assert.Equal(t, expected, results)
}

// TestNextWithErrors verifies error handling for directories with read errors.
func TestNextWithErrors(t *testing.T) {
	as := assemble.New[walk.Void](walk.NewWalkConfig[walk.Void](), walk.Nil, nil)

	rootPath := path.New(rootPathStr)
	rootDir := createTestDir(rootPath, []testDirEntry{
		{"baddir", true, 0},
		{"goodfile.txt", false, 100},
	}, nil)
	as.Add(rootDir)

	// Add bad dir with error
	badPath := rootPath.Append("baddir")
	badDir := createTestDir(badPath, []testDirEntry{}, assert.AnError)
	as.Add(badDir)

	var results []struct {
		path  string
		isErr bool
	}
	for {
		fnext, _, direrr, more, blocked := as.Next()
		if !more {
			break
		}
		if blocked {
			t.Fatal("Should not block")
		}
		results = append(results, struct {
			path  string
			isErr bool
		}{fnext.String(), direrr != nil})
	}

	// Check that baddir has error
	assert.Equal(t, rootPathStr, results[0].path)
	assert.False(t, results[0].isErr)
	assert.Equal(t, filepath.Join(rootPathStr, "baddir"), results[1].path)
	assert.True(t, results[1].isErr)
	assert.Equal(t, filepath.Join(rootPathStr, "goodfile.txt"), results[2].path)
	assert.False(t, results[2].isErr)
}

// TestEntryIndex tests the binary search functionality for finding entries.
func TestEntryIndex(t *testing.T) {
	entries := []testDirEntry{
		{"a", true, 0},
		{"b", false, 0},
		{"c", true, 0},
	}
	d := createTestDir(path.New("/test"), entries, nil)

	assert.Equal(t, 0, d.EntryIndex("a"))
	assert.Equal(t, 1, d.EntryIndex("b"))
	assert.Equal(t, 2, d.EntryIndex("c"))
	assert.Equal(t, -1, d.EntryIndex("d"))
}
