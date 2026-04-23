package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

func TestHandleEntryListFiltersByGroups(t *testing.T) {
	handler, store := newDashboardTestHandler(t)

	if err := store.Put(dashboard.Entry{
		ID:            "entry-visible",
		Type:          dashboard.EntryTypeHost,
		Name:          "Visible Host",
		Description:   "Visible to homelab users",
		Icon:          dashboard.EntryIconBrowser,
		AllowedGroups: []string{"homelab-users"},
		Enabled:       true,
		Host:          "visible.internal:3389",
	}); err != nil {
		t.Fatalf("put visible entry: %v", err)
	}
	if err := store.Put(dashboard.Entry{
		ID:            "entry-hidden",
		Type:          dashboard.EntryTypeHost,
		Name:          "Hidden Host",
		Description:   "Visible to admins only",
		AllowedGroups: []string{"rdpgw-admins"},
		Enabled:       true,
		Host:          "hidden.internal:3389",
	}); err != nil {
		t.Fatalf("put hidden entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleEntryList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var summaries []DashboardEntrySummary
	if err := json.Unmarshal(rr.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(summaries) != 1 {
		t.Fatalf("entry count = %d, want %d", len(summaries), 1)
	}
	if summaries[0].ID != "entry-visible" {
		t.Fatalf("entry id = %q, want %q", summaries[0].ID, "entry-visible")
	}
	if summaries[0].DownloadURL != "/connect/entries/entry-visible.rdp" {
		t.Fatalf("download url = %q", summaries[0].DownloadURL)
	}
	if summaries[0].Icon != dashboard.EntryIconBrowser {
		t.Fatalf("icon = %q, want %q", summaries[0].Icon, dashboard.EntryIconBrowser)
	}
}

func TestHandleEntryListPreservesUploadedIconReference(t *testing.T) {
	handler, store, iconStore := newDashboardEntryIconTestHandler(t)

	icon, err := iconStore.SaveIcon("custom.png", []byte{137, 80, 78, 71})
	if err != nil {
		t.Fatalf("save icon: %v", err)
	}
	customIcon := "uploaded:" + icon.ID

	if err := store.Put(dashboard.Entry{
		ID:            "entry-custom-icon",
		Type:          dashboard.EntryTypeHost,
		Name:          "Custom Icon Host",
		Icon:          customIcon,
		AllowedGroups: []string{"homelab-users"},
		Enabled:       true,
		Host:          "custom.internal:3389",
	}); err != nil {
		t.Fatalf("put visible entry: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/entries", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleEntryList(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var summaries []DashboardEntrySummary
	if err := json.Unmarshal(rr.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("entry count = %d, want %d", len(summaries), 1)
	}
	if summaries[0].Icon != customIcon {
		t.Fatalf("summary icon = %q, want %q", summaries[0].Icon, customIcon)
	}
}

func TestHandleDashboardUserInfoIncludesAdminState(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/user", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboardUserInfo(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	var info DashboardUserInfo
	if err := json.Unmarshal(rr.Body.Bytes(), &info); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if info.Username != "admin@example.com" {
		t.Fatalf("username = %q", info.Username)
	}
	if info.DisplayName != "Admin User" {
		t.Fatalf("display name = %q", info.DisplayName)
	}
	if info.Email != "admin@example.com" {
		t.Fatalf("email = %q", info.Email)
	}
	if len(info.Groups) != 2 {
		t.Fatalf("groups len = %d", len(info.Groups))
	}
	if !info.Authenticated {
		t.Fatalf("expected authenticated=true")
	}
	if info.AuthTime.IsZero() {
		t.Fatalf("expected non-zero auth time")
	}
	if !info.IsAdmin {
		t.Fatalf("expected isAdmin=true")
	}
}

func TestHandleDashboardRendersTemplate(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	customHTML := "<!DOCTYPE html><html><body>dashboard template marker</body></html>"
	if err := os.WriteFile(filepath.Join(handler.templatesPath, "dashboard.html"), []byte(customHTML), 0o600); err != nil {
		t.Fatalf("write dashboard template: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}
	if rr.Body.String() != customHTML {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

func TestDashboardLocaleFromRequest(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		acceptLanguage string
		want           string
	}{
		{
			name: "defaults to english",
			want: dashboardLocaleEnglish,
		},
		{
			name:  "query override wins",
			query: "/?lang=zh-TW",
			want:  dashboardLocaleTraditionalChinese,
		},
		{
			name:           "accept language alone does not change dashboard locale",
			acceptLanguage: "zh-HK,zh;q=0.9,en;q=0.8",
			want:           dashboardLocaleEnglish,
		},
		{
			name:           "unsupported locale falls back to english",
			acceptLanguage: "fr-FR,fr;q=0.9",
			want:           dashboardLocaleEnglish,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := tt.query
			if target == "" {
				target = "/"
			}
			req := httptest.NewRequest(http.MethodGet, target, nil)
			if tt.acceptLanguage != "" {
				req.Header.Set("Accept-Language", tt.acceptLanguage)
			}

			if got := dashboardLocaleFromRequest(req); got != tt.want {
				t.Fatalf("dashboardLocaleFromRequest() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHandleDashboardRendersTraditionalChineseFromQueryLanguage(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `<html lang="zh-Hant">`) {
		t.Fatalf("expected zh-Hant html lang, body=%q", body)
	}
	if !strings.Contains(body, "連線儀表板") {
		t.Fatalf("expected Traditional Chinese dashboard title, body=%q", body)
	}
	if !strings.Contains(body, "下載 RDP") {
		t.Fatalf("expected Traditional Chinese button label in i18n payload, body=%q", body)
	}
}

func TestHandleDashboardQueryLangOverrideBeatsHeader(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?lang=en", nil)
	req.Header.Set("Accept-Language", "zh-TW,zh;q=0.9")
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("expected english html lang, body=%q", body)
	}
	if !strings.Contains(body, "Connection Dashboard") {
		t.Fatalf("expected English dashboard title, body=%q", body)
	}
}

func TestHandleDashboardFallbackTemplateAlsoUsesLocalizedCopy(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "目前沒有可供您群組使用的項目。") {
		t.Fatalf("expected localized fallback empty state, body=%q", body)
	}
}

func TestHandleDashboardRendersLanguageSwitchLinks(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `href="/?lang=en"`) {
		t.Fatalf("expected English switch link, body=%q", body)
	}
	if !strings.Contains(body, `href="/?lang=zh-Hant"`) {
		t.Fatalf("expected Traditional Chinese switch link, body=%q", body)
	}
	if !strings.Contains(body, `class="lang-switch-link is-active"`) {
		t.Fatalf("expected active language switch class, body=%q", body)
	}
}

func TestHandleDashboardRendersVisibleLoadingState(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = identity.AddToRequestCtx(newIdentity(t, false), req)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Loading dashboard entries...") {
		t.Fatalf("expected visible loading state text, body=%q", body)
	}
	if strings.Contains(body, `id="entriesLoading" hidden`) {
		t.Fatalf("expected loading state to be visible by default, body=%q", body)
	}
}

func TestHandleAdminPageDefaultsToEnglishWithoutLangQuery(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `<html lang="en">`) {
		t.Fatalf("expected english html lang, body=%q", body)
	}
	if !strings.Contains(body, "Entry Administration") {
		t.Fatalf("expected english admin title, body=%q", body)
	}
	if !strings.Contains(body, "Back to Dashboard") {
		t.Fatalf("expected english back link, body=%q", body)
	}
	if !strings.Contains(body, "Create RDP Gateway User") {
		t.Fatalf("expected updated English direct-auth heading, body=%q", body)
	}
}

func TestHandleAdminPageRendersGlobalBrandingUploadForm(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `id="section-branding"`) {
		t.Fatalf("expected global branding section, body=%q", body)
	}
	if !strings.Contains(body, `id="iconForm" class="stack-form" action="/api/v1/admin/icon" method="post" enctype="multipart/form-data"`) {
		t.Fatalf("expected icon form to post to global icon API, body=%q", body)
	}
}

func TestHandleAdminPageRejectsNonGetRequests(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/admin", strings.NewReader("ignored"))
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleAdminPageRendersTraditionalChineseFromQueryLanguage(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `<html lang="zh-Hant">`) {
		t.Fatalf("expected zh-Hant html lang, body=%q", body)
	}
	if !strings.Contains(body, "項目管理") {
		t.Fatalf("expected Traditional Chinese admin title, body=%q", body)
	}
	if !strings.Contains(body, "返回儀表板") {
		t.Fatalf("expected Traditional Chinese back link, body=%q", body)
	}
	if !strings.Contains(body, "建立主機項目") {
		t.Fatalf("expected Traditional Chinese form heading, body=%q", body)
	}
}

func TestHandleAdminPageRendersLanguageSwitchLinks(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `href="/admin?lang=en"`) {
		t.Fatalf("expected english switch link, body=%q", body)
	}
	if !strings.Contains(body, `href="/admin?lang=zh-Hant"`) {
		t.Fatalf("expected Traditional Chinese switch link, body=%q", body)
	}
	if !strings.Contains(body, `class="lang-switch-link is-active"`) {
		t.Fatalf("expected active language switch class, body=%q", body)
	}
}

func TestHandleAdminPageFallbackTemplateAlsoUsesLocalizedCopy(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "目前沒有已設定的項目。") {
		t.Fatalf("expected localized admin empty-state payload, body=%q", body)
	}
}

func TestHandleAdminPageIncludesLocalizedStatusActionLabels(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/admin?lang=zh-Hant", nil)
	req = identity.AddToRequestCtx(newIdentity(t, true), req)
	rr := httptest.NewRecorder()

	handler.HandleAdminPage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"enableButton":"啟用"`) {
		t.Fatalf("expected Traditional Chinese enable action in payload, body=%q", body)
	}
	if !strings.Contains(body, `"disableButton":"停用"`) {
		t.Fatalf("expected Traditional Chinese disable action in payload, body=%q", body)
	}
	if !strings.Contains(body, `"enabledStatus":"已啟用"`) {
		t.Fatalf("expected Traditional Chinese enabled status in payload, body=%q", body)
	}
	if !strings.Contains(body, `"disabledStatus":"已停用"`) {
		t.Fatalf("expected Traditional Chinese disabled status in payload, body=%q", body)
	}
}

func TestHandleDashboardRedirectsUnauthenticatedUsersToRoot(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.HandleDashboard(rr, req)

	if rr.Code != http.StatusFound {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusFound)
	}
	if location := rr.Header().Get("Location"); location != "/" {
		t.Fatalf("location = %q, want %q", location, "/")
	}
}

func TestRenderTemplatePageDoesNotPartiallyWriteOnExecuteError(t *testing.T) {
	handler, _ := newDashboardTestHandler(t)

	const broken = "<!DOCTYPE html><html><body>prefix {{call .Fail}}</body></html>"
	if err := os.WriteFile(filepath.Join(handler.templatesPath, "broken.html"), []byte(broken), 0o600); err != nil {
		t.Fatalf("write broken template: %v", err)
	}

	rr := httptest.NewRecorder()
	handler.renderTemplatePage(rr, "broken.html", broken, map[string]any{
		"Fail": func() (string, error) {
			return "", errors.New("boom")
		},
	})

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status code = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if body := rr.Body.String(); strings.Contains(body, "prefix") {
		t.Fatalf("expected no partial template body, got %q", body)
	}
}

func newDashboardTestHandler(t *testing.T) (*Handler, dashboard.Store) {
	t.Helper()

	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}

	handler := (&Config{
		Hosts:          []string{"fallback.internal:3389"},
		HostSelection:  "roundrobin",
		DashboardStore: store,
		AdminGroups:    []string{"rdpgw-admins"},
		TemplatesPath:  tmpDir,
	}).NewHandler()

	return handler, store
}

func newDashboardEntryIconTestHandler(t *testing.T) (*Handler, dashboard.Store, dashboard.IconStore) {
	t.Helper()

	tmpDir := t.TempDir()
	store, err := dashboard.NewFileStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "uploads"))
	if err != nil {
		t.Fatalf("new dashboard store: %v", err)
	}
	iconStore, err := dashboard.NewFileIconStore(filepath.Join(tmpDir, "catalog"), filepath.Join(tmpDir, "icons"))
	if err != nil {
		t.Fatalf("new icon store: %v", err)
	}

	handler := (&Config{
		Hosts:              []string{"fallback.internal:3389"},
		HostSelection:      "roundrobin",
		DashboardStore:     store,
		DashboardIconStore: iconStore,
		AdminGroups:        []string{"rdpgw-admins"},
		TemplatesPath:      tmpDir,
	}).NewHandler()

	return handler, store, iconStore
}

func newIdentity(t *testing.T, admin bool) identity.Identity {
	t.Helper()

	id := identity.NewUser()
	id.SetAuthenticated(true)
	id.SetAuthTime(time.Now().UTC())

	if admin {
		id.SetUserName("admin@example.com")
		id.SetDisplayName("Admin User")
		id.SetEmail("admin@example.com")
		id.SetGroups([]string{"homelab-users", "rdpgw-admins"})
		return id
	}

	id.SetUserName("user@example.com")
	id.SetDisplayName("Regular User")
	id.SetEmail("user@example.com")
	id.SetGroups([]string{"homelab-users"})
	return id
}
