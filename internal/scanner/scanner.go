package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
)

// ProgressCallback is called to report progress during scanning
// current: number of directories processed so far
// total: total number of directories to process
// message: status message (e.g., directory name being processed)
type ProgressCallback func(current, total int, message string)

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

// GitMetadata represents git repository metadata for a directory
type GitMetadata struct {
	IsGitRepo      bool   `json:"is_git_repo"`
	RemoteURL      string `json:"remote_url,omitempty"`
	CurrentBranch  string `json:"current_branch,omitempty"`
	HasUncommitted bool   `json:"has_uncommitted,omitempty"`
	StatusSummary  string `json:"status_summary,omitempty"`
}

// IsGitRepository checks if a directory contains a git repository
func IsGitRepository(dirPath string) bool {
	gitDir := filepath.Join(dirPath, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

// CollectGitMetadata collects git metadata for a directory using go-git
// Returns metadata with IsGitRepo=false if the directory is not a git repository
func CollectGitMetadata(dirPath string) (*GitMetadata, error) {
	// Try to open the repository using go-git
	repo, err := git.PlainOpen(dirPath)
	if err != nil {
		// Not a git repository or can't be opened
		return &GitMetadata{IsGitRepo: false}, nil
	}

	metadata := &GitMetadata{
		IsGitRepo: true,
	}

	// Get remote URL (prefer origin)
	remotes, err := repo.Remotes()
	if err == nil {
		for _, remote := range remotes {
			if remote.Config().Name == "origin" {
				urls := remote.Config().URLs
				if len(urls) > 0 {
					metadata.RemoteURL = urls[0]
					break
				}
			}
		}
		// If no origin found, use the first remote
		if metadata.RemoteURL == "" && len(remotes) > 0 {
			urls := remotes[0].Config().URLs
			if len(urls) > 0 {
				metadata.RemoteURL = urls[0]
			}
		}
	}

	// Get current branch
	head, err := repo.Head()
	if err == nil {
		metadata.CurrentBranch = head.Name().Short()
	}

	// Get git status (uncommitted changes)
	worktree, err := repo.Worktree()
	if err == nil {
		status, err := worktree.Status()
		if err == nil {
			metadata.HasUncommitted = !status.IsClean()

			// Build status summary similar to git status --porcelain format
			if !status.IsClean() {
				var statusLines []string
				count := 0
				for file, fileStatus := range status {
					if count >= 5 {
						break
					}
					// Format: XY filename (X = index status, Y = worktree status)
					// StatusCode.String() returns the single character code
					stagingCode := string(fileStatus.Staging)
					worktreeCode := string(fileStatus.Worktree)
					statusLine := fmt.Sprintf("%s%s %s", stagingCode, worktreeCode, file)
					statusLines = append(statusLines, statusLine)
					count++
				}

				if len(statusLines) > 0 {
					metadata.StatusSummary = strings.Join(statusLines, "; ")
					totalFiles := len(status)
					if totalFiles > 5 {
						metadata.StatusSummary += fmt.Sprintf(" ... (%d more)", totalFiles-5)
					}
				} else {
					metadata.StatusSummary = "clean"
				}
			} else {
				metadata.StatusSummary = "clean"
			}
		}
	}

	return metadata, nil
}

// DirectoryInfo represents metadata about a directory
type DirectoryInfo struct {
	Path        string       `json:"path"`
	GitMetadata *GitMetadata `json:"git_metadata,omitempty"`
}

// ScanDirectoriesWithMetadata scans a directory and returns top-level directories
// with their git metadata, respecting the provided scanner configuration.
// If progressCallback is not nil, it will be called to report progress.
func ScanDirectoriesWithMetadata(path string, ignoreDirs []string, includeHidden bool, progressCallback ProgressCallback) ([]DirectoryInfo, error) {
	dirs, err := ListTopLevelDirs(path, ignoreDirs, includeHidden)
	if err != nil {
		return nil, err
	}

	total := len(dirs)
	if progressCallback != nil {
		progressCallback(0, total, fmt.Sprintf("Found %d directories to scan", total))
	}

	infos := make([]DirectoryInfo, len(dirs))
	for i, dir := range dirs {
		if progressCallback != nil {
			dirName := filepath.Base(dir)
			progressCallback(i, total, fmt.Sprintf("Scanning: %s", dirName))
		}

		gitMetadata, err := CollectGitMetadata(dir)
		if err != nil {
			// If metadata collection fails, still include the directory but without metadata
			infos[i] = DirectoryInfo{Path: dir}
			if progressCallback != nil {
				progressCallback(i+1, total, fmt.Sprintf("Completed: %s (metadata error)", filepath.Base(dir)))
			}
			continue
		}
		infos[i] = DirectoryInfo{
			Path:        dir,
			GitMetadata: gitMetadata,
		}

		if progressCallback != nil {
			progressCallback(i+1, total, fmt.Sprintf("Completed: %s", filepath.Base(dir)))
		}
	}

	return infos, nil
}
