package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Cache holds the cached usage data.
type Cache struct {
	Version        int                         `json:"version"`
	LastScanUnixMs int64                       `json:"lastScanUnixMs"`
	Files          map[string]FileUsage        `json:"files"`
	Days           map[string]map[string][]int `json:"days"` // dayKey -> model -> packed usage
}

// FileUsage represents the usage data for a single log file.
type FileUsage struct {
	MtimeUnixMs int64                       `json:"mtimeUnixMs"`
	Size        int64                       `json:"size"`
	Days        map[string]map[string][]int `json:"days"`
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{
		Version: 1,
		Files:   make(map[string]FileUsage),
		Days:    make(map[string]map[string][]int),
	}
}

func defaultCacheRoot() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "Alter", "ccusage-min"), nil
}

func cacheFileURL(provider string, cacheRoot string) (string, error) {
	var root string
	var err error
	if cacheRoot != "" {
		root = cacheRoot
	} else {
		root, err = defaultCacheRoot()
		if err != nil {
			return "", err
		}
	}

	return filepath.Join(root, fmt.Sprintf("%s-v1.json", provider)), nil
}

// LoadCache loads the usage cache from disk for a given provider.
func LoadCache(provider string, cacheRoot string) (*Cache, error) {
	url, err := cacheFileURL(provider, cacheRoot)
	if err != nil {
		return NewCache(), err
	}

	data, err := os.ReadFile(url)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCache(), nil // Return new cache if file doesn't exist
		}
		return NewCache(), err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return NewCache(), err
	}

	if cache.Version != 1 {
		return NewCache(), nil
	}
	if cache.Files == nil {
		cache.Files = make(map[string]FileUsage)
	}
	if cache.Days == nil {
		cache.Days = make(map[string]map[string][]int)
	}

	return &cache, nil
}

// SaveCache saves the usage cache to disk.
func SaveCache(provider string, cache *Cache, cacheRoot string) error {
	url, err := cacheFileURL(provider, cacheRoot)
	if err != nil {
		return err
	}

	dir := filepath.Dir(url)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	// Write to temp file then rename, for atomic write.
	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp-%s.json", uuid.New().String()))
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, url)
}
