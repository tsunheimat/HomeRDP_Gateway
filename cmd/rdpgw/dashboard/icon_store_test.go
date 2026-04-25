package dashboard

import (
	"encoding/json"
	"errors"
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
	second, err := store.SaveIcon("second.png", []byte{137, 80, 78, 71})
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
	if active.ContentType != "image/png" {
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

func TestFileIconStoreRejectsSVGIcon(t *testing.T) {
	store, err := NewFileIconStore(t.TempDir(), filepath.Join(t.TempDir(), "icons"))
	if err != nil {
		t.Fatalf("new icon store: %v", err)
	}

	for _, filename := range []string{"custom.svg", "custom.SVG", "custom.png.svg"} {
		t.Run(filename, func(t *testing.T) {
			if _, err := store.SaveIcon(filename, []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`)); err == nil {
				t.Fatalf("expected svg icon upload %q to fail", filename)
			}
		})
	}
}

func TestFileIconStoreIgnoresLegacyStoredSVGIcons(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := NewFileIconStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "icons"))
	if err != nil {
		t.Fatalf("new icon store: %v", err)
	}

	legacy := []Icon{{
		ID:           "legacy-svg",
		Filename:     "legacy.svg",
		OriginalName: "legacy.svg",
		ContentType:  "image/svg+xml",
		Active:       true,
	}}
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy metadata: %v", err)
	}
	if err := os.WriteFile(store.metadataPath, data, 0o600); err != nil {
		t.Fatalf("write legacy metadata: %v", err)
	}

	icons, err := store.ListIcons()
	if err != nil {
		t.Fatalf("list icons: %v", err)
	}
	if len(icons) != 0 {
		t.Fatalf("legacy unsafe icons should be filtered, got %+v", icons)
	}
	if _, err := store.ActiveIcon(); !errors.Is(err, ErrIconNotFound) {
		t.Fatalf("active icon error = %v, want %v", err, ErrIconNotFound)
	}
}

func TestFileIconStoreAcceptsSupportedSafeIconExtensions(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		data        []byte
		contentType string
	}{
		{name: "ico", filename: "custom.ico", data: []byte{0, 0, 1, 0}, contentType: "image/x-icon"},
		{name: "icon", filename: "custom.icon", data: []byte{0, 0, 1, 0}, contentType: "image/x-icon"},
		{name: "png", filename: "custom.png", data: []byte{137, 80, 78, 71}, contentType: "image/png"},
		{name: "jpg", filename: "custom.jpg", data: []byte{0xff, 0xd8, 0xff, 0xdb}, contentType: "image/jpeg"},
		{name: "jpeg", filename: "custom.jpeg", data: []byte{0xff, 0xd8, 0xff, 0xdb}, contentType: "image/jpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			store, err := NewFileIconStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "icons"))
			if err != nil {
				t.Fatalf("new icon store: %v", err)
			}

			icon, err := store.SaveIcon(tt.filename, tt.data)
			if err != nil {
				t.Fatalf("save icon: %v", err)
			}
			if icon.ContentType != tt.contentType {
				t.Fatalf("content type = %q, want %q", icon.ContentType, tt.contentType)
			}
			if _, err := os.Stat(store.ResolveIcon(icon.Filename)); err != nil {
				t.Fatalf("icon file should exist: %v", err)
			}
		})
	}
}
