package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileIconStoreSavesListsAndSelectsActiveIcon(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileIconStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "icons"))
	if err != nil {
		t.Fatalf("new icon store: %v", err)
	}

	first, err := store.SaveIcon("first.ico", []byte{0, 0, 1, 0})
	if err != nil {
		t.Fatalf("save first icon: %v", err)
	}
	second, err := store.SaveIcon("second.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`))
	if err != nil {
		t.Fatalf("save second icon: %v", err)
	}

	icons, err := store.ListIcons()
	if err != nil {
		t.Fatalf("list icons: %v", err)
	}
	if len(icons) != 2 {
		t.Fatalf("icon count = %d, want 2", len(icons))
	}
	if !icons[0].Active {
		t.Fatalf("first uploaded icon should become active by default")
	}

	if err := store.SetActiveIcon(second.ID); err != nil {
		t.Fatalf("set active icon: %v", err)
	}
	active, err := store.ActiveIcon()
	if err != nil {
		t.Fatalf("active icon: %v", err)
	}
	if active.ID != second.ID {
		t.Fatalf("active icon id = %q, want %q", active.ID, second.ID)
	}
	if active.ContentType != "image/svg+xml" {
		t.Fatalf("active icon content type = %q", active.ContentType)
	}
	if _, err := os.Stat(store.ResolveIcon(first.Filename)); err != nil {
		t.Fatalf("first icon file should exist: %v", err)
	}
}

func TestFileIconStoreRejectsUnsupportedIconExtension(t *testing.T) {
	store, err := NewFileIconStore(t.TempDir(), filepath.Join(t.TempDir(), "icons"))
	if err != nil {
		t.Fatalf("new icon store: %v", err)
	}

	if _, err := store.SaveIcon("notes.txt", []byte("not an icon")); err == nil {
		t.Fatalf("expected unsupported icon extension to fail")
	}
}
