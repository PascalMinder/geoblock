package geoblock_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	geoblock "github.com/PascalMinder/geoblock"
)

func TestValidatePersistencePath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	okDir := filepath.Join(tmp, "ok")
	if err := os.MkdirAll(okDir, 0o755); err != nil {
		t.Fatalf("mkdir okDir: %v", err)
	}

	roDir := filepath.Join(tmp, "ro")
	if err := os.MkdirAll(roDir, 0o755); err != nil {
		t.Fatalf("mkdir roDir: %v", err)
	}
	// Make read-only for the writability probe.
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatalf("chmod roDir: %v", err)
	}
	// Restore permissions at the end (best-effort).
	t.Cleanup(func() { _ = os.Chmod(roDir, 0o755) })

	notDir := filepath.Join(tmp, "notdir")
	if err := os.WriteFile(notDir, []byte("x"), 0o600); err != nil {
		t.Fatalf("create non-dir file: %v", err)
	}

	type tc struct {
		name          string
		in            string
		wantEnabled   bool
		wantErr       bool
		skipOnWindows bool
	}
	cases := []tc{
		{
			name:        "empty string disables feature without error",
			in:          "   ",
			wantEnabled: false,
			wantErr:     false,
		},
		{
			name:        "ok new file under writable parent",
			in:          filepath.Join(okDir, "db.bin"),
			wantEnabled: true,
			wantErr:     false,
		},
		{
			name:    "missing parent directory",
			in:      filepath.Join(tmp, "missing", "db.bin"),
			wantErr: true,
		},
		{
			name:    "parent is a file (not a directory)",
			in:      filepath.Join(notDir, "x"),
			wantErr: true,
		},
		{
			name:          "read-only parent directory (not writable)",
			in:            filepath.Join(roDir, "db.bin"),
			wantErr:       true,
			skipOnWindows: true, // Windows ACLs can make this flaky
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			if c.skipOnWindows && runtime.GOOS == "windows" {
				t.Skip("skipping on Windows due to permission semantics")
			}

			out, err := geoblock.ValidatePersistencePath(c.in)

			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (enabled=%v, out=%q)", len(out) > 0, out)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(out) > 0 != c.wantEnabled {
				t.Fatalf("enabled: got %v, want %v", len(out) > 0, c.wantEnabled)
			}
			if c.wantEnabled && out == "" {
				t.Fatalf("expected non-empty output path when enabled")
			}
			if !c.wantEnabled && out != "" {
				t.Fatalf("expected empty output path when disabled, got %q", out)
			}
		})
	}
}
