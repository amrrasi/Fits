package fits

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
)

// ScanDir walks root recursively and returns paths of all FITS files found.
// Recognised extensions: .fits, .fit, .fts (case-insensitive).
func ScanDir(root string) ([]string, error) {
	log := logger.S().With("root", root)

	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, &ScanError{Dir: root, Cause: err}
	}

	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// Log but continue — don't abort the whole scan for one bad entry
			log.Warnw("fits: scan walk error", "path", path, "err", err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if isFITSFile(path) {
			paths = append(paths, path)
			log.Debugw("fits: found file", "path", path)
		}
		return nil
	})
	if err != nil {
		return nil, &ScanError{Dir: root, Cause: err}
	}

	log.Infow("fits: scan complete", "found", len(paths))
	return paths, nil
}

func isFITSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".fits", ".fit", ".fts":
		return true
	}
	return false
}

// ScanError is returned when the scan directory cannot be walked.
type ScanError struct {
	Dir   string
	Cause error
}

func (e *ScanError) Error() string {
	return "fits: scan " + e.Dir + ": " + e.Cause.Error()
}

func (e *ScanError) Unwrap() error { return e.Cause }
