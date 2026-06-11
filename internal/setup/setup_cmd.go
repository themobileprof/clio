package setup

import (
	"fmt"
	"strings"
)

const (
	// ModuleID is the termux_setup YAML module.
	ModuleID = "termux_setup"
	// FlowName is the default setup wizard flow.
	FlowName = "setup"
)

// RunCommand is the command students run to start the Termux dev environment wizard.
func RunCommand() string {
	return "clio-run-module " + ModuleID + " " + FlowName
}

// ShortDescription summarizes the setup wizard for menus and detection results.
func ShortDescription() string {
	return "Termux dev environment wizard (Zsh, Vim, Git, storage, optional languages)"
}

var setupExact = map[string]bool{
	"setup": true, "termux-setup": true, "termux setup": true,
	"setup termux": true, "configure termux": true,
}

// setupPhrases require all terms to appear (setup-related natural language).
var setupPhrases = [][]string{
	{"setup", "termux"},
	{"configure", "termux"},
	{"termux", "wizard"},
	{"dev", "environment"},
	{"development", "environment"},
	{"complete", "setup"},
	{"first", "time", "termux"},
	{"setup", "phone"},
	{"setup", "environment"},
	{"install", "everything"},
	{"termux", "ready"},
	{"make", "termux"},
	{"abeg", "setup"},
	{"wan", "setup"},
}

// IsSetupRequest reports whether input is asking for the Termux setup wizard.
func IsSetupRequest(input string) bool {
	normalized := strings.TrimSpace(strings.ToLower(input))
	if setupExact[normalized] {
		return true
	}
	// On Termux, bare "setup" family is always the wizard
	if IsTermux() && (normalized == "start" || normalized == "begin") {
		return true
	}

	tokens := tokenSet(normalized)
	for _, terms := range setupPhrases {
		if allTermsPresent(tokens, terms) {
			return true
		}
	}
	return false
}

func tokenSet(input string) map[string]bool {
	set := make(map[string]bool)
	for _, t := range strings.Fields(input) {
		t = strings.Trim(t, "?!.,;:\"'()")
		if t != "" {
			set[t] = true
		}
	}
	// Pidgin / casual
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

// ShowGuide prints the prominent Termux setup instructions.
func ShowGuide() {
	cmd := RunCommand()
	complete := IsSetupComplete()

	fmtHeader(complete)
	fmt.Println()
	fmt.Println("Run this in your Termux terminal:")
	fmt.Println()
	fmt.Printf("  %s\n", cmd)
	fmt.Println()
	fmt.Println("Includes:")
	fmt.Println("  • Package updates & storage access")
	fmt.Println("  • Zsh + Oh-My-Zsh, Vim, Git, GitHub CLI")
	fmt.Println("  • Optional: Python, Node, Go, PHP")
	fmt.Println("  • Takes about 10–20 minutes, run once")
	fmt.Println()
	if !complete {
		fmt.Println("Tip: run 'sync' first if you haven't downloaded modules yet.")
	} else {
		fmt.Println("Setup was completed before — safe to run again to update tools.")
	}
	fmt.Println()
}

func fmtHeader(complete bool) {
	if complete {
		fmt.Println("╔══════════════════════════════════════════════════════════╗")
		fmt.Println("║  🚀 TERMUX SETUP — full dev environment wizard           ║")
		fmt.Println("╚══════════════════════════════════════════════════════════╝")
		return
	}
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  ⭐ TERMUX SETUP — start here (first-class command)      ║")
	fmt.Println("║     Type 'setup' anytime · Zsh · Vim · Git · coding      ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
}
