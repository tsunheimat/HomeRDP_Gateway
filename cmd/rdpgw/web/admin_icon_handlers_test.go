package web

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
)

func TestAdminUploadGlobalIconStoresSelectsAndServesIcon(t *testing.T) {
	handler, iconStore := newAdminIconTestHandler(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("icon", "custom.ico")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	iconBytes := []byte{0, 0, 1, 0, 1, 0}
	if _, err := fileWriter.Write(iconBytes); err != nil {
		t.Fatalf("write icon upload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/icon", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminUploadIcon(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status code = %d, want %d; body=%q", rr.Code, http.StatusCreated, rr.Body.String())
	}

	active, err := iconStore.ActiveIcon()
	if err != nil {
		t.Fatalf("active icon: %v", err)
	}
	if active.OriginalName != "custom.ico" {
		t.Fatalf("original name = %q", active.OriginalName)
	}

	serveReq := httptest.NewRequest(http.MethodGet, "/assets/app-icon", nil)
	serveRR := httptest.NewRecorder()
	handler.ServeAppIcon(serveRR, serveReq)

	if serveRR.Code != http.StatusOK {
		t.Fatalf("serve status = %d, want %d", serveRR.Code, http.StatusOK)
	}
	if serveRR.Header().Get("Content-Type") != "image/x-icon" {
		t.Fatalf("content type = %q", serveRR.Header().Get("Content-Type"))
	}
	if !bytes.Equal(serveRR.Body.Bytes(), iconBytes) {
		t.Fatalf("served icon body mismatch")
	}
}

func TestAdminSelectGlobalIconSwitchesActiveIcon(t *testing.T) {
	handler, iconStore := newAdminIconTestHandler(t)

	first, err := iconStore.SaveIcon("first.ico", []byte{0, 0, 1, 0})
	if err != nil {
		t.Fatalf("save first icon: %v", err)
	}
	second, err := iconStore.SaveIcon("second.png", []byte{137, 80, 78, 71})
	if err != nil {
		t.Fatalf("save second icon: %v", err)
	}

	payload, err := json.Marshal(map[string]string{"id": second.ID})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/icon/active", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminSelectIcon(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d; body=%q", rr.Code, http.StatusOK, rr.Body.String())
	}
	active, err := iconStore.ActiveIcon()
	if err != nil {
		t.Fatalf("active icon: %v", err)
	}
	if active.ID != second.ID {
		t.Fatalf("active icon = %q, want %q", active.ID, second.ID)
	}
	if active.ID == first.ID {
		t.Fatalf("first icon should no longer be active")
	}
}

func TestAdminUploadGlobalIconRejectsUnsupportedFile(t *testing.T) {
	handler, _ := newAdminIconTestHandler(t)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	fileWriter, err := writer.CreateFormFile("icon", "notes.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := fileWriter.Write([]byte("not an icon")); err != nil {
		t.Fatalf("write icon upload: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/icon", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://example.com")
	req = withAdminIdentity(req)
	rr := httptest.NewRecorder()

	handler.HandleAdminUploadIcon(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func newAdminIconTestHandler(t *testing.T) (*Handler, dashboard.IconStore) {
	t.Helper()

	tmpDir := t.TempDir()
	entryStore, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}
	iconStore, err := dashboard.NewFileIconStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "icons"))
	if err != nil {
		t.Fatalf("new icon store: %v", err)
	}

	handler := (&Config{
		Hosts:                   []string{"fallback.internal:3389"},
		HostSelection:           "roundrobin",
		DashboardStore:          entryStore,
		DashboardIconStore:      iconStore,
		DashboardMaxUploadBytes: 1 << 20,
		AdminGroups:             []string{"rdpgw-admins"},
	}).NewHandler()

	return handler, iconStore
}
