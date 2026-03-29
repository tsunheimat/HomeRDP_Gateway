package web

import (
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/rdp"
	"github.com/gorilla/mux"
)

func TestHandleEntryDownloadHostEntry(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	entry := dashboard.Entry{
		ID:            "lab-win11.rdp",
		Type:          dashboard.EntryTypeHost,
		Name:          "Lab Win11",
		AllowedGroups: []string{"homelab-users"},
		Enabled:       true,
		Host:          "lab-{{ preferred_username }}.internal:3389",
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	handler := (&Config{
		Hosts:             []string{"fallback.internal:3389"},
		HostSelection:     "roundrobin",
		DashboardStore:    store,
		PAATokenGenerator: paaTokenMock,
		GatewayAddress:    gatewayAddress,
	}).NewHandler()

	req := httptest.NewRequest("GET", "/connect/entries/lab-win11.rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "lab-win11.rdp"})

	id := identity.NewUser()
	id.SetUserName("alice")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"homelab-users"})
	req = identity.AddToRequestCtx(id, req)

	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}

	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "lab-alice.internal:3389" {
		t.Fatalf("full address = %q, want %q", data["full address"], "lab-alice.internal:3389")
	}
	if data["gatewayhostname"] != gatewayAddress.Host {
		t.Fatalf("gatewayhostname = %q, want %q", data["gatewayhostname"], gatewayAddress.Host)
	}
}

func TestHandleEntryDownloadTemplateEntryPreservesRemoteApp(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	template := strings.Join([]string{
		"full address:s:template-target.internal:3389",
		"remoteapplicationmode:i:1",
		"remoteapplicationprogram:s:||notepad",
		"remoteapplicationname:s:Notepad",
	}, rdp.CRLF) + rdp.CRLF
	uploadPath, err := store.SaveUpload(strings.NewReader(template))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}

	entry := dashboard.Entry{
		ID:                   "office-app.rdp",
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 "Office App",
		AllowedGroups:        []string{"office-users"},
		Enabled:              true,
		UploadedTemplatePath: uploadPath,
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	handler := (&Config{
		Hosts:             []string{"fallback.internal:3389"},
		HostSelection:     "roundrobin",
		DashboardStore:    store,
		PAATokenGenerator: paaTokenMock,
		GatewayAddress:    gatewayAddress,
	}).NewHandler()

	req := httptest.NewRequest("GET", "/connect/entries/office-app.rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "office-app.rdp"})

	id := identity.NewUser()
	id.SetUserName("bob")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"office-users"})
	req = identity.AddToRequestCtx(id, req)

	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}

	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "template-target.internal:3389" {
		t.Fatalf("full address = %q, want %q", data["full address"], "template-target.internal:3389")
	}
	if data["remoteapplicationmode"] != "1" {
		t.Fatalf("remoteapplicationmode = %q, want %q", data["remoteapplicationmode"], "1")
	}
	if data["remoteapplicationprogram"] != "||notepad" {
		t.Fatalf("remoteapplicationprogram = %q, want %q", data["remoteapplicationprogram"], "||notepad")
	}
	if data["remoteapplicationname"] != "Notepad" {
		t.Fatalf("remoteapplicationname = %q, want %q", data["remoteapplicationname"], "Notepad")
	}
}
