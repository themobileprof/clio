package main

import (
	"os"
	"path/filepath"
	"strings"
)

const termuxPrefix = "/data/data/com.termux/files/"

// stripTermuxLinkerArg drops the extra argv entry Termux inserts.
//
// Android 10+ will not exec a file under an app's private directory, so
// Termux starts the program with /system/bin/linker64 and places this
// binary's absolute path in os.Args[1]. Later arguments, including
// --print-module, shift one place to the right.
func stripTermuxLinkerArg(args []string) []string {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	return stripLinkerArg(args, exe, runningUnderTermux(exe))
}

func runningUnderTermux(exe string) bool {
	if os.Getenv("TERMUX_VERSION") != "" {
		return true
	}
	return strings.HasPrefix(exe, termuxPrefix)
}

func stripLinkerArg(args []string, exe string, underTermux bool) []string {
	if !underTermux || exe == "" || len(args) < 2 || !filepath.IsAbs(args[1]) {
		return args
	}
	if !sameFile(args[1], exe) {
		return args
	}
	return append(args[:1:1], args[2:]...)
}

func sameFile(a, b string) bool {
	ia, ea := os.Stat(a)
	ib, eb := os.Stat(b)
	if ea != nil || eb != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return os.SameFile(ia, ib)
}
