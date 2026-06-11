//go:build ignore

package main

import (
	"fmt"

	"clio/internal/intent"
	"clio/internal/layer1"
)

func main() {
	queries := []string{
		"list running processes", "memory usage", "what files are here",
		"show end of log", "view current directory", "unzip file",
		"change file permissions", "chek disk space", "where am I",
		"list files", "check disk space", "extract zip archive",
	}
	for _, q := range queries {
		v, n := layer1.ParseIntent(q)
		r, err := intent.Detect(q)
		cmd, src := "ERR", ""
		if err == nil {
			cmd, src = r.Command, r.Source
		}
		fmt.Printf("%-28s verb=%-8s noun=%-12s => %-22s [%s]\n", q, v, n, cmd, src)
	}
}
