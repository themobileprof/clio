package layer3

import (
	"clio/internal/config"
	"database/sql"
	"fmt"
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

// ModuleMeta is module metadata without YAML content.
type ModuleMeta struct {
	ModuleID    string
	Name        string
	Description string
	Version     string
	Tags        string
}

var (
	dbInstance *sql.DB
	dbOnce     sync.Once
	dbErr      error
)

// GetDB returns the singleton DB instance, initializing it lazily.
func GetDB() (*sql.DB, error) {
	dbOnce.Do(func() {
		dbPath := config.GetDBPath()
		if dbPath == "" {
			dbErr = fmt.Errorf("could not resolve database path")
			return
		}
		if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
			dbErr = err
			return
		}

		migrateLegacyDB(dbPath)

		db, err := sql.Open("sqlite", dbPath)
		if err != nil {
			dbErr = err
			return
		}

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if err := configureSQLite(db); err != nil {
			db.Close()
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

// migrateLegacyDB renames ~/.clio/modules.db to clio.db for older installs.
func migrateLegacyDB(dbPath string) {
	if _, err := os.Stat(dbPath); err == nil {
		return
	}
	legacy := filepath.Join(filepath.Dir(dbPath), "modules.db")
	if _, err := os.Stat(legacy); err != nil {
		return
	}
	_ = os.Rename(legacy, dbPath)
}

func configureSQLite(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
	}
	if config.IsLiteProfile() {
		// Tight limits for 2 GB Termux devices
		pragmas = append(pragmas,
			"PRAGMA cache_size = -512",  // 512 KiB page cache
			"PRAGMA mmap_size = 0",      // disable memory-mapped I/O
			"PRAGMA journal_mode = DELETE",
			"PRAGMA synchronous = NORMAL",
			"PRAGMA temp_store = FILE",
		)
	} else {
		pragmas = append(pragmas,
			"PRAGMA cache_size = -2000",
			"PRAGMA journal_mode = WAL",
		)
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("sqlite pragma %q: %w", p, err)
		}
	}
	return nil
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

	CREATE INDEX IF NOT EXISTS idx_modules_search ON modules(name, tags);
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

// ModuleExists reports whether a module ID is present (metadata only, no content load).
func ModuleExists(moduleID string) (bool, error) {
	db, err := GetDB()
	if err != nil {
		return false, err
	}
	var n int
	err = db.QueryRow("SELECT 1 FROM modules WHERE module_id = ? LIMIT 1", moduleID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
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

// ListModuleMeta returns metadata for every cached module.
func ListModuleMeta() ([]ModuleMeta, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT module_id, name, description, version, tags FROM modules ORDER BY module_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModuleMeta
	for rows.Next() {
		var m ModuleMeta
		if err := rows.Scan(&m.ModuleID, &m.Name, &m.Description, &m.Version, &m.Tags); err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// FindModuleMeta returns metadata for one module_id.
func FindModuleMeta(moduleID string) (*ModuleMeta, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	var m ModuleMeta
	err = db.QueryRow(
		`SELECT module_id, name, description, version, tags FROM modules WHERE module_id = ?`,
		moduleID,
	).Scan(&m.ModuleID, &m.Name, &m.Description, &m.Version, &m.Tags)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

// SearchModules searches the database for modules matching the given keywords.
// Only metadata columns are read — never loads YAML content or bash_script.
func SearchModules(keywords []string) ([]Module, error) {
	if len(keywords) == 0 {
		return nil, nil
	}

	db, err := GetDB()
	if err != nil {
		return nil, err
	}

	// Cap keywords on lite profile to reduce query size
	if config.IsLiteProfile() && len(keywords) > 2 {
		keywords = keywords[:2]
	}

	query := "SELECT id, name, description, module_id, tags FROM modules WHERE "
	args := []interface{}{}

	for i, kw := range keywords {
		if i > 0 {
			query += " OR "
		}
		query += "name LIKE ? OR description LIKE ? OR tags LIKE ?"
		term := "%" + kw + "%"
		args = append(args, term, term, term)
	}

	query += " LIMIT 3"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var modules []Module
	for rows.Next() {
		var m Module
		var modID, tags string
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &modID, &tags); err != nil {
			continue
		}
		m.Command = "clio-run-module " + modID
		m.Keywords = tags
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
