package fits

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveScanDir(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	ok := []string{"", "sub", filepath.Join(root, "sub"), "."}
	for _, r := range ok {
		if _, err := ResolveScanDir(root, r); err != nil {
			t.Errorf("%q should be allowed: %v", r, err)
		}
	}
	bad := []string{"..", "../x", outside, "/etc", "escape", "sub/../..", "nope", "sub\x00"}
	for _, r := range bad {
		if _, err := ResolveScanDir(root, r); err == nil {
			t.Errorf("%q must be rejected", r)
		}
	}
}
