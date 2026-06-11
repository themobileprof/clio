package main

import (
	"bufio"
	"clio/internal/config"
	"clio/internal/intent"
	"clio/internal/repl"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

func main() {
	applyMemoryProfile()

	if !isInteractive() {
		runPipeMode()
		return
	}
	repl.Run()
}

func applyMemoryProfile() {
	if limit := config.GetMemoryLimit(); limit > 0 {
		debug.SetMemoryLimit(limit)
	}
	if config.IsLiteProfile() {
		// Single core is typical on low-end phones; reduces concurrent alloc pressure
		runtime.GOMAXPROCS(1)
	}
}

func isInteractive() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return true
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func runPipeMode() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 256), 4096)
	for scanner.Scan() {
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		result, err := intent.Detect(input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "no match: %s\n", input)
			os.Exit(1)
		}
		fmt.Println(result.Command)
		return
	}
}
