package path

import (
	"os"
	"path/filepath"
	"strings"
)

// IsCaseInsensitive returns true if the filesystem at dir appears case-insensitive.
func IsCaseInsensitive(dir string) (bool, error) {
	f, err := os.CreateTemp(dir, "caseprobe-*")
	if err != nil {
		return false, err
	}
	orig := f.Name()
	f.Close()
	defer os.Remove(orig)

	base := filepath.Base(orig)
	dirpath := filepath.Dir(orig)
	altBase := strings.ToUpper(base)

	altPath := filepath.Join(dirpath, altBase)

	// Try to create the alternate name exclusively.
	f2, err := os.OpenFile(altPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err == nil {
		// Created distinct file => case-sensitive
		f2.Close()
		os.Remove(altPath)
		return false, nil
	}
	if os.IsExist(err) {
		// Creation failed because file exists (same name under different case) => case-insensitive
		return true, nil
	}

	// If we get here, creation failed for another reason (permissions etc).
	// Fall back to stat-based check: see if alt path resolves to the same file.
	fi1, err1 := os.Stat(orig)
	if err1 != nil {
		return false, err1
	}
	fi2, err2 := os.Stat(altPath)
	if err2 == nil {
		// Both exist; check if they are same file
		if os.SameFile(fi1, fi2) {
			return true, nil
		}
		return false, nil
	}
	if os.IsNotExist(err2) {
		// alt doesn't exist => case-sensitive
		return false, nil
	}
	return false, err2
}
