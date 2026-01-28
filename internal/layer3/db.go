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
		name TEXT NOT NULL,
		description TEXT,
		command TEXT NOT NULL,
		keywords TEXT
	);
	`
	_, err := db.Exec(query)
    if err != nil {
        return fmt.Errorf("init schema: %w", err)
    }
    
    // Check if empty and seed purely for testing/demo purposes if needed
    // In a real app, this might come from a remote sync.
	return nil
}

// SearchModules searches the database for modules matching the given keywords.
func SearchModules(keywords []string) ([]Module, error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}

    // Simple search: check if any keyword matches description or name
    // Building dynamic query
    query := "SELECT id, name, description, command, keywords FROM modules WHERE "
    args := []interface{}{}
    
    for i, kw := range keywords {
        if i > 0 {
            query += " OR "
        }
        query += "name LIKE ? OR description LIKE ? OR keywords LIKE ?"
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
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.Command, &m.Keywords); err != nil {
			continue
		}
		modules = append(modules, m)
	}
	return modules, nil
}
