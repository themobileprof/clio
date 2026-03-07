package layer3

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite" // CGO-free SQLite
)

type Module struct {
	ID          int
	Name        string
	Description string
	Command     string
	Keywords    string
}

var (
	dbInstance *sql.DB
	dbOnce     sync.Once
	dbErr      error
)

// GetDB returns the singleton DB instance, initializing it lazily.
func GetDB() (*sql.DB, error) {
	dbOnce.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			dbErr = err
			return
		}

		dbPath := filepath.Join(home, ".clio", "clio.db")
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			dbErr = err
			return
		}

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			dbErr = err
			return
		}

		if err := initSchema(db); err != nil {
			db.Close()
			dbErr = err
			return
		}

		dbInstance = db
	})
	return dbInstance, dbErr
}

func initSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS modules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		module_id TEXT UNIQUE,
		name TEXT NOT NULL,
		description TEXT,
		tags TEXT,
		version TEXT,
		content TEXT,
		bash_script TEXT,
		checksum TEXT,
		synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS sync_metadata (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		last_sync_timestamp TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

// UpsertModule inserts or updates a module in the database (without checksum)
func UpsertModule(modID, name, desc, tags, version, content, bashScript string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	query := `
    INSERT INTO modules (module_id, name, description, tags, version, content, bash_script)
    VALUES (?, ?, ?, ?, ?, ?, ?)
    ON CONFLICT(module_id) DO UPDATE SET
        name=excluded.name,
        description=excluded.description,
        tags=excluded.tags,
        version=excluded.version,
        content=excluded.content,
        bash_script=excluded.bash_script;
    `
	_, err = db.Exec(query, modID, name, desc, tags, version, content, bashScript)
	return err
}

// UpsertModuleWithChecksum inserts or updates a module with checksum tracking
func UpsertModuleWithChecksum(modID, name, desc, tags, version, content, bashScript, checksum string) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	query := `
    INSERT INTO modules (module_id, name, description, tags, version, content, bash_script, checksum, synced_at)
    VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
    ON CONFLICT(module_id) DO UPDATE SET
        name=excluded.name,
        description=excluded.description,
        tags=excluded.tags,
        version=excluded.version,
        content=excluded.content,
        bash_script=excluded.bash_script,
        checksum=excluded.checksum,
        synced_at=CURRENT_TIMESTAMP;
    `
	_, err = db.Exec(query, modID, name, desc, tags, version, content, bashScript, checksum)
	return err
}

// GetModuleChecksum returns the stored checksum for a module
func GetModuleChecksum(moduleID string) (string, error) {
	db, err := GetDB()
	if err != nil {
		return "", err
	}

	var checksum sql.NullString
	err = db.QueryRow("SELECT checksum FROM modules WHERE module_id = ?", moduleID).Scan(&checksum)
	if err != nil {
		return "", err
	}
	if !checksum.Valid {
		return "", nil
	}
	return checksum.String, nil
}

// GetLastSyncTimestamp returns when modules were last synced
func GetLastSyncTimestamp() (time.Time, error) {
	db, err := GetDB()
	if err != nil {
		return time.Time{}, err
	}

	var timestamp sql.NullTime
	err = db.QueryRow("SELECT last_sync_timestamp FROM sync_metadata WHERE id = 1").Scan(&timestamp)
	if err == sql.ErrNoRows || !timestamp.Valid {
		return time.Time{}, nil // Never synced before
	}
	return timestamp.Time, err
}

// SaveLastSyncTimestamp updates the last sync time
func SaveLastSyncTimestamp(t time.Time) error {
	db, err := GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO sync_metadata (id, last_sync_timestamp) VALUES (1, ?)
		ON CONFLICT(id) DO UPDATE SET last_sync_timestamp=excluded.last_sync_timestamp
	`, t)

	return err
}

// SearchModules searches the database for modules matching the given keywords.
func SearchModules(keywords []string) ([]Module, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	// Simple search: check if any keyword matches description, name or tags
	query := "SELECT id, name, description, '', tags FROM modules WHERE "
	args := []interface{}{}

	for i, kw := range keywords {
		if i > 0 {
			query += " OR "
		}
		query += "name LIKE ? OR description LIKE ? OR tags LIKE ?"
		term := "%" + kw + "%"
		args = append(args, term, term, term)
	}

	// Limit results
	query += " LIMIT 5"

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Printf("Query error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var modules []Module
	for rows.Next() {
		var m Module
		var tags string
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Command, &tags); err != nil {
			continue
		}
		m.Keywords = tags // Mapping tags to keywords struct field
		modules = append(modules, m)
	}
	return modules, nil
}

// GetModuleByID retrieves a module by its module_id and returns the full YAML content
func GetModuleByID(moduleID string) (string, error) {
	db, err := GetDB()
	if err != nil {
		return "", err
	}

	query := "SELECT content FROM modules WHERE module_id = ?"
	var content string
	err = db.QueryRow(query, moduleID).Scan(&content)
	if err != nil {
		return "", fmt.Errorf("module not found: %w", err)
	}

	return content, nil
}
