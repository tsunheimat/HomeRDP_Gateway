package web

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
		"icon":          dashboard.EntryIconTerminal,
		"allowedGroups": []string{"homelab-users", "admins"},
		"host":          "lab.internal:3389",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/host", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
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
	if stored.Icon != payload["icon"] {
		t.Fatalf("stored icon = %q, want %q", stored.Icon, payload["icon"])
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

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/template", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.com")
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

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/entries/template-1", nil)
	req.Header.Set("Origin", "http://example.com")
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

func TestAdminDeleteEntryMissingReturnsNotFound(t *testing.T) {
	handler, _ := newAdminTestHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/entries/missing", nil)
	req.Header.Set("Origin", "http://example.com")
	req = mux.SetURLVars(req, map[string]string{"id": "missing"})
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminDeleteEntry(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestAdminListEntries(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	entry := dashboard.Entry{
		ID:            "host-1",
		Type:          dashboard.EntryTypeHost,
		Name:          "Host 1",
		Icon:          dashboard.EntryIconBrowser,
		AllowedGroups: []string{"homelab-users"},
		Enabled:       true,
		Host:          "host.internal:3389",
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/entries", nil)
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminListEntries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []dashboard.Entry
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 1 || entries[0].ID != entry.ID {
		t.Fatalf("unexpected entries payload: %+v", entries)
	}
	if entries[0].Icon != dashboard.EntryIconBrowser {
		t.Fatalf("response icon = %q, want %q", entries[0].Icon, dashboard.EntryIconBrowser)
	}
}

func TestAdminListEntriesDoesNotExposeUploadedTemplatePath(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	uploadPath, err := store.SaveUpload(strings.NewReader("full address:s:host.internal:3389\r\n"))
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

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/entries", nil)
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminListEntries(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var entries []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &entries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	if _, ok := entries[0]["uploadedTemplatePath"]; ok {
		t.Fatalf("unexpected uploadedTemplatePath in response: %+v", entries[0])
	}
}

func TestAdminUpdateEntry(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	entry := dashboard.Entry{
		ID:            "host-1",
		Type:          dashboard.EntryTypeHost,
		Name:          "Host 1",
		AllowedGroups: []string{"homelab-users"},
		Enabled:       true,
		Host:          "host.internal:3389",
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	payload := map[string]interface{}{
		"name":          "Host 2",
		"icon":          dashboard.EntryIconDatabase,
		"allowedGroups": []string{"admins"},
		"host":          "new-host.internal:3389",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/entries/host-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req = mux.SetURLVars(req, map[string]string{"id": entry.ID})
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminUpdateEntry(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	updated, err := store.Get(entry.ID)
	if err != nil {
		t.Fatalf("get updated entry: %v", err)
	}
	if updated.Name != "Host 2" || updated.Host != "new-host.internal:3389" {
		t.Fatalf("unexpected updated entry: %+v", updated)
	}
	if updated.Icon != dashboard.EntryIconDatabase {
		t.Fatalf("updated icon = %q, want %q", updated.Icon, dashboard.EntryIconDatabase)
	}
}

func TestAdminCreateTemplateEntry(t *testing.T) {
	handler, store := newAdminTestHandler(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("name", "Good Template"); err != nil {
		t.Fatalf("write form field name: %v", err)
	}
	if err := writer.WriteField("icon", dashboard.EntryIconWord); err != nil {
		t.Fatalf("write form field icon: %v", err)
	}
	if err := writer.WriteField("allowedGroups", "homelab-users, admins"); err != nil {
		t.Fatalf("write form field allowedGroups: %v", err)
	}
	if err := writer.WriteField("targetHostOverride", "app.internal:3389"); err != nil {
		t.Fatalf("write form field targetHostOverride: %v", err)
	}
	fileWriter, err := writer.CreateFormFile("template", "good.rdp")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("full address:s:template.internal:3389\r\n")); err != nil {
		t.Fatalf("write upload content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/template", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateTemplateEntry(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusCreated)
	}

	var created map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := created["uploadedTemplatePath"]; ok {
		t.Fatalf("unexpected uploadedTemplatePath in response: %+v", created)
	}
	entries, err := store.List()
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1", len(entries))
	}
	if entries[0].UploadedTemplatePath == "" {
		t.Fatalf("expected stored uploaded template path to be set")
	}
	if entries[0].Icon != dashboard.EntryIconWord {
		t.Fatalf("stored icon = %q, want %q", entries[0].Icon, dashboard.EntryIconWord)
	}
	if _, err := os.Stat(store.ResolveUpload(entries[0].UploadedTemplatePath)); err != nil {
		t.Fatalf("expected uploaded file to exist: %v", err)
	}
}

func TestAdminCreateAuthUserWritesHelperConfig(t *testing.T) {
	handler, authStore, helperConfigPath := newAdminAuthTestHandler(t)

	body := strings.NewReader(`{"username":"alice","password":"secret","enabled":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth-users", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateAuthUser(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusCreated)
	}

	stored, err := authStore.Get("alice")
	if err != nil {
		t.Fatalf("get auth user: %v", err)
	}
	if stored.Password != "secret" || !stored.Enabled {
		t.Fatalf("unexpected stored auth user: %+v", stored)
	}

	data, err := os.ReadFile(helperConfigPath)
	if err != nil {
		t.Fatalf("read helper config: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "alice") || !strings.Contains(text, "secret") {
		t.Fatalf("expected helper config to contain created user, got %q", text)
	}
}

func TestAdminListAuthUsersDoesNotExposePassword(t *testing.T) {
	handler, authStore, _ := newAdminAuthTestHandler(t)
	if err := authStore.Put(dashboard.AuthUser{Username: "alice", Password: "secret", Enabled: true}); err != nil {
		t.Fatalf("put auth user: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth-users", nil)
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminListAuthUsers(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var payload []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("auth user count = %d, want 1", len(payload))
	}
	if _, ok := payload[0]["password"]; ok {
		t.Fatalf("password should not be exposed in list response: %+v", payload[0])
	}
}

func TestAdminUpdateAuthUserDisablesHelperAccess(t *testing.T) {
	handler, authStore, helperConfigPath := newAdminAuthTestHandler(t)
	if err := authStore.Put(dashboard.AuthUser{Username: "alice", Password: "secret", Enabled: true}); err != nil {
		t.Fatalf("put auth user: %v", err)
	}
	if err := dashboard.WriteAuthHelperConfig(helperConfigPath, []dashboard.AuthUser{{Username: "alice", Password: "secret", Enabled: true}}); err != nil {
		t.Fatalf("seed helper config: %v", err)
	}

	body := strings.NewReader(`{"password":"newsecret","enabled":false}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/auth-users/alice", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req = mux.SetURLVars(req, map[string]string{"username": "alice"})
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminUpdateAuthUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	stored, err := authStore.Get("alice")
	if err != nil {
		t.Fatalf("get auth user: %v", err)
	}
	if stored.Password != "newsecret" || stored.Enabled {
		t.Fatalf("unexpected stored auth user: %+v", stored)
	}

	data, err := os.ReadFile(helperConfigPath)
	if err != nil {
		t.Fatalf("read helper config: %v", err)
	}
	if strings.Contains(string(data), "alice") {
		t.Fatalf("disabled auth user should be omitted from helper config, got %q", string(data))
	}
}

func TestAdminDeleteAuthUserRemovesHelperEntry(t *testing.T) {
	handler, authStore, helperConfigPath := newAdminAuthTestHandler(t)
	if err := authStore.Put(dashboard.AuthUser{Username: "alice", Password: "secret", Enabled: true}); err != nil {
		t.Fatalf("put auth user: %v", err)
	}
	if err := dashboard.WriteAuthHelperConfig(helperConfigPath, []dashboard.AuthUser{{Username: "alice", Password: "secret", Enabled: true}}); err != nil {
		t.Fatalf("seed helper config: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/auth-users/alice", nil)
	req.Header.Set("Origin", "http://example.com")
	req = mux.SetURLVars(req, map[string]string{"username": "alice"})
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminDeleteAuthUser(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if _, err := authStore.Get("alice"); err != dashboard.ErrAuthUserNotFound {
		t.Fatalf("expected ErrAuthUserNotFound, got %v", err)
	}

	data, err := os.ReadFile(helperConfigPath)
	if err != nil {
		t.Fatalf("read helper config: %v", err)
	}
	if strings.Contains(string(data), "alice") {
		t.Fatalf("deleted auth user should be removed from helper config, got %q", string(data))
	}
}

func TestAdminOnlyRequiresAdmin(t *testing.T) {
	handler, _ := newAdminTestHandler(t)

	next := handler.AdminOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/entries", nil)
	id := identity.NewUser()
	id.SetUserName("user@example.com")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"homelab-users"})
	req = identity.AddToRequestCtx(id, req)
	rr := httptest.NewRecorder()

	next.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAdminCreateHostEntryValidationReturnsBadRequest(t *testing.T) {
	handler, _ := newAdminTestHandler(t)

	payload := map[string]interface{}{
		"name":          "",
		"description":   "Missing name",
		"allowedGroups": []string{"homelab-users"},
		"host":          "lab.internal:3389",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/host", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateHostEntry(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestAdminCreateHostEntryRejectsCrossOriginRequest(t *testing.T) {
	handler, _ := newAdminTestHandler(t)

	payload := `{"name":"Lab Host","allowedGroups":["homelab-users"],"host":"lab.internal:3389"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/host", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateHostEntry(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAdminCreateTemplateEntryRejectsCrossOriginRequest(t *testing.T) {
	handler, _ := newAdminTestHandler(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("name", "Good Template"); err != nil {
		t.Fatalf("write form field name: %v", err)
	}
	if err := writer.WriteField("allowedGroups", "homelab-users"); err != nil {
		t.Fatalf("write form field allowedGroups: %v", err)
	}
	fileWriter, err := writer.CreateFormFile("template", "good.rdp")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("full address:s:template.internal:3389\r\n")); err != nil {
		t.Fatalf("write upload content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/template", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "https://evil.example")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateTemplateEntry(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAdminCreateHostEntryMapsStoreValidationErrorsToBadRequest(t *testing.T) {
	putErr := (dashboard.Entry{}).Validate()
	if putErr == nil {
		t.Fatal("expected validation error")
	}

	handler := (&Config{
		Hosts:          []string{"fallback.internal:3389"},
		HostSelection:  "roundrobin",
		DashboardStore: stubAdminStore{putErr: putErr},
		AdminGroups:    []string{"rdpgw-admins"},
	}).NewHandler()

	payload := `{"name":"Lab Host","allowedGroups":["homelab-users"],"host":"lab.internal:3389"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/entries/host", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminCreateHostEntry(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusBadRequest)
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

func newAdminAuthTestHandler(t *testing.T) (*Handler, dashboard.AuthUserStore, string) {
	t.Helper()

	tmpDir := t.TempDir()
	entryStore, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}
	authUserStore, err := dashboard.NewFileAuthUserStore(filepath.Join(tmpDir, "catalog", "auth-users.json"))
	if err != nil {
		t.Fatalf("new auth user store: %v", err)
	}
	helperConfigPath := filepath.Join(tmpDir, "catalog", "rdpgw-auth.yaml")

	handler := (&Config{
		Hosts:                   []string{"fallback.internal:3389"},
		HostSelection:           "roundrobin",
		DashboardStore:          entryStore,
		DashboardAuthUserStore:  authUserStore,
		AuthHelperConfigPath:    helperConfigPath,
		DashboardMaxUploadBytes: 1 << 20,
		AdminGroups:             []string{"rdpgw-admins"},
	}).NewHandler()

	return handler, authUserStore, helperConfigPath
}

func withAdminIdentity(req *http.Request) *http.Request {
	id := identity.NewUser()
	id.SetUserName("admin@example.com")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"rdpgw-admins"})
	return identity.AddToRequestCtx(id, req)
}

type stubAdminStore struct {
	putErr error
}

func (s stubAdminStore) List() ([]dashboard.Entry, error) {
	return nil, nil
}

func (s stubAdminStore) Get(id string) (dashboard.Entry, error) {
	return dashboard.Entry{}, dashboard.ErrEntryNotFound
}

func (s stubAdminStore) Put(entry dashboard.Entry) error {
	return s.putErr
}

func (s stubAdminStore) Delete(id string) error {
	return nil
}

func (s stubAdminStore) SaveUpload(src io.Reader) (string, error) {
	return "", nil
}

func (s stubAdminStore) ResolveUpload(path string) string {
	return path
}
