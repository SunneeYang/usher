package player

import "testing"

func TestInstalledOptionsAlwaysIncludesDefaultAndCustom(t *testing.T) {
	opts := InstalledOptions()
	if len(opts) < 2 {
		t.Fatalf("opts = %#v", opts)
	}
	if opts[0].ID != "default" {
		t.Fatalf("first option = %q, want default", opts[0].ID)
	}
	last := opts[len(opts)-1]
	if last.ID != "custom" {
		t.Fatalf("last option = %q, want custom", last.ID)
	}
}

func TestSanitizeSelectionUnknown(t *testing.T) {
	id, app := SanitizeSelection("not-a-player", "")
	if id != "default" || app != "" {
		t.Fatalf("got %q %q", id, app)
	}
}

func TestIsAvailableDefault(t *testing.T) {
	if !IsAvailable("default", "") {
		t.Fatal("default should always be available")
	}
}

func TestIsAvailableCustomMissingPath(t *testing.T) {
	if IsAvailable("custom", "") {
		t.Fatal("custom without path should be unavailable")
	}
}
