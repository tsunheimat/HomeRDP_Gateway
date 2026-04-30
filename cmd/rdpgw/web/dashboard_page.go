package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

type DashboardEntrySummary struct {
	ID                  string `json:"id"`
	Type                string `json:"type"`
	Name                string `json:"name"`
	Description         string `json:"description"`
	Icon                string `json:"icon"`
	Target              string `json:"target"`
	HasUploadedTemplate bool   `json:"hasUploadedTemplate"`
	DownloadURL         string `json:"downloadUrl"`
}

type DashboardUserInfo struct {
	Username      string    `json:"username"`
	DisplayName   string    `json:"displayName"`
	Email         string    `json:"email"`
	Groups        []string  `json:"groups"`
	Authenticated bool      `json:"authenticated"`
	AuthTime      time.Time `json:"authTime"`
	IsAdmin       bool      `json:"isAdmin"`
}

func (h *Handler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	id := identity.FromRequestCtx(r)
	if id == nil || !id.Authenticated() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	locale := dashboardLocaleFromRequest(r)
	messages := dashboardMessagesForLocale(locale)
	h.renderTemplatePage(w, "dashboard.html", fallbackDashboardTemplate, map[string]any{
		"Lang":          locale,
		"LanguageLinks": dashboardLanguageLinks(r, locale),
		"Messages":      messages,
		"MessagesJSON":  dashboardMessagesJSON(messages),
		"Title":         messages.WindowTitle,
	})
}

func (h *Handler) HandleAdminPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	id := identity.FromRequestCtx(r)
	if id == nil || !id.Authenticated() {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	if !IsAdmin(id, h.adminGroups) {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	locale := dashboardLocaleFromRequest(r)
	messages := adminMessagesForLocale(locale)
	h.renderTemplatePage(w, "admin.html", fallbackAdminTemplate, map[string]any{
		"Lang":          locale,
		"LanguageLinks": dashboardLanguageLinks(r, locale),
		"Messages":      messages,
		"MessagesJSON":  adminMessagesJSON(messages),
		"Title":         messages.WindowTitle,
	})
}

func (h *Handler) HandleEntryList(w http.ResponseWriter, r *http.Request) {
	id := identity.FromRequestCtx(r)
	if id == nil || !id.Authenticated() {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	entries, err := h.dashboardStore.List()
	if err != nil {
		http.Error(w, "unable to list dashboard entries", http.StatusInternalServerError)
		return
	}

	visible := dashboard.VisibleEntries(entries, id.Groups())
	summaries := make([]DashboardEntrySummary, 0, len(visible))
	for _, entry := range visible {
		summaries = append(summaries, DashboardEntrySummary{
			ID:                  entry.ID,
			Type:                string(entry.Type),
			Name:                entry.Name,
			Description:         entry.Description,
			Icon:                dashboard.NormalizeEntryIcon(entry.Icon),
			Target:              dashboardEntryActiveTarget(entry),
			HasUploadedTemplate: entry.UploadedTemplatePath != "",
			DownloadURL:         fmt.Sprintf("/connect/entries/%s.rdp", entry.ID),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
}

func dashboardEntryActiveTarget(entry dashboard.Entry) string {
	if entry.TargetIPOverride != "" && entry.ForceTargetIPOverride {
		return entry.TargetIPOverride
	}
	if entry.Type == dashboard.EntryTypeTemplate && entry.TargetHostOverride != "" {
		return entry.TargetHostOverride
	}
	return entry.Host
}

func (h *Handler) HandleDashboardUserInfo(w http.ResponseWriter, r *http.Request) {
	id := identity.FromRequestCtx(r)
	if id == nil || !id.Authenticated() {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	user := DashboardUserInfo{
		Username:      id.UserName(),
		DisplayName:   id.DisplayName(),
		Email:         id.Email(),
		Groups:        id.Groups(),
		Authenticated: id.Authenticated(),
		AuthTime:      id.AuthTime(),
		IsAdmin:       IsAdmin(id, h.adminGroups),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(user)
}

func (h *Handler) renderTemplatePage(w http.ResponseWriter, filename, fallback string, data any) {
	tmpl := h.loadTemplateWithFallback(filename, fallback)
	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		log.Printf("Failed to execute template %s: %v", filename, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = rendered.WriteTo(w)
}

func (h *Handler) loadTemplateWithFallback(filename, fallback string) *template.Template {
	templatePath, ok := h.resolveTemplateFile(filename)
	if ok {
		tmpl, parseErr := template.ParseFiles(templatePath)
		if parseErr == nil {
			return tmpl
		}
		log.Printf("Warning: Failed to parse template %s: %v", templatePath, parseErr)
	}

	log.Printf("Using fallback template for %s", filename)
	return template.Must(template.New(filename).Parse(fallback))
}

const fallbackDashboardTemplate = `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>{{.Title}}</title>
	<link rel="stylesheet" href="/static/style.css">
	<link rel="icon" href="/assets/app-icon">
</head>
<body>
	<div class="app-shell" id="app-shell">
		<header class="shell-topbar" id="app-topbar">
			<div class="logo">
				<img src="/assets/app-icon" alt="Logo">
				RDP Gateway
			</div>
			<div class="user-info">
				<div class="lang-switch" aria-label="Language switcher">{{range .LanguageLinks}}<a class="lang-switch-link{{if .Current}} is-active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}</div>
				<div class="user-avatar" id="userAvatar"></div>
				<span id="dashboardUsername">{{.Messages.LoadingUser}}</span>
				<a class="admin-link" id="adminLink" href="/admin" hidden>{{.Messages.AdminLink}}</a>
			</div>
		</header>
		<main class="shell-main" id="app-main">
			<div class="page-intro" id="app-intro">
				<h1 class="title">{{.Messages.Heading}}</h1>
				<p class="subtitle">{{.Messages.Subtitle}}</p>
				<div id="summaryStrip" class="summary-strip" hidden></div>
			</div>
			<div class="dashboard-controls" id="dashboardControls" hidden>
				<input type="search" id="entrySearch" class="search-input" placeholder="{{.Messages.SearchPlaceholder}}">
			</div>
			<div class="inline-alert is-error" id="dashboardError">
				<span id="dashboardErrorText"></span>
				<button type="button" id="dashboardRetryBtn" class="secondary-button retry-btn" hidden>{{.Messages.RetryButton}}</button>
			</div>
			<div class="inline-alert is-success" id="dashboardSuccess"></div>
			<div class="content-section">
				<div id="entriesLoading" class="entries-grid">
					<div class="skeleton-row"></div>
					<div class="skeleton-row"></div>
					<div class="skeleton-row"></div>
				</div>
				<div class="entries-grid" id="entriesGrid"></div>
				<div class="empty-state" id="entriesEmpty" hidden>
					<div class="empty-icon">
						<svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"></path><polyline points="3.27 6.96 12 12.01 20.73 6.96"></polyline><line x1="12" y1="22.08" x2="12" y2="12"></line></svg>
					</div>
					<p>{{.Messages.EmptyState}}</p>
				</div>
			</div>
		</main>
	</div>
	<script>window.dashboardMessages = {{.MessagesJSON}};</script>
	<script src="/static/dashboard.js"></script>
</body>
</html>
`

const fallbackAdminTemplate = `<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/static/style.css">
    <link rel="icon" href="/assets/app-icon">
    <link rel="alternate icon" href="/assets/app-icon">
</head>
<body>
    <div class="app-shell" id="app-shell">
        <header class="shell-topbar" id="app-topbar">
            <div class="logo">
                <img src="/assets/app-icon" alt="Logo" id="adminLogoImage">
                RDP Gateway
            </div>
            <div class="user-info">
                <div class="lang-switch" aria-label="Language switcher">{{range .LanguageLinks}}<a class="lang-switch-link{{if .Current}} is-active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}</div>
                <span id="adminUsername">{{.Messages.LoadingUser}}</span>
                <a class="admin-link" href="/">{{.Messages.BackToDashboard}}</a>
            </div>
        </header>

        <main class="shell-main" id="app-main">
            <div class="page-intro" id="app-intro">
                <h1 class="title">{{.Messages.Heading}}</h1>
                <p class="subtitle">{{.Messages.Subtitle}}</p>
            </div>

            <nav class="section-switcher" aria-label="Admin sections">
                <button type="button" class="switcher-tab is-active" data-target="section-entries">{{.Messages.TabPublishedEntries}}</button>
                <button type="button" class="switcher-tab" data-target="section-auth-users">{{.Messages.TabDirectAuthUsers}}</button>
                <button type="button" class="switcher-tab" data-target="section-branding">{{.Messages.TabBranding}}</button>
            </nav>

            <div id="section-entries" class="admin-section">
                <div class="inline-alert is-error" id="entriesError"></div>
                <div class="inline-alert is-success" id="entriesSuccess"></div>

                <div class="content-section">
                    <section class="admin-panel">
                        <h2 class="section-header">{{.Messages.CurrentEntriesHeading}}</h2>
                        <div id="adminEntries"></div>
                    </section>
                </div>

                <div class="content-section">
                    <div class="admin-grid">
                        <section class="admin-panel">
                            <h2 class="section-header">{{.Messages.CreateHostEntryHeading}}</h2>
                            <form id="hostForm" class="stack-form">
                                <label>
                                    {{.Messages.NameLabel}}
                                    <input type="text" name="name" required>
                                </label>
                                <label>
                                    {{.Messages.DescriptionLabel}}
                                    <input type="text" name="description">
                                </label>
                                <label>
                                    {{.Messages.EntryIconLabel}}
                                    <select name="icon" data-entry-icon-select></select>
                                </label>
                                <label>
                                    {{.Messages.AllowedGroupsLabel}}
                                    <input type="text" name="allowedGroups" placeholder="{{.Messages.AllowedGroupsPlaceholder}}" required>
                                </label>
                                <label>
                                    {{.Messages.HostLabel}}
                                    <input type="text" name="host" placeholder="{{.Messages.HostPlaceholder}}" required>
                                </label>
                                <label>
                                    {{.Messages.TargetIPOverrideLabel}}
                                    <input type="text" name="targetIPOverride" placeholder="{{.Messages.TargetIPOverridePlaceholder}}">
                                </label>
                                <label class="checkbox-row">
                                    <input type="checkbox" name="forceTargetIPOverride">
                                    {{.Messages.ForceTargetIPOverrideLabel}}
                                </label>
                                <button type="submit" class="primary-button">{{.Messages.CreateHostEntryButton}}</button>
                            </form>
                        </section>

                        <section class="admin-panel">
                            <h2 class="section-header">{{.Messages.CreateTemplateEntryHeading}}</h2>
                            <form id="templateForm" class="stack-form" enctype="multipart/form-data">
                                <label>
                                    {{.Messages.NameLabel}}
                                    <input type="text" name="name" required>
                                </label>
                                <label>
                                    {{.Messages.DescriptionLabel}}
                                    <input type="text" name="description">
                                </label>
                                <label>
                                    {{.Messages.EntryIconLabel}}
                                    <select name="icon" data-entry-icon-select></select>
                                </label>
                                <label>
                                    {{.Messages.AllowedGroupsLabel}}
                                    <input type="text" name="allowedGroups" placeholder="{{.Messages.AllowedGroupsPlaceholder}}" required>
                                </label>
                                <label>
                                    {{.Messages.TargetHostOverrideLabel}}
                                    <input type="text" name="targetHostOverride" placeholder="{{.Messages.TargetHostOverridePlaceholder}}">
                                </label>
                                <label>
                                    {{.Messages.TargetIPOverrideLabel}}
                                    <input type="text" name="targetIPOverride" placeholder="{{.Messages.TargetIPOverridePlaceholder}}">
                                </label>
                                <label class="checkbox-row">
                                    <input type="checkbox" name="forceTargetIPOverride">
                                    {{.Messages.ForceTargetIPOverrideLabel}}
                                </label>
                                <label>
                                    {{.Messages.RdpTemplateFileLabel}}
                                    <input type="file" name="template" accept=".rdp" required>
                                </label>
                                <p class="form-hint" id="templateSuggestionStatus">{{.Messages.TemplateSuggestionHint}}</p>
                                <button type="submit" class="primary-button">{{.Messages.UploadTemplateEntryButton}}</button>
                            </form>
                        </section>
                    </div>
                </div>
            </div>

            <div id="section-auth-users" class="admin-section" hidden>
                <div class="inline-alert is-error" id="authUsersError" role="alert"></div>
                <div class="inline-alert is-success" id="authUsersSuccess" aria-live="polite"></div>

                <div class="content-section">
                    <section class="admin-panel">
                        <h2 class="section-header">{{.Messages.CurrentDirectAuthUsersHeading}}</h2>
                        <div id="adminAuthUsers"></div>
                    </section>
                </div>

                <div class="content-section">
                    <div class="admin-grid">
                        <section class="admin-panel">
                            <h2 class="section-header">{{.Messages.CreateDirectAuthUserHeading}}</h2>
                            <form id="authUserForm" class="stack-form">
                                <label>
                                    {{.Messages.UsernameLabel}}
                                    <input type="text" name="username" required>
                                </label>
                                <label>
                                    {{.Messages.PasswordLabel}}
                                    <input type="password" name="password" required>
                                </label>
                                <label class="checkbox-row">
                                    <input type="checkbox" name="enabled" checked>
                                    {{.Messages.EnabledLabel}}
                                </label>
                                <button type="submit" class="primary-button">{{.Messages.CreateDirectAuthUserButton}}</button>
                            </form>
                        </section>
                    </div>
                </div>
            </div>

            <div id="section-branding" class="admin-section" hidden>
                <div class="inline-alert is-error" id="brandingError"></div>
                <div class="inline-alert is-success" id="brandingSuccess"></div>

                <div class="content-section">
                    <div class="admin-grid">
                        <section class="admin-panel">
                            <h2 class="section-header">{{.Messages.CurrentBrandingHeading}}</h2>
                            <div class="branding-preview">
                                <img src="/assets/app-icon" alt="{{.Messages.CurrentIconAlt}}" id="currentAppIcon">
                                <div>
                                    <strong id="activeIconName">{{.Messages.DefaultIconLabel}}</strong>
                                    <p class="form-hint">{{.Messages.CurrentIconHint}}</p>
                                </div>
                            </div>
                            <div id="brandingIcons"></div>
                        </section>

                        <section class="admin-panel">
                            <h2 class="section-header">{{.Messages.UploadIconHeading}}</h2>
                            <form id="iconForm" class="stack-form" action="/api/v1/admin/icon" method="post" enctype="multipart/form-data">
                                <label>
                                    {{.Messages.IconFileLabel}}
                                    <input type="file" name="icon" accept=".ico,.icon,.png,.jpg,.jpeg" required>
                                </label>
                                <p class="form-hint">{{.Messages.IconUploadHint}}</p>
                                <button type="submit" class="primary-button">{{.Messages.UploadIconButton}}</button>
                            </form>
                        </section>
                    </div>
                </div>
            </div>
        </main>
    </div>

    <script>window.adminMessages = {{.MessagesJSON}};</script>
    <script src="/static/admin.js"></script>
</body>
</html>
`
