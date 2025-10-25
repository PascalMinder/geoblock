package geoblock

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePersistencePath validates a target file path for persistence.
// Behavior:
//   - Empty or whitespace-only raw => returns "", nil (feature OFF).
//   - Returns the absolute, symlink-resolved path on success.
//   - Fails if the parent directory does not exist, is not a directory,
//     or is not writable (detected by a create+remove probe).
//
// It does not create directories.
func ValidatePersistencePath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}

	clean := filepath.Clean(raw)

	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If the file itself doesn't exist yet, EvalSymlinks will still resolve
		var pathErr *os.PathError
		if errors.As(err, &pathErr) && errors.Is(pathErr.Err, os.ErrNotExist) {
			// Try to resolve parent only; the leaf may be new.
			parent := filepath.Dir(abs)
			parentResolved, perr := filepath.EvalSymlinks(parent)
			if perr != nil {
				return "", fmt.Errorf("resolve parent symlinks: %w", perr)
			}
			resolved = filepath.Join(parentResolved, filepath.Base(abs))
		} else {
			return "", fmt.Errorf("eval symlinks: %w", err)
		}
	}

	parent := filepath.Dir(resolved)

	info, err := os.Stat(parent)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("parent folder does not exist: %s", parent)
		}
		return "", fmt.Errorf("stat parent folder: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("parent path is not a folder: %s", parent)
	}

	// Writability probe: create+remove a temp file in parent.
	f, err := os.CreateTemp(parent, "geoblock-probe-*")
	if err != nil {
		return "", fmt.Errorf("parent folder not writable: %s: %w", parent, err)
	}

	name := f.Name()
	if cerr := f.Close(); cerr != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close probe file: %w", cerr)
	}
	if rerr := os.Remove(name); rerr != nil {
		return "", fmt.Errorf("remove probe file: %w", rerr)
	}

	return resolved, nil
}
