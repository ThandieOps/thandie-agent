package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ThandieOps/thandie-agent/internal/scanner"
)

// ScanResult represents the cached results of a workspace scan
type ScanResult struct {
	WorkspacePath  string                  `json:"workspace_path"`
	ScannedAt      time.Time               `json:"scanned_at"`
	Directories    []string                `json:"directories"` // Deprecated: use DirectoryInfos instead
	Count          int                     `json:"count"`
	DirectoryInfos []scanner.DirectoryInfo `json:"directory_infos"`
}

// Cache manages scan result caching
type Cache struct {
	cacheDir string
}

// New creates a new cache instance
func New() (*Cache, error) {
	cacheDir, err := getCacheDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache directory: %w", err)
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{
		cacheDir: cacheDir,
	}, nil
}

// getCacheDir returns the platform-appropriate cache directory
func getCacheDir() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		// Fallback to home directory if cache dir unavailable
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		cacheDir = filepath.Join(homeDir, ".cache")
	}

	return filepath.Join(cacheDir, "thandie", "cache"), nil
}

// SaveScanResult saves scan results to the cache
// This method is deprecated in favor of SaveScanResultWithMetadata
func (c *Cache) SaveScanResult(workspacePath string, directories []string) error {
	// Convert directories to DirectoryInfos
	infos := make([]scanner.DirectoryInfo, len(directories))
	for i, dir := range directories {
		infos[i] = scanner.DirectoryInfo{Path: dir}
	}
	return c.SaveScanResultWithMetadata(workspacePath, infos)
}

// SaveScanResultWithMetadata saves scan results with metadata to the cache
func (c *Cache) SaveScanResultWithMetadata(workspacePath string, directoryInfos []scanner.DirectoryInfo) error {
	// Extract directory paths for backward compatibility
	directories := make([]string, len(directoryInfos))
	for i, info := range directoryInfos {
		directories[i] = info.Path
	}

	result := ScanResult{
		WorkspacePath:  workspacePath,
		ScannedAt:      time.Now(),
		Directories:    directories, // Keep for backward compatibility
		Count:          len(directoryInfos),
		DirectoryInfos: directoryInfos,
	}

	// Create a safe filename from workspace path (hash or sanitize)
	cacheFile := c.getCacheFilePath(workspacePath)

	// Marshal to JSON
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal scan result: %w", err)
	}

	// Write to file
	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// LoadScanResult loads the most recent scan result for a workspace
func (c *Cache) LoadScanResult(workspacePath string) (*ScanResult, error) {
	cacheFile := c.getCacheFilePath(workspacePath)

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("no cached scan result found for workspace: %s", workspacePath)
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	// Unmarshal JSON
	var result ScanResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache file: %w", err)
	}

	return &result, nil
}

// HasCachedResult checks if a cached scan result exists for a workspace
func (c *Cache) HasCachedResult(workspacePath string) bool {
	cacheFile := c.getCacheFilePath(workspacePath)
	_, err := os.Stat(cacheFile)
	return err == nil
}

// GetCacheFilePath returns the cache file path for a given workspace
func (c *Cache) GetCacheFilePath(workspacePath string) string {
	return c.getCacheFilePath(workspacePath)
}

// getCacheFilePath generates a safe cache file path from workspace path
func (c *Cache) getCacheFilePath(workspacePath string) string {
	// Use SHA256 hash of the workspace path for a safe, deterministic filename
	hash := sha256.Sum256([]byte(workspacePath))
	hashStr := hex.EncodeToString(hash[:])
	// Use first 16 characters of hash (sufficient for uniqueness)
	return filepath.Join(c.cacheDir, fmt.Sprintf("scan_%s.json", hashStr[:16]))
}

// GetCacheDir returns the cache directory path (for debugging/info)
func (c *Cache) GetCacheDir() string {
	return c.cacheDir
}

// ClearCache removes all cached scan results
func (c *Cache) ClearCache() error {
	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			filePath := filepath.Join(c.cacheDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				return fmt.Errorf("failed to remove cache file %s: %w", filePath, err)
			}
		}
	}

	return nil
}
