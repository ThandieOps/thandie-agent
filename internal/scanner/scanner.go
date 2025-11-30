package scanner

import (
	"os"
	"path/filepath"
)

func ListTopLevelDirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(path, e.Name()))
		}
	}
	return dirs, nil
}
