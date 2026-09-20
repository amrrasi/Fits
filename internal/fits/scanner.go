package fits

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/amrrasi/fits/internal/logger"
)

func ScanDir(root string) ([]string, error) {
	log := logger.S().With("root", root)

	if fi, err := os.Stat(root); err != nil {
		return nil, &ScanError{Dir: root, Cause: err}
	} else if !fi.IsDir() {
		return nil, &ScanError{Dir: root, Cause: errors.New("مسیر یک پوشه نیست")}
	}

	var paths []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			log.Warnw("fits: scan walk error", "path", path, "err", err)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
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

type ScanError struct {
	Dir   string
	Cause error
}

func (e *ScanError) Error() string {
	return "fits: scan " + e.Dir + ": " + e.Cause.Error()
}

func (e *ScanError) Unwrap() error { return e.Cause }

func ResolveScanDir(root, requested string) (string, error) {
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", errors.New("پوشه‌ی اصلی اسکن روی سرور در دسترس نیست")
	}
	realRoot, _ = filepath.Abs(realRoot)
	target := realRoot
	if requested = strings.TrimSpace(requested); requested != "" {
		if strings.ContainsRune(requested, 0) {
			return "", errors.New("مسیر نامعتبر است")
		}
		if filepath.IsAbs(requested) {
			target = filepath.Clean(requested)
		} else {
			target = filepath.Join(realRoot, requested)
		}
	}
	real, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", errors.New("مسیر واردشده وجود ندارد")
	}
	rel, err := filepath.Rel(realRoot, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", errors.New("مسیر باید داخل پوشه‌ی مجاز اسکن باشد")
	}
	if fi, err := os.Stat(real); err != nil || !fi.IsDir() {
		return "", errors.New("مسیر واردشده یک پوشه نیست")
	}
	return real, nil
}
