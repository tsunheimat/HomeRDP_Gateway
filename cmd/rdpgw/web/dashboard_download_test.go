package web

import (
	"context"
	"errors"
	"net"
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

func TestHandleEntryDownloadUsesInternalDNSForInternalDomain(t *testing.T) {
	store := newSingleHostEntryStore(t, dashboard.Entry{
		ID:            "internal-app",
		Type:          dashboard.EntryTypeHost,
		Name:          "Internal App",
		AllowedGroups: []string{"office-users"},
		Enabled:       true,
		Host:          "app.corp.internal:3389",
	})

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	var tokenHost string
	handler := (&Config{
		DashboardStore:    store,
		PAATokenGenerator: recordingPAATokenMock(&tokenHost),
		GatewayAddress:    gatewayAddress,
		InternalDomains:   []string{"corp.internal"},
		InternalDNSServer: "10.0.0.53:53",
		InternalDNSLookupIP: func(ctx context.Context, dnsServer string, host string) ([]net.IP, error) {
			if dnsServer != "10.0.0.53:53" || host != "app.corp.internal" {
				t.Fatalf("lookup called with dns=%q host=%q", dnsServer, host)
			}
			return []net.IP{net.ParseIP("10.1.2.3")}, nil
		},
	}).NewHandler()

	recorder := performEntryDownload(t, handler, "internal-app", "alice", []string{"office-users"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "10.1.2.3:3389" {
		t.Fatalf("full address = %q, want internal DNS IP", data["full address"])
	}
	if tokenHost != "10.1.2.3:3389" {
		t.Fatalf("token host = %q, want resolved IP host", tokenHost)
	}
}

func TestHandleEntryDownloadFallsBackToConfiguredTargetIPWhenInternalDNSFails(t *testing.T) {
	store := newSingleHostEntryStore(t, dashboard.Entry{
		ID:               "fallback-app",
		Type:             dashboard.EntryTypeHost,
		Name:             "Fallback App",
		AllowedGroups:    []string{"office-users"},
		Enabled:          true,
		Host:             "app.corp.internal:3389",
		TargetIPOverride: "10.9.8.7",
	})

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	handler := (&Config{
		DashboardStore:    store,
		PAATokenGenerator: paaTokenMock,
		GatewayAddress:    gatewayAddress,
		InternalDomains:   []string{"corp.internal"},
		InternalDNSServer: "10.0.0.53:53",
		InternalDNSLookupIP: func(ctx context.Context, dnsServer string, host string) ([]net.IP, error) {
			return nil, errors.New("dns unavailable")
		},
	}).NewHandler()

	recorder := performEntryDownload(t, handler, "fallback-app", "alice", []string{"office-users"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "10.9.8.7:3389" {
		t.Fatalf("full address = %q, want fallback IP", data["full address"])
	}
}

func TestHandleEntryDownloadForceTargetIPOverrideSkipsInternalDNS(t *testing.T) {
	store := newSingleHostEntryStore(t, dashboard.Entry{
		ID:                    "forced-app",
		Type:                  dashboard.EntryTypeHost,
		Name:                  "Forced App",
		AllowedGroups:         []string{"office-users"},
		Enabled:               true,
		Host:                  "app.corp.internal:3389",
		TargetIPOverride:      "10.9.8.7",
		ForceTargetIPOverride: true,
	})

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	handler := (&Config{
		DashboardStore:    store,
		PAATokenGenerator: paaTokenMock,
		GatewayAddress:    gatewayAddress,
		InternalDomains:   []string{"corp.internal"},
		InternalDNSServer: "10.0.0.53:53",
		InternalDNSLookupIP: func(ctx context.Context, dnsServer string, host string) ([]net.IP, error) {
			t.Fatal("internal DNS should not be called when force target IP override is enabled")
			return nil, nil
		},
	}).NewHandler()

	recorder := performEntryDownload(t, handler, "forced-app", "alice", []string{"office-users"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "10.9.8.7:3389" {
		t.Fatalf("full address = %q, want forced IP", data["full address"])
	}
}

func TestHandleEntryDownloadLiteralIPAddressIsNotOverwrittenByInternalDNS(t *testing.T) {
	store := newSingleHostEntryStore(t, dashboard.Entry{
		ID:                    "ip-app",
		Type:                  dashboard.EntryTypeHost,
		Name:                  "IP App",
		AllowedGroups:         []string{"office-users"},
		Enabled:               true,
		Host:                  "10.2.3.4:3389",
		TargetIPOverride:      "10.9.8.7",
		ForceTargetIPOverride: true,
	})

	gatewayAddress, _ := url.Parse("https://gw.example.com:443")
	handler := (&Config{
		DashboardStore:    store,
		PAATokenGenerator: paaTokenMock,
		GatewayAddress:    gatewayAddress,
		InternalDomains:   []string{"corp.internal"},
		InternalDNSServer: "10.0.0.53:53",
		InternalDNSLookupIP: func(ctx context.Context, dnsServer string, host string) ([]net.IP, error) {
			t.Fatal("internal DNS should not be called for literal IP targets")
			return nil, nil
		},
	}).NewHandler()

	recorder := performEntryDownload(t, handler, "ip-app", "alice", []string{"office-users"})
	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	data := rdpToMap(strings.Split(recorder.Body.String(), rdp.CRLF))
	if data["full address"] != "10.2.3.4:3389" {
		t.Fatalf("full address = %q, want literal IP unchanged", data["full address"])
	}
}

func newSingleHostEntryStore(t *testing.T, entry dashboard.Entry) dashboard.Store {
	t.Helper()
	store, err := dashboard.NewFileStore(filepath.Join(t.TempDir(), "catalog"), filepath.Join(t.TempDir(), "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}
	if err := store.Put(entry); err != nil {
		t.Fatalf("put entry: %v", err)
	}
	return store
}

func performEntryDownload(t *testing.T, handler *Handler, entryID string, username string, groups []string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/connect/entries/"+entryID+".rdp", nil)
	req = mux.SetURLVars(req, map[string]string{"id": entryID})
	id := identity.NewUser()
	id.SetUserName(username)
	id.SetAuthenticated(true)
	id.SetGroups(groups)
	req = identity.AddToRequestCtx(id, req)
	recorder := httptest.NewRecorder()
	handler.HandleEntryDownload(recorder, req)
	return recorder
}

func recordingPAATokenMock(hostOut *string) TokenGeneratorFunc {
	return func(ctx context.Context, username string, host string) (string, error) {
		*hostOut = host
		return username + "_" + host, nil
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
