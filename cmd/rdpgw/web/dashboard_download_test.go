package web

import (
	"context"
	"net/http"
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
		ID:            "lab-win11",
		Type:          dashboard.EntryTypeHost,
		Name:          "Lab Win11",
		AllowedGroups: []string{"homelab-users"},
		Enabled:       true,
		Host:          "lab-host.internal:3389",
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	handler := (&Config{
		Hosts:             []string{"fallback.internal:3389"},
		HostSelection:     "roundrobin",
		DashboardStore:    store,
		EnableUserToken:   true,
		PAATokenGenerator: paaTokenMock,
		UserTokenGenerator: func(ctx context.Context, user string) (string, error) {
			return user + "-token", nil
		},
		GatewayAddress: gatewayAddress,
		RdpOpts: RdpOpts{
			UsernameTemplate: "{{ username }}#{{ token }}",
			SplitUserDomain:  true,
		},
	}).NewHandler()

	req := httptest.NewRequest("GET", "/connect/entries/lab-win11.rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "lab-win11"})

	id := identity.NewUser()
	id.SetUserName("alice@example.com")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"homelab-users"})
	req = identity.AddToRequestCtx(id, req)

	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)

	if recorder.Code != 200 {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}

	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "lab-host.internal:3389" {
		t.Fatalf("full address = %q, want %q", data["full address"], "lab-host.internal:3389")
	}
	if data["gatewayhostname"] != gatewayAddress.Host {
		t.Fatalf("gatewayhostname = %q, want %q", data["gatewayhostname"], gatewayAddress.Host)
	}
	if data["username"] != "alice#alice-token" {
		t.Fatalf("username = %q, want %q", data["username"], "alice#alice-token")
	}
	if data["domain"] != "example.com" {
		t.Fatalf("domain = %q, want %q", data["domain"], "example.com")
	}
}

func TestHandleEntryDownloadTemplateEntryPreservesRemoteApp(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	template := strings.Join([]string{
		"full address:s:template-{{ preferred_username }}.internal:3389",
		"username:s:stale-user",
		"domain:s:stale-domain",
		"remoteapplicationmode:i:1",
		"remoteapplicationprogram:s:||notepad",
		"remoteapplicationname:s:Notepad",
	}, rdp.CRLF) + rdp.CRLF
	uploadPath, err := store.SaveUpload(strings.NewReader(template))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}

	entry := dashboard.Entry{
		ID:                   "office-app",
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
	req = mux.SetURLVars(req, map[string]string{"id": "office-app"})

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
	if data["full address"] != "template-bob.internal:3389" {
		t.Fatalf("full address = %q, want %q", data["full address"], "template-bob.internal:3389")
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
	if data["username"] != "bob" {
		t.Fatalf("username = %q, want %q", data["username"], "bob")
	}
	if _, found := data["domain"]; found {
		t.Fatalf("domain should be cleared, got %q", data["domain"])
	}
}

func TestHandleEntryDownloadOverridesTemplateRedirectionSettingsFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	template := strings.Join([]string{
		"full address:s:template.internal:3389",
		"redirectclipboard:i:1",
		"redirectprinters:i:1",
		"redirectcomports:i:1",
		"drivestoredirect:s:*",
		"devicestoredirect:s:*",
		"usbdevicestoredirect:s:*",
	}, rdp.CRLF) + rdp.CRLF
	uploadPath, err := store.SaveUpload(strings.NewReader(template))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}

	entry := dashboard.Entry{
		ID:                   "locked-down-app",
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 "Locked Down App",
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
		RdpRedirection: RdpRedirectionPolicy{
			Clipboard: false,
			Drive:     false,
			Printer:   false,
			Port:      false,
			Device:    false,
			Pnp:       false,
		},
	}).NewHandler()

	req := httptest.NewRequest("GET", "/connect/entries/locked-down-app.rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "locked-down-app"})

	id := identity.NewUser()
	id.SetUserName("bob")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"office-users"})
	req = identity.AddToRequestCtx(id, req)

	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["redirectclipboard"] != "0" {
		t.Fatalf("redirectclipboard = %q, want 0", data["redirectclipboard"])
	}
	if data["redirectprinters"] != "0" {
		t.Fatalf("redirectprinters = %q, want 0", data["redirectprinters"])
	}
	if data["redirectcomports"] != "0" {
		t.Fatalf("redirectcomports = %q, want 0", data["redirectcomports"])
	}
	if data["drivestoredirect"] != "false" {
		t.Fatalf("drivestoredirect = %q, want false", data["drivestoredirect"])
	}
	if data["devicestoredirect"] != "false" {
		t.Fatalf("devicestoredirect = %q, want false", data["devicestoredirect"])
	}
	if data["usbdevicestoredirect"] != "false" {
		t.Fatalf("usbdevicestoredirect = %q, want false", data["usbdevicestoredirect"])
	}
}

func TestHandleEntryDownloadTemplateEntryRequiresTargetHost(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	template := "remoteapplicationmode:i:1\r\n"
	uploadPath, err := store.SaveUpload(strings.NewReader(template))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}

	entry := dashboard.Entry{
		ID:                   "broken-app",
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 "Broken App",
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

	req := httptest.NewRequest("GET", "/connect/entries/broken-app.rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "broken-app"})

	id := identity.NewUser()
	id.SetUserName("bob")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"office-users"})
	req = identity.AddToRequestCtx(id, req)

	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}

func TestHandleEntryDownloadRejectsInvalidRenderedTemplateHost(t *testing.T) {
	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	template := "full address:s:template-{{ preferred_username }}.internal:3389\r\n"
	uploadPath, err := store.SaveUpload(strings.NewReader(template))
	if err != nil {
		t.Fatalf("save upload: %v", err)
	}

	entry := dashboard.Entry{
		ID:                   "broken-host-template",
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 "Broken Host Template",
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

	req := httptest.NewRequest(http.MethodGet, "/connect/entries/broken-host-template.rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "broken-host-template"})

	id := identity.NewUser()
	id.SetUserName("bob:3389")
	id.SetAuthenticated(true)
	id.SetGroups([]string{"office-users"})
	req = identity.AddToRequestCtx(id, req)

	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
}
