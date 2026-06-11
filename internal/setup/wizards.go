package setup

import "strings"

// Wizard is an interactive setup/installation workflow backed by a YAML module.
type Wizard struct {
	ID          string
	ModuleID    string
	Flow        string
	Title       string
	Description string
	Icon        string
	Featured    bool
	Aliases     []string
	Phrases     [][]string
	Includes    []string
	Time        string
}

// AllWizards returns every registered setup wizard.
func AllWizards() []Wizard {
	return []Wizard{
		{
			ID: "termux", ModuleID: "termux_setup", Flow: "setup",
			Title: "Termux Setup", Icon: "⭐",
			Description: "Full phone dev environment (Zsh, storage, essentials)",
			Featured:    true,
			Aliases:     []string{"termux", "termux setup", "setup termux", "termux-setup", "configure termux"},
			Phrases: [][]string{
				{"setup", "termux"}, {"configure", "termux"}, {"termux", "wizard"},
				{"dev", "environment"}, {"development", "environment"},
				{"complete", "setup"}, {"first", "time", "termux"},
				{"setup", "phone"}, {"setup", "environment"}, {"install", "everything"},
				{"termux", "ready"}, {"make", "termux"}, {"abeg", "setup"}, {"wan", "setup"},
			},
			Includes: []string{
				"Package updates & Android storage access",
				"Zsh + Oh-My-Zsh, Vim, Git, GitHub CLI",
				"Optional: Python, Node.js, Go, PHP",
			},
			Time: "10–20 minutes",
		},
		{
			ID: "vim", ModuleID: "vim_setup", Flow: "setup",
			Title: "Vim Setup", Icon: "📝",
			Description: "Vim with vim-plug and dev plugins (NERDTree, ALE, Git)",
			Aliases:     []string{"vim", "vim setup", "setup vim", "vim-setup", "configure vim"},
			Phrases: [][]string{
				{"setup", "vim"}, {"configure", "vim"}, {"vim", "plugins"},
				{"vim", "config"}, {"install", "vim"}, {"vim", "editor"},
			},
			Includes: []string{
				"vim-plug plugin manager",
				"NERDTree, CtrlP, Airline, GitGutter, ALE",
				"Sensible defaults for coding on Termux",
			},
			Time: "5–10 minutes",
		},
		{
			ID: "git", ModuleID: "git_setup", Flow: "setup",
			Title: "Git Setup", Icon: "🔀",
			Description: "Git identity, aliases, defaults + GitHub CLI (gh)",
			Aliases:     []string{"git", "git setup", "setup git", "git-setup", "configure git", "gh setup"},
			Phrases: [][]string{
				{"setup", "git"}, {"configure", "git"}, {"install", "gh"},
				{"github", "cli"}, {"git", "github"}, {"git", "config"},
			},
			Includes: []string{
				"Install git and gh (GitHub CLI)",
				"Set your name and email for commits",
				"Useful aliases and recommended defaults",
			},
			Time: "3–5 minutes",
		},
		{
			ID: "devtools", ModuleID: "devtools_setup", Flow: "setup",
			Title: "Dev Tools Setup", Icon: "🛠️",
			Description: "Pick languages & frameworks: Python, Node, Go, PHP, Rust",
			Aliases:     []string{"devtools", "dev tools", "devtools setup", "setup devtools", "languages", "install python", "install nodejs", "install golang"},
			Phrases: [][]string{
				{"dev", "tools"}, {"setup", "languages"}, {"coding", "tools"},
				{"frontend", "backend"}, {"programming", "languages"},
				{"install", "languages"}, {"setup", "devtools"},
			},
			Includes: []string{
				"Interactive pick: Python, Node.js, Go, PHP, Rust",
				"Make & build tools where available",
				"Termux-aware package installs",
			},
			Time: "5–15 minutes",
		},
		{
			ID: "database", ModuleID: "database_setup", Flow: "setup",
			Title: "Database Setup", Icon: "🗄️",
			Description: "Choose databases: PostgreSQL, MySQL/MariaDB, Redis, SQLite",
			Aliases:     []string{"database", "db", "database setup", "setup database", "db setup", "install postgres", "install mysql", "install redis"},
			Phrases: [][]string{
				{"database", "setup"}, {"setup", "database"}, {"choose", "database"},
				{"db", "setup"}, {"install", "database"}, {"postgres", "database"},
				{"mysql", "database"}, {"redis", "database"},
			},
			Includes: []string{
				"Pick one or more: PostgreSQL, MariaDB, Redis, SQLite",
				"Termux notes (MongoDB not available on Termux)",
				"Quick start connection examples",
			},
			Time: "5–10 minutes",
		},
	}
}

// WizardByID returns a wizard by its short ID, or nil.
func WizardByID(id string) *Wizard {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, w := range AllWizards() {
		if w.ID == id {
			copy := w
			return &copy
		}
	}
	return nil
}

// RunCommandFor returns the clio-run-module command for a wizard.
func RunCommandFor(w Wizard) string {
	return "clio-run-module " + w.ModuleID + " " + w.Flow
}

// IsSetupModule reports whether a registry module ID is a first-class setup wizard.
func IsSetupModule(moduleID string) bool {
	moduleID = strings.ToLower(strings.TrimSpace(moduleID))
	for _, w := range AllWizards() {
		if strings.EqualFold(w.ModuleID, moduleID) {
			return true
		}
	}
	return false
}

// WizardFromCommand finds a wizard from a clio-run-module command string.
func WizardFromCommand(cmd string) *Wizard {
	cmd = strings.ToLower(cmd)
	for _, w := range AllWizards() {
		if strings.Contains(cmd, w.ModuleID) {
			copy := w
			return &copy
		}
	}
	return nil
}
