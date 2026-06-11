package setup

import "fmt"

// ShowWizardCatalogSection lists setup wizards with clear [SETUP WIZARD] labeling.
func ShowWizardCatalogSection(isCached func(moduleID string) bool) {
	fmt.Println("── ⭐ SETUP WIZARDS ─────────────────────────────────────────")
	fmt.Println("   Install & configure your environment — usually run once")
	fmt.Println("   Ask naturally: \"setup termux\", \"install git and gh\", \"setup vim\"")
	fmt.Println("   Command: setup <name>")
	fmt.Println()

	for i, w := range AllWizards() {
		status := ""
		if isCached != nil {
			if isCached(w.ModuleID) {
				status = "  [✓ ready]"
			} else {
				status = "  [○ download on first use]"
			}
		}
		marker := " "
		if w.Featured && IsTermux() {
			marker = "⭐"
		}
		fmt.Printf("  %d) [SETUP WIZARD] %s %s — %s%s\n", i+1, marker, w.Title, w.Description, status)
		fmt.Printf("      start:  setup %s\n", w.ID)
		fmt.Printf("      run:    %s\n", RunCommandFor(w))
		if w.Time != "" {
			fmt.Printf("      time:   %s\n", w.Time)
		}
		fmt.Println()
	}
}
