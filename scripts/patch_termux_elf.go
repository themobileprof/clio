//go:build ignore

// Command patch_termux_elf raises the PT_TLS alignment of a Go linux/arm64 PIE.
//
// Go writes p_align 8. Once Android's linker loads the binary, arm64 Bionic
// rejects that on Android 10 and Android 16 alike:
// "TLS segment is underaligned: alignment is 8, needs to be at least 64".
package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	ptTLS    = 7
	minAlign = 64
	phdrSize = 56
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run scripts/patch_termux_elf.go binary")
		os.Exit(2)
	}
	if err := patchTLSAlign(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func patchTLSAlign(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	var ident [16]byte
	if _, err := f.ReadAt(ident[:], 0); err != nil {
		return err
	}
	if ident[0] != 0x7f || ident[1] != 'E' || ident[2] != 'L' || ident[3] != 'F' || ident[4] != 2 || ident[5] != 1 {
		return fmt.Errorf("%s: not a little-endian ELF64", path)
	}

	hdr := make([]byte, 64)
	if _, err := f.ReadAt(hdr, 0); err != nil {
		return err
	}
	phoff := binary.LittleEndian.Uint64(hdr[32:40])
	phentsize := binary.LittleEndian.Uint16(hdr[54:56])
	phnum := binary.LittleEndian.Uint16(hdr[56:58])
	if phentsize != phdrSize {
		return fmt.Errorf("%s: program header size %d, want %d", path, phentsize, phdrSize)
	}

	ph := make([]byte, phdrSize)
	for i := 0; i < int(phnum); i++ {
		off := int64(phoff) + int64(i)*phdrSize
		if _, err := f.ReadAt(ph, off); err != nil {
			return err
		}
		if binary.LittleEndian.Uint32(ph[0:4]) != ptTLS {
			continue
		}
		align := binary.LittleEndian.Uint64(ph[48:56])
		if align >= minAlign {
			fmt.Printf("%s: PT_TLS align already %d\n", path, align)
			return nil
		}
		binary.LittleEndian.PutUint64(ph[48:56], minAlign)
		if _, err := f.WriteAt(ph, off); err != nil {
			return err
		}
		fmt.Printf("%s: PT_TLS align %d -> %d\n", path, align, minAlign)
		return nil
	}
	return fmt.Errorf("%s: no PT_TLS header", path)
}
