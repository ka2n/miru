package cache

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var (
	// allowedCharsPattern defines the pattern of characters allowed in cache keys
	// Allows: alphanumeric, hyphen, underscore, dot, and forward slash
	allowedCharsPattern = regexp.MustCompile(`^[a-zA-Z0-9\-_./]+$`)

	// DefaultTTL is the default time-to-live for cached entries
	DefaultTTL = 24 * time.Hour

	// DefaultDir is the default cache directory
	DefaultDir string
)

// Entry represents a cached item
type Entry[T any] struct {
	Value     T
	CreatedAt time.Time
}

const CACHE_VERSION = "v1"

// Cache provides a generic caching mechanism
type Cache[T any] struct {
	kind string
	dir  string
	ttl  time.Duration
}

func init() {
	prepareCacheDir()
}

func prepareCacheDir() {
	var baseDir string
	if os.Getenv("MIRU_NO_CACHE") == "1" {
		tmp, err := os.MkdirTemp(os.TempDir(), "miru")
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating temporary directory with MkdirTemp:", err)
			tmp = filepath.Join(os.TempDir(), "miru")
		}
		baseDir = tmp
	} else {
		cacheHome, err := os.UserCacheDir()
		if err != nil {
			baseDir = filepath.Join(os.TempDir(), "miru")
		} else {
			baseDir = filepath.Join(cacheHome, "miru")
		}
	}

	DefaultDir = filepath.Join(baseDir, CACHE_VERSION)

	if err := os.MkdirAll(DefaultDir, 0755); err != nil {
		// The functionality works even if the cache is unavailable
		fmt.Fprintln(os.Stderr, "Error creating cache directory:", err)
	}

	go cleanupOldCache(baseDir)
}

func cleanupOldCache(baseDir string) {
	// If other version of cache exists, remove it
	// List directories under the baseDir
	dirEntries, err := os.ReadDir(baseDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading cache directory:", err)
		return
	}
	// Iterate through the directories
	for _, entry := range dirEntries {
		if entry.IsDir() {
			dirName := entry.Name()
			// Check if the directory name starts with "v" and is not the current version
			if strings.HasPrefix(dirName, "v") && dirName != CACHE_VERSION {
				// Remove the directory
				err := os.RemoveAll(filepath.Join(baseDir, dirName))
				if err != nil {
					// Log error if needed
					fmt.Fprintln(os.Stderr, "Error removing old cache directory:", err)
				}
			}
		}
	}
}

func New[T any](kind string) *Cache[T] {
	return &Cache[T]{
		kind: kind,
		dir:  DefaultDir,
		ttl:  DefaultTTL,
	}
}

// normalizeKey converts a cache key into a filesystem-safe format
func normalizeKey(key string) string {
	// Replace any character that's not allowed with underscore
	normalized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '/' {
			return r
		}
		return '_'
	}, key)

	// Replace consecutive dots with a single dot
	for strings.Contains(normalized, "..") {
		normalized = strings.ReplaceAll(normalized, "..", ".")
	}

	// Replace consecutive slashes with a single slash
	for strings.Contains(normalized, "//") {
		normalized = strings.ReplaceAll(normalized, "//", "/")
	}

	return normalized
}

// GetOrSet retrieves a value from cache or stores it if it doesn't exist
func (c *Cache[T]) GetOrSet(key string, fn func() (T, error), forceUpdate bool) (T, error) {
	normalizedKey := normalizeKey(key)
	path := filepath.Join(c.dir, normalizedKey+"_"+c.kind+".gob")

	// Attempt to load from cache (only if forceUpdate=false)
	if !forceUpdate {
		if entry, err := c.loadEntry(path); err == nil {
			// TTL check
			if time.Since(entry.CreatedAt) < c.ttl {
				return entry.Value, nil
			}
		}
	}

	// Generate value
	value, err := fn()
	if err != nil {
		var zero T
		return zero, err
	}

	// Save to cache
	entry := Entry[T]{
		Value:     value,
		CreatedAt: time.Now(),
	}

	if err := c.saveEntry(path, entry); err != nil {
		return value, err // Return the value even if cache saving fails
	}

	return value, nil
}

func (c *Cache[T]) loadEntry(path string) (*Entry[T], error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entry Entry[T]
	if err := gob.NewDecoder(f).Decode(&entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

func (c *Cache[T]) saveEntry(path string, entry Entry[T]) error {
	// Create directory
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return gob.NewEncoder(f).Encode(entry)
}

// Clear removes all cached entries
func (c *Cache[T]) Clear() error {
	return os.RemoveAll(c.dir)
}

// SetTTL updates the cache TTL
func (c *Cache[T]) SetTTL(d time.Duration) {
	c.ttl = d
}

// SetDir updates the cache directory
func (c *Cache[T]) SetDir(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	c.dir = dir
	return nil
}

func Clear() error {
	return os.RemoveAll(DefaultDir)
}
