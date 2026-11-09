package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStripLinkerArgRemovesInsertedBinaryPath(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "clio.bin")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := stripLinkerArg(
		[]string{"clio.bin", bin, "--print-module", "termux_setup"},
		bin,
		true,
	)
	want := []string{"clio.bin", "--print-module", "termux_setup"}
	if stringsJoin(got) != stringsJoin(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestStripLinkerArgLeavesRealArguments(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "clio.bin")
	other := filepath.Join(dir, "notes.txt")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := [][]string{
		{"clio.bin", "--print-module", "termux_setup"},
		{"clio.bin", other, "--print-module", "termux_setup"},
		{"clio.bin"},
	}
	for _, args := range cases {
		got := stripLinkerArg(append([]string(nil), args...), bin, true)
		if stringsJoin(got) != stringsJoin(args) {
			t.Fatalf("stripLinkerArg(%q) = %q", args, got)
		}
	}

	got := stripLinkerArg([]string{"clio.bin", bin, "sync"}, bin, false)
	if stringsJoin(got) != "clio.bin "+bin+" sync" {
		t.Fatalf("disabled strip changed args to %q", got)
	}
}

func stringsJoin(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
