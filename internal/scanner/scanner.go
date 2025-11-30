package scanner

import (
	"os"
	"path/filepath"
	"strings"
)

// ListTopLevelDirs scans a directory and returns top-level directories,
// respecting the provided scanner configuration
func ListTopLevelDirs(path string, ignoreDirs []string, includeHidden bool) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	// Create a map for faster lookups
	ignoreMap := make(map[string]bool)
	for _, dir := range ignoreDirs {
		ignoreMap[dir] = true
	}

	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		dirName := e.Name()

		// Skip hidden directories if includeHidden is false
		if !includeHidden && strings.HasPrefix(dirName, ".") {
			continue
		}

		// Skip directories in the ignore list
		if ignoreMap[dirName] {
			continue
		}

		dirs = append(dirs, filepath.Join(path, dirName))
	}
	return dirs, nil
}
