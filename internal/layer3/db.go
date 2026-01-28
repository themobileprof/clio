package layer3

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
    "log"

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
		content TEXT
	);
	`
	_, err := db.Exec(query)
    if err != nil {
        return fmt.Errorf("init schema: %w", err)
    }
	return nil
}

// UpsertModule inserts or updates a module in the database
func UpsertModule(modID, name, desc, tags, version, content string) error {
    db, err := GetDB()
    if err != nil {
        return err
    }
    
    query := `
    INSERT INTO modules (module_id, name, description, tags, version, content)
    VALUES (?, ?, ?, ?, ?, ?)
    ON CONFLICT(module_id) DO UPDATE SET
        name=excluded.name,
        description=excluded.description,
        tags=excluded.tags,
        version=excluded.version,
        content=excluded.content;
    `
    _, err = db.Exec(query, modID, name, desc, tags, version, content)
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
