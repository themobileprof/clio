package repl

import (
	"bufio"
	"clio/internal/intent"
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
		fmt.Println("   Complete development environment setup:")
		fmt.Println("   1. Run 'sync' to download modules")
		fmt.Println("   2. Run: clio-run-module termux_setup setup")
		fmt.Println("   (Zsh, Vim, Git, LLM, and more)")
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
			fmt.Println("╔══════════════════════════════════════════════════════════╗")
			fmt.Println("║       🚀 Module Execution                                ║")
			fmt.Println("╚══════════════════════════════════════════════════════════╝")
			fmt.Println()
			fmt.Println("Module workflows are executed via the clio-run-module script.")
			fmt.Println()
			fmt.Println("Run this command:")
			if setup.IsTermux() {
				fmt.Println("  clio-run-module termux_setup setup")
			} else {
				fmt.Println("  clio-run-module <module_id> [flow_name]")
				fmt.Println()
				fmt.Println("Example: clio-run-module my_workflow setup")
			}
			fmt.Println()
			fmt.Println("(Run 'sync' first if you haven't already)")
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
		fmt.Println("  3) Search for another command")
		fmt.Println("  0) Cancel")
		fmt.Println("")
		fmt.Print("Choice [1-3, 0]: ")

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

	// Extract base command (first word)
	baseCmd := strings.Fields(res.Command)[0]

	// Provide concise, practical examples (max 3-4 per command)
	switch baseCmd {
	case "ls":
		fmt.Println("\nCommon usage:")
		fmt.Println("  ls -la       - All files with details (most common)")
		fmt.Println("  ls -lh       - Human-readable file sizes")
		fmt.Println("  ls -lt       - Sort by time (newest first)")

	case "find":
		fmt.Println("\nCommon usage:")
		fmt.Println("  find . -name \"*.txt\"        - Find all .txt files")
		fmt.Println("  find . -type f -size +100M  - Files larger than 100MB")
		fmt.Println("  find . -mtime -7            - Modified in last 7 days")

	case "grep":
		fmt.Println("\nCommon usage:")
		fmt.Println("  grep -r \"pattern\" .         - Recursive search")
		fmt.Println("  grep -i \"pattern\" file      - Case-insensitive")
		fmt.Println("  grep -n \"pattern\" file      - Show line numbers")

	case "tar":
		fmt.Println("\nCommon usage:")
		fmt.Println("  tar -xzvf file.tar.gz       - Extract .tar.gz")
		fmt.Println("  tar -czvf archive.tar.gz dir/ - Create archive")
		fmt.Println("\nFlags: -x=extract -c=create -z=gzip -v=verbose -f=file")

	case "chmod":
		fmt.Println("\nCommon usage:")
		fmt.Println("  chmod +x script.sh          - Make executable")
		fmt.Println("  chmod 755 file              - rwxr-xr-x")
		fmt.Println("  chmod 644 file              - rw-r--r--")
		fmt.Println("\nTip: 7=rwx, 6=rw, 5=rx, 4=r (Owner-Group-Other)")

	case "chown":
		fmt.Println("\nCommon usage:")
		fmt.Println("  chown user:group file       - Change owner and group")
		fmt.Println("  chown -R user:group dir/    - Recursive")

	case "cp":
		fmt.Println("\nCommon usage:")
		fmt.Println("  cp file1 file2              - Copy file")
		fmt.Println("  cp -r dir1/ dir2/           - Copy directory recursively")
		fmt.Println("  cp -i file dest             - Prompt before overwrite")

	case "mv":
		fmt.Println("\nCommon usage:")
		fmt.Println("  mv old new                  - Rename file")
		fmt.Println("  mv file dir/                - Move to directory")
		fmt.Println("  mv -i file dest             - Prompt before overwrite")

	case "rm":
		fmt.Println("\nCommon usage:")
		fmt.Println("  rm file                     - Delete file")
		fmt.Println("  rm -rf directory/           - Delete directory and contents")
		fmt.Println("\n⚠️  WARNING: rm -rf is permanent!")

	case "df":
		fmt.Println("\nCommon usage:")
		fmt.Println("  df -h                       - Human-readable sizes")
		fmt.Println("  df -h /                     - Space for root filesystem")

	case "free":
		fmt.Println("\nCommon usage:")
		fmt.Println("  free -h                     - Human-readable memory usage")
		fmt.Println("  free -s 5                   - Update every 5 seconds")

	case "ps":
		fmt.Println("\nCommon usage:")
		fmt.Println("  ps aux                      - All processes with details")
		fmt.Println("  ps aux | grep name          - Find specific process")

	case "top", "htop":
		fmt.Println("\nInteractive keys:")
		fmt.Println("  q = quit, k = kill process")
		fmt.Println("  M = sort by memory, P = sort by CPU")

	case "cat":
		fmt.Println("\nCommon usage:")
		fmt.Println("  cat file                    - Display contents")
		fmt.Println("  cat file1 file2             - Concatenate files")

	case "tail":
		fmt.Println("\nCommon usage:")
		fmt.Println("  tail -n 20 file             - Last 20 lines")
		fmt.Println("  tail -f logfile             - Follow file (watch updates)")

	case "head":
		fmt.Println("\nCommon usage:")
		fmt.Println("  head -n 20 file             - First 20 lines")

	case "wc":
		fmt.Println("\nCommon usage:")
		fmt.Println("  wc -l file                  - Count lines")
		fmt.Println("  wc -w file                  - Count words")

	case "mkdir":
		fmt.Println("\nCommon usage:")
		fmt.Println("  mkdir -p path/to/dir        - Create nested directories")

	case "wget":
		fmt.Println("\nCommon usage:")
		fmt.Println("  wget URL                    - Download file")
		fmt.Println("  wget -c URL                 - Resume download")

	case "curl":
		fmt.Println("\nCommon usage:")
		fmt.Println("  curl -O URL                 - Download file")
		fmt.Println("  curl ifconfig.me            - Check public IP")

	case "sed":
		fmt.Println("\nCommon usage:")
		fmt.Println("  sed 's/old/new/g' file      - Replace all occurrences")
		fmt.Println("  sed -i 's/old/new/g' file   - Edit file in-place")

	case "awk":
		fmt.Println("\nCommon usage:")
		fmt.Println("  awk '{print $1}' file       - Print first column")
		fmt.Println("  awk -F: '{print $1}' file   - Use : as delimiter")

	case "ssh":
		fmt.Println("\nCommon usage:")
		fmt.Println("  ssh user@host               - Connect to remote")
		fmt.Println("  ssh -p 2222 user@host       - Use specific port")

	case "scp":
		fmt.Println("\nCommon usage:")
		fmt.Println("  scp file user@host:/path    - Copy to remote")
		fmt.Println("  scp user@host:/path/file .  - Copy from remote")

	case "rsync":
		fmt.Println("\nCommon usage:")
		fmt.Println("  rsync -avz src/ dest/       - Sync with compression")
		fmt.Println("  rsync -av --delete src/ dest/ - Delete extra files")

	case "git":
		fmt.Println("\nCommon usage:")
		fmt.Println("  git status                  - Check status")
		fmt.Println("  git add file                - Stage changes")
		fmt.Println("  git commit -m \"msg\"         - Commit changes")

	case "unzip":
		fmt.Println("\nCommon usage:")
		fmt.Println("  unzip file.zip              - Extract zip")
		fmt.Println("  unzip -l file.zip           - List contents")

	case "zip":
		fmt.Println("\nCommon usage:")
		fmt.Println("  zip -r archive.zip dir/     - Compress directory")

	case "du":
		fmt.Println("\nCommon usage:")
		fmt.Println("  du -sh dir/                 - Directory size summary")
		fmt.Println("  du -h --max-depth=1         - Subdirectories size")

	case "ln":
		fmt.Println("\nCommon usage:")
		fmt.Println("  ln -s target linkname       - Create symbolic link")

	case "which":
		fmt.Println("\nCommon usage:")
		fmt.Println("  which command               - Show path to command")

	case "ping":
		fmt.Println("\nCommon usage:")
		fmt.Println("  ping -c 4 host              - Send 4 packets")
		fmt.Println("  ping google.com             - Test connectivity")

	case "kill":
		fmt.Println("\nCommon usage:")
		fmt.Println("  kill PID                    - Terminate process")
		fmt.Println("  kill -9 PID                 - Force kill")

	default:
		// For commands without specific examples, show generic help
		fmt.Println("\nTip: Use 'man " + baseCmd + "' for detailed documentation")
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
