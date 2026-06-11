package setup

import (
	"fmt"
	"strings"
)

// MatchKind describes how setup input was resolved.
type MatchKind int

const (
	MatchNone MatchKind = iota
	MatchMenu
	MatchWizard
)

// ResolveSetup resolves setup-related input to a wizard or menu request.
func ResolveSetup(input string) (MatchKind, *Wizard) {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if normalized == "" {
		return MatchNone, nil
	}

	// "setup <wizard>" or "setup <wizard> setup"
	if strings.HasPrefix(normalized, "setup ") {
		rest := strings.TrimSpace(normalized[6:])
		rest = strings.TrimSuffix(rest, " setup")
		if w := matchWizardToken(rest); w != nil {
			return MatchWizard, w
		}
	}

	for _, w := range AllWizards() {
		for _, alias := range w.Aliases {
			if normalized == alias {
				return MatchWizard, &w
			}
		}
	}

	// Bare "setup" / "wizards" → menu
	if isMenuRequest(normalized) {
		return MatchMenu, nil
	}

	tokens := tokenSet(normalized)
	for _, w := range AllWizards() {
		for _, terms := range w.Phrases {
			if allTermsPresent(tokens, terms) {
				copy := w
				return MatchWizard, &copy
			}
		}
	}

	return MatchNone, nil
}

func matchWizardToken(token string) *Wizard {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	// Allow module id aliases
	aliases := map[string]string{
		"termux_setup": "termux", "vim_setup": "vim", "git_setup": "git",
		"devtools_setup": "devtools", "database_setup": "database",
	}
	if id, ok := aliases[token]; ok {
		return WizardByID(id)
	}
	return WizardByID(token)
}

func isMenuRequest(normalized string) bool {
	switch normalized {
	case "setup", "wizards", "wizard", "install", "installations":
		return true
	}
	if IsTermux() && (normalized == "start" || normalized == "begin") {
		return true
	}
	return false
}

// IsSetupRequest reports whether input is asking for any setup wizard or menu.
func IsSetupRequest(input string) bool {
	kind, _ := ResolveSetup(input)
	return kind != MatchNone
}

// RunCommand returns the default wizard command (Termux setup on Termux, else menu hint).
func RunCommand() string {
	if IsTermux() {
		if w := WizardByID("termux"); w != nil {
			return RunCommandFor(*w)
		}
	}
	return "setup"
}

// ShortDescription summarizes setup for detection results.
func ShortDescription() string {
	return "Interactive setup wizards (Termux, Vim, Git, dev tools, databases)"
}

// ShowMenu prints the setup wizard catalog.
func ShowMenu() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  ⭐ SETUP WIZARDS — not the same as automation modules   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	ShowWizardCatalogSection(nil)
	fmt.Println("For day-to-day task workflows (copy files, check disk, etc.):")
	fmt.Println("  type 'modules' or 'catalog'")
	fmt.Println()
}

// ShowGuide shows the default setup entry point (menu on multi-wizard, or Termux guide).
func ShowGuide() {
	if IsTermux() && !IsSetupComplete() {
		if w := WizardByID("termux"); w != nil {
			ShowWizardGuide(*w, true)
			fmt.Println("Other wizards: type 'setup' to see Vim, Git, dev tools, databases.")
			return
		}
	}
	ShowMenu()
}

// ShowWizardGuide prints instructions for a specific wizard.
func ShowWizardGuide(w Wizard, firstRun bool) {
	if firstRun && w.ID == "termux" {
		fmt.Println("╔══════════════════════════════════════════════════════════╗")
		fmt.Println("║  ⭐ TERMUX SETUP — start here (first-class command)      ║")
		fmt.Println("║     Type 'setup' for all wizards · Zsh · Vim · Git       ║")
		fmt.Println("╚══════════════════════════════════════════════════════════╝")
	} else {
		fmt.Printf("╔══════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║  [SETUP WIZARD] %s %s\n", w.Icon, strings.ToUpper(w.Title))
		fmt.Printf("╚══════════════════════════════════════════════════════════╝\n")
	}
	fmt.Println()
	fmt.Println("  Type: environment setup (not a day-to-day automation task)")
	fmt.Printf("  %s\n", w.Description)
	if w.Time != "" {
		fmt.Printf("  ⏱  %s\n", w.Time)
	}
	fmt.Println()
	fmt.Println("Run this in your terminal:")
	fmt.Println()
	fmt.Printf("  %s\n", RunCommandFor(w))
	fmt.Println()
	if len(w.Includes) > 0 {
		fmt.Println("Includes:")
		for _, item := range w.Includes {
			fmt.Printf("  • %s\n", item)
		}
		fmt.Println()
	}
	if w.ID == "termux" && !IsSetupComplete() {
		fmt.Println("Tip: run 'sync' first if you haven't downloaded modules yet.")
	} else {
		fmt.Println("Safe to re-run anytime to update or reconfigure.")
	}
	fmt.Println()
}

func tokenSet(input string) map[string]bool {
	set := make(map[string]bool)
	for _, t := range strings.Fields(input) {
		t = strings.Trim(t, "?!.,;:\"'()")
		if t != "" {
			set[t] = true
		}
	}
	if set["abeg"] && set["setup"] {
		set["setup"] = true
	}
	if set["wan"] && (set["setup"] || set["start"]) {
		set["setup"] = true
	}
	return set
}

func allTermsPresent(set map[string]bool, terms []string) bool {
	for _, term := range terms {
		if !set[term] {
			return false
		}
	}
	return true
}
