package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zjregee/alter/internal/utils"
)

type Cache struct {
	Version        int                         `json:"version"`
	LastScanUnixMs int64                       `json:"lastScanUnixMs"`
	Files          map[string]FileUsage        `json:"files"`
	Days           map[string]map[string][]int `json:"days"`
}

type FileUsage struct {
	MtimeUnixMs int64                       `json:"mtimeUnixMs"`
	Size        int64                       `json:"size"`
	Days        map[string]map[string][]int `json:"days"`
}

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

func cacheFilePath(provider string, cacheRoot string) (string, error) {
	if provider == "" || strings.ContainsAny(provider, "/\\") {
		return "", fmt.Errorf("invalid provider name: %s", provider)
	}

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

func LoadCache(provider string, cacheRoot string) (*Cache, error) {
	path, err := cacheFilePath(provider, cacheRoot)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewCache(), nil
		}
		return nil, err
	}

	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return NewCache(), nil
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

func SaveCache(provider string, cache *Cache, cacheRoot string) error {
	path, err := cacheFilePath(provider, cacheRoot)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	tmpFile := filepath.Join(dir, fmt.Sprintf(".tmp-%s.json", utils.GenerateUUID()))
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return err
	}
	return nil
}
