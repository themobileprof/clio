package repl

import (
	"bufio"
	"clio/internal/intent"
	"clio/internal/layer3"
	"clio/internal/modules"
	"clio/internal/safeexec"
	"clio/internal/setup"
	"fmt"
	"os"
	"strings"
)

// Run starts the REPL loop
func Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("CLIPilot Client (Offline First) - Type 'exit' to quit.")
	fmt.Println("-----------------------------------------------------")

	// Initialize builtin modules (extracts embedded YAML and loads into DB)
	if err := modules.EnsureBuiltinModulesLoaded(); err != nil {
		fmt.Printf("Warning: Failed to load builtin modules: %v\n", err)
	}

	// Check if on Termux and setup is needed
	if setup.IsTermux() && !setup.IsSetupComplete() {
		fmt.Println("\n💡 First time on Termux?")
		fmt.Println("   Run 'setup' for a complete development environment configuration.")
		fmt.Println("   (Includes Zsh, Vim, Git, LLM, and more)")
		fmt.Println()
	}

	for {
		fmt.Print(">> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if input == "exit" || input == "quit" {
			break
		}
		if input == "clear" {
			// clear screen
			print("\033[H\033[2J")
			continue
		}
		if input == "setup" {
			if !setup.IsTermux() {
				fmt.Println("⚠️  Termux setup is only available on Termux.")
				continue
			}

			// Load the termux_setup module
			yamlContent, err := layer3.GetModuleByID("termux_setup")
			if err != nil {
				fmt.Printf("Setup module not found: %v\n", err)
				fmt.Println("Try running 'sync' to download modules.")
				continue
			}

			module, err := modules.LoadModule(yamlContent)
			if err != nil {
				fmt.Printf("Failed to parse setup module: %v\n", err)
				continue
			}

			// Execute the setup flow
			if err := modules.ExecuteModule(module, "setup", scanner); err != nil {
				fmt.Printf("Setup error: %v\n", err)
			}
			continue
		}
		if input == "sync" {
			if err := modules.Sync(); err != nil {
				fmt.Printf("Sync error: %v\n", err)
			}
			continue
		}

		// Intent Detection
		result, err := intent.Detect(input)
		if err != nil {
			fmt.Printf("⚠ No matching command found for '%s'. Try rephrasing.\n", input)
			continue
		}

		handleResult(result, scanner)
	}
}

func handleResult(res *intent.DetectionResult, scanner *bufio.Scanner) {
	for {
		// Exact UI from guide (refreshed)
		fmt.Printf("\n✓ Use: %s\n", res.Command)
		fmt.Println("────────────────────────")
		fmt.Printf("Purpose : %s\n\n", res.Description)
		fmt.Println("What would you like to do?")
		fmt.Println("  1) Show examples and usage")
		fmt.Println("  2) Run the command")
		fmt.Println("  3) Copy command to clipboard (Print only)")
		fmt.Println("  4) Search for another command")
		fmt.Println("  0) Cancel")
		fmt.Println("")
		fmt.Print("Choice [1-4, 0]: ")

		if !scanner.Scan() {
			return
		}
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			showExamples(res)
			// Loop continues to let user run it after seeing examples
			fmt.Println("\nPress Enter to return to menu...")
			scanner.Scan()
		case "2":
			runCommand(res, scanner)
			return // Exit after running (usually what you want)
		case "3":
			fmt.Printf("\nCommand:\n\n    %s\n\n(Select and copy above)\n", res.Command)
			return
		case "4":
			return // Returns to main loop (new search)
		case "0":
			return // Returns to main loop
		default:
			fmt.Println("Invalid choice.")
		}
	}
}

func showExamples(res *intent.DetectionResult) {
	fmt.Println("\n--- Examples / Usage ---")
	fmt.Printf("Command: %s\n", res.Command)
	fmt.Printf("Details: %s\n", res.Description)

	// If it's a man page result, we could technically try to fetch more sections.
	// For now, let's just show helpful hints.
	if strings.HasPrefix(res.Command, "tar") {
		fmt.Println("\nTip: 'tar -xzvf' extracts .tar.gz files.")
		fmt.Println("     -x: extract, -z: gzip, -v: verbose, -f: file")
	}
}

func runCommand(res *intent.DetectionResult, scanner *bufio.Scanner) {
	fmt.Printf("\nRun: %s [y/N/edit]: ", res.Command)
	if !scanner.Scan() {
		return
	}
	ans := strings.ToLower(strings.TrimSpace(scanner.Text()))

	finalCmd := res.Command

	if ans == "edit" || ans == "e" {
		fmt.Print("Edit command: ")
		if scanner.Scan() {
			finalCmd = strings.TrimSpace(scanner.Text())
		}
	} else if ans != "y" && ans != "yes" {
		fmt.Println("Aborted.")
		return
	}

	// Execute safely
	parts := strings.Fields(finalCmd)
	if len(parts) == 0 {
		return
	}

	cmd := safeexec.Command(parts[0], parts[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing command: %v\n", err)
	}
}
