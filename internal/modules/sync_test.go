package modules

import "testing"

func TestLatestChangedKeepsNewestUpload(t *testing.T) {
	mods := []ChangedModule{
		{ID: "termux_setup", Version: "1.0.0", UpdatedAt: "2026-10-26T17:49:08Z", ChecksumSHA256: "old"},
		{ID: "neovim_setup", Version: "1.3.0", UpdatedAt: "2026-10-26T20:39:21Z", ChecksumSHA256: "nvim"},
		{ID: "termux_setup", Version: "1.2.1", UpdatedAt: "2026-11-02T12:13:48Z", ChecksumSHA256: "new"},
	}
	got := latestChanged(mods)
	if len(got) != 2 {
		t.Fatalf("got %d modules, want 2", len(got))
	}
	if got[0].ID != "termux_setup" || got[0].Version != "1.2.1" || got[0].ChecksumSHA256 != "new" {
		t.Fatalf("termux row = %+v", got[0])
	}
	if got[1].ID != "neovim_setup" {
		t.Fatalf("second row = %+v", got[1])
	}
}
