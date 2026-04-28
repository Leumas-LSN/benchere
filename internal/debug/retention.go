package debug

import (
	"os"
	"path/filepath"
	"time"
)

// CleanOldJobDirs walks root and deletes any direct subdirectory whose mtime
// is older than maxAge. Returns the count of removed directories. Errors are
// silently ignored, the function is intentionally a fire-and-forget on
// daemon startup.
func CleanOldJobDirs(root string, maxAge time.Duration) int {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		full := filepath.Join(root, e.Name())
		info, err := os.Stat(full)
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			if err := os.RemoveAll(full); err == nil {
				removed++
			}
		}
	}
	return removed
}
