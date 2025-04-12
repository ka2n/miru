package cache

import (
	"encoding/gob"
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

	defaultCache *Cache[string]
)

// Entry represents a cached item
type Entry[T any] struct {
	Value     T
	CreatedAt time.Time
}

// Cache provides a generic caching mechanism
type Cache[T any] struct {
	dir string
	ttl time.Duration
}

func init() {
	cacheHome, err := os.UserCacheDir()
	if err != nil {
		DefaultDir = filepath.Join(os.TempDir(), "miru")
	} else {
		DefaultDir = filepath.Join(cacheHome, "miru")
	}

	if err := os.MkdirAll(DefaultDir, 0755); err != nil {
		// 初期化エラーはログに記録
		// キャッシュが使えなくても機能は動作する
	}

	defaultCache = &Cache[string]{
		dir: DefaultDir,
		ttl: DefaultTTL,
	}
}

// GetOrSet retrieves a value from cache or stores it if it doesn't exist
func GetOrSet(key string, fn func() (string, error), forceUpdate bool) (string, error) {
	return defaultCache.GetOrSet(key, fn, forceUpdate)
}

// Clear removes all cached entries
func Clear() error {
	return defaultCache.Clear()
}

// SetTTL updates the cache TTL
func SetTTL(d time.Duration) {
	defaultCache.SetTTL(d)
}

// SetDir updates the cache directory
func SetDir(dir string) error {
	return defaultCache.SetDir(dir)
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
	path := filepath.Join(c.dir, normalizedKey+".gob")

	// キャッシュの読み込み試行（forceUpdate=falseの場合のみ）
	if !forceUpdate {
		if entry, err := c.loadEntry(path); err == nil {
			// TTLチェック
			if time.Since(entry.CreatedAt) < c.ttl {
				return entry.Value, nil
			}
		}
	}

	// 値の生成
	value, err := fn()
	if err != nil {
		var zero T
		return zero, err
	}

	// キャッシュの保存
	entry := Entry[T]{
		Value:     value,
		CreatedAt: time.Now(),
	}

	if err := c.saveEntry(path, entry); err != nil {
		return value, err // キャッシュの保存に失敗しても値は返す
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
	// ディレクトリの作成
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
