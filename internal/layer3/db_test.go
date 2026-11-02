package layer3

import (
	"database/sql"
	"testing"
)

func TestMigrateAddsBashScriptColumn(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Schema from before bash_script was added to CREATE TABLE.
	_, err = db.Exec(`
		CREATE TABLE modules (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			module_id TEXT UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			tags TEXT,
			version TEXT,
			content TEXT,
			checksum TEXT,
			synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	if err := initSchema(db); err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO modules (module_id, name, description, tags, version, content, bash_script, checksum)
		VALUES ('termux_setup', 'Termux', 'setup', 'termux', '1.2.1', 'yaml', 'bash', 'abc')
	`)
	if err != nil {
		t.Fatal(err)
	}

	var script string
	if err := db.QueryRow(`SELECT bash_script FROM modules WHERE module_id = 'termux_setup'`).Scan(&script); err != nil {
		t.Fatal(err)
	}
	if script != "bash" {
		t.Fatalf("bash_script = %q", script)
	}
}
