package repl

import (
	"bufio"
	"clio/internal/intent"
	"clio/internal/safeexec"
	"fmt"
	"os"
	"strings"
)

// Run starts the REPL loop
func Run() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("CLIPilot Client (Offline First) - Type 'exit' to quit.")
	fmt.Println("-----------------------------------------------------")

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
    // Exact UI from guide
	fmt.Printf("\n✓ Command found: %s\n", res.Command)
	fmt.Println("────────────────────────")
	fmt.Printf("Purpose : %s\n\n", res.Description)
	fmt.Println("What would you like to do?")
	fmt.Println("  1) Show examples and usage  (recommended)")
	fmt.Println("  2) Run the command          (interactive)")
	fmt.Println("  3) Show command only        (exit)")
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
	case "2":
		runCommand(res, scanner)
	case "3":
		// Just exit the menu, which effectively does nothing as we loop back
		fmt.Println("Exiting menu.")
	case "4":
		fmt.Println("Enter new search query:")
        // Loop will continue naturally
	case "0":
		fmt.Println("Cancelled.")
	default:
		fmt.Println("Invalid choice.")
	}
    fmt.Println("")
}

func showExamples(res *intent.DetectionResult) {
    fmt.Println("\n--- Examples / Usage ---")
    // Simple fallback: try 'man' on the command name (usually first word)
    parts := strings.Fields(res.Command)
    if len(parts) > 0 {
        // Run man -f cmdName or just man cmdName?
        // Let's try displaying the description again or if we have usage from Layer 4
        fmt.Println(res.Description)
        
        // If Layer 2 or Layer 4 didn't give full usage, maybe try man
        if res.Source == "man" || res.Source == "static" {
             // Try a quick man lookup snippet?
             // For now just re-iterating description is safe.
             // Ideally we'd pull "EXAMPLES" section from man.
        }
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
