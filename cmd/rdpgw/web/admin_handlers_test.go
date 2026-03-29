package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/gorilla/mux"
)

func TestAdminCreateHostEntry(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	payload := map[string]interface{}{
		"name":          "Lab Host",
		"description":   "Primary workstation",
		"allowedGroups": []string{"homelab-users", "admins"},
		"host":          "lab.internal:3389",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/api/dashboard/entries/host", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateHostEntry(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusCreated)
	}

	var got dashboard.Entry
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response entry: %v", err)
	}

	if got.ID == "" {
		t.Fatalf("expected generated id, got empty string")
	}
	if got.Type != dashboard.EntryTypeHost {
		t.Fatalf("entry type = %q, want %q", got.Type, dashboard.EntryTypeHost)
	}
	if !got.Enabled {
		t.Fatalf("expected created entry to be enabled")
	}

	stored, err := store.Get(got.ID)
	if err != nil {
		t.Fatalf("load stored entry: %v", err)
	}

	if stored.Host != payload["host"] {
		t.Fatalf("stored host = %q, want %q", stored.Host, payload["host"])
	}
	if stored.Name != payload["name"] {
		t.Fatalf("stored name = %q, want %q", stored.Name, payload["name"])
	}
}

func TestAdminCreateTemplateEntryRejectsBadUpload(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", "Bad Template"); err != nil {
		t.Fatalf("write form field name: %v", err)
	}
	if err := writer.WriteField("allowedGroups", "homelab-users"); err != nil {
		t.Fatalf("write form field allowedGroups: %v", err)
	}

	fileWriter, err := writer.CreateFormFile("template", "broken.rdp")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("this is not a valid rdp file")); err != nil {
		t.Fatalf("write upload content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/api/dashboard/entries/template", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateTemplateEntry(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	entries, err := store.List()
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected no persisted entries, got %d", len(entries))
	}
}

func TestAdminDeleteEntryRemovesTemplate(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	uploadPath, err := store.SaveUpload(strings.NewReader("full address:s:template.internal:3389\r\n"))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}

	entry := dashboard.Entry{
		ID:                   "template-1",
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 "Template",
		AllowedGroups:        []string{"homelab-users"},
		Enabled:              true,
		UploadedTemplatePath: uploadPath,
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/admin/api/dashboard/entries/template-1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": entry.ID})
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminDeleteEntry(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNoContent)
	}

	_, err = store.Get(entry.ID)
	if !errors.Is(err, dashboard.ErrEntryNotFound) {
		t.Fatalf("expected entry to be deleted, got err=%v", err)
	}

	uploadFullPath := store.ResolveUpload(uploadPath)
	if _, err := os.Stat(uploadFullPath); !os.IsNotExist(err) {
		t.Fatalf("expected uploaded template to be deleted, stat err=%v", err)
	}
}

func newAdminTestHandler(t *testing.T) (*Handler, dashboard.Store) {
	t.Helper()

	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	handler := (&Config{
		Hosts:                   []string{"fallback.internal:3389"},
		HostSelection:           "roundrobin",
		DashboardStore:          store,
		DashboardMaxUploadBytes: 1 << 20,
		AdminGroups:             []string{"rdpgw-admins"},
	}).NewHandler()

	return handler, store
}

func withAdminIdentity(req *http.Request) *http.Request {
	id := identity.NewUser()
	id.SetUserName("admin@example.com")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"rdpgw-admins"})
	return identity.AddToRequestCtx(id, req)
}
