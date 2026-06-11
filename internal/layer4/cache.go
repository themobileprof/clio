package layer4

import (
	"clio/internal/layer3"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"strings"
	"time"
)

const maxCacheEntries = 80
const defaultCacheTTL = 7 * 24 * time.Hour

func hashQuery(query string) string {
	normalized := strings.TrimSpace(strings.ToLower(query))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func ensureCacheSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS query_cache (
			query_hash TEXT PRIMARY KEY,
			command TEXT NOT NULL,
			description TEXT,
			cached_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

// GetCached returns a previously cached remote result if still fresh.
func GetCached(query string, ttl time.Duration) (CommandResult, bool) {
	if ttl <= 0 {
		ttl = defaultCacheTTL
	}
	db, err := layer3.GetDB()
	if err != nil {
		return CommandResult{}, false
	}
	if err := ensureCacheSchema(db); err != nil {
		return CommandResult{}, false
	}

	hash := hashQuery(query)
	var cmd, desc string
	var cachedAt time.Time
	err = db.QueryRow(
		`SELECT command, COALESCE(description,''), cached_at FROM query_cache WHERE query_hash = ?`,
		hash,
	).Scan(&cmd, &desc, &cachedAt)
	if err != nil {
		return CommandResult{}, false
	}
	if time.Since(cachedAt) > ttl {
		_, _ = db.Exec(`DELETE FROM query_cache WHERE query_hash = ?`, hash)
		return CommandResult{}, false
	}
	return CommandResult{Name: cmd, Description: desc, Cached: true}, true
}

// PutCached stores a remote result locally so repeat queries cost zero network.
func PutCached(query string, result CommandResult) error {
	db, err := layer3.GetDB()
	if err != nil {
		return err
	}
	if err := ensureCacheSchema(db); err != nil {
		return err
	}

	hash := hashQuery(query)
	_, err = db.Exec(`
		INSERT INTO query_cache (query_hash, command, description, cached_at)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(query_hash) DO UPDATE SET
			command=excluded.command,
			description=excluded.description,
			cached_at=CURRENT_TIMESTAMP
	`, hash, result.Name, result.Description)
	if err != nil {
		return err
	}

	// LRU trim — keep cache tiny on 2 GB phones
	var count int
	_ = db.QueryRow(`SELECT COUNT(*) FROM query_cache`).Scan(&count)
	if count > maxCacheEntries {
		_, _ = db.Exec(`
			DELETE FROM query_cache WHERE query_hash IN (
				SELECT query_hash FROM query_cache
				ORDER BY cached_at ASC
				LIMIT ?
			)
		`, count-maxCacheEntries)
	}
	return nil
}

// CacheStats returns entry count for diagnostics.
func CacheStats() (int, error) {
	db, err := layer3.GetDB()
	if err != nil {
		return 0, err
	}
	if err := ensureCacheSchema(db); err != nil {
		return 0, err
	}
	var n int
	err = db.QueryRow(`SELECT COUNT(*) FROM query_cache`).Scan(&n)
	return n, err
}

