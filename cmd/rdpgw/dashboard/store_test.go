package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEntryValidate(t *testing.T) {
	t.Parallel()

	validHost := Entry{
		ID:            "host-1",
		Type:          EntryTypeHost,
		Name:          "Primary host",
		AllowedGroups: []string{"admins"},
		Enabled:       true,
		Host:          "rdp.internal:3389",
	}
	if err := validHost.Validate(); err != nil {
		t.Fatalf("expected valid host entry, got error: %v", err)
	}

	missingGroups := validHost
	missingGroups.AllowedGroups = nil
	if err := missingGroups.Validate(); err == nil {
		t.Fatalf("expected missing groups to fail validation")
	}
}

func TestVisibleEntries(t *testing.T) {
	t.Parallel()

	entries := []Entry{
		{
			ID:            "host-visible",
			Type:          EntryTypeHost,
			Name:          "Visible",
			AllowedGroups: []string{"admins"},
			Enabled:       true,
			Host:          "server-a:3389",
		},
		{
			ID:            "host-disabled",
			Type:          EntryTypeHost,
			Name:          "Disabled",
			AllowedGroups: []string{"admins"},
			Enabled:       false,
			Host:          "server-b:3389",
		},
		{
			ID:            "host-other-group",
			Type:          EntryTypeHost,
			Name:          "Other Group",
			AllowedGroups: []string{"ops"},
			Enabled:       true,
			Host:          "server-c:3389",
		},
	}

	visible := VisibleEntries(entries, []string{" admins "})
	if len(visible) != 1 {
		t.Fatalf("expected exactly one visible entry, got %d", len(visible))
	}
	if visible[0].ID != "host-visible" {
		t.Fatalf("expected host-visible, got %s", visible[0].ID)
	}
}

func TestFileStoreUpsertAndDelete(t *testing.T) {
	t.Parallel()

	baseDir := t.TempDir()
	store, err := NewFileStore(filepath.Join(baseDir, "catalog"), filepath.Join(baseDir, "uploads"))
	if err != nil {
		t.Fatalf("new file store: %v", err)
	}

	uploadPath, err := store.SaveUpload(strings.NewReader("full address:s:server-template"))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}
	if filepath.Ext(uploadPath) != ".rdp" {
		t.Fatalf("expected .rdp upload path, got %q", uploadPath)
	}
	if _, err := os.Stat(store.ResolveUpload(uploadPath)); err != nil {
		t.Fatalf("expected upload file to exist: %v", err)
	}

	entry := Entry{
		ID:                   "template-1",
		Type:                 EntryTypeTemplate,
		Name:                 "Template",
		AllowedGroups:        []string{"admins"},
		Enabled:              true,
		UploadedTemplatePath: uploadPath,
		TargetHostOverride:   "target.internal",
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	loadedStore, err := NewFileStore(filepath.Join(baseDir, "catalog"), filepath.Join(baseDir, "uploads"))
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}

	loaded, err := loadedStore.Get(entry.ID)
	if err != nil {
		t.Fatalf("get entry: %v", err)
	}
	if loaded.UploadedTemplatePath != uploadPath {
		t.Fatalf("expected uploaded path %q, got %q", uploadPath, loaded.UploadedTemplatePath)
	}

	listed, err := loadedStore.List()
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one entry, got %d", len(listed))
	}

	if err := loadedStore.Delete(entry.ID); err != nil {
		t.Fatalf("delete entry: %v", err)
	}
	if _, err := os.Stat(loadedStore.ResolveUpload(uploadPath)); !os.IsNotExist(err) {
		t.Fatalf("expected upload file to be removed, stat err=%v", err)
	}

	afterDelete, err := loadedStore.List()
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected zero entries after delete, got %d", len(afterDelete))
	}
}
