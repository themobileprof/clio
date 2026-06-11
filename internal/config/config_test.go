package config

import (
	"clio/internal/setup"
	"os"
	"testing"
)

func TestEffectiveProfileAutoOnTermux(t *testing.T) {
	ResetCache()
	t.Setenv("TERMUX_VERSION", "test")
	defer t.Setenv("TERMUX_VERSION", "")

	if !setup.IsTermux() {
		t.Fatal("expected Termux detection")
	}
	if EffectiveProfile() != ProfileLite {
		t.Fatalf("expected lite profile on Termux, got %q", EffectiveProfile())
	}
}

func TestParseMemoryLimit(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"48MiB", 48 << 20},
		{"512KiB", 512 << 10},
		{"", 0},
	}
	for _, c := range cases {
		got := parseMemoryLimit(c.in)
		if got != c.want {
			t.Errorf("parseMemoryLimit(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestGetMemoryLimitLiteDefault(t *testing.T) {
	ResetCache()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TERMUX_VERSION", "test")
	defer os.Unsetenv("TERMUX_VERSION")

	if got := GetMemoryLimit(); got != 48<<20 {
		t.Fatalf("GetMemoryLimit() = %d, want %d", got, 48<<20)
	}
}
