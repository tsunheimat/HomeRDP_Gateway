package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
)

type DashboardEntrySummary struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DownloadURL string `json:"downloadUrl"`
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
			ID:          entry.ID,
			Type:        string(entry.Type),
			Name:        entry.Name,
			Description: entry.Description,
			DownloadURL: fmt.Sprintf("/connect/entries/%s.rdp", entry.ID),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(summaries)
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
	templatePath := filepath.Join(h.templatesPath, filename)
	if _, err := os.Stat(templatePath); err == nil {
		tmpl, parseErr := template.ParseFiles(templatePath)
		if parseErr == nil {
			return tmpl
		}
		log.Printf("Warning: Failed to parse template %s: %v", templatePath, parseErr)
	} else if !os.IsNotExist(err) {
		log.Printf("Warning: Failed to stat template %s: %v", templatePath, err)
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
	<link rel="icon" type="image/svg+xml" href="/assets/icon.svg">
</head>
<body>
	<div class="header">
		<div class="logo">
			<img src="/assets/icon.svg" alt="Logo">
			RDP Gateway
		</div>
		<div class="user-info">
			<div class="lang-switch" aria-label="Language switcher">{{range .LanguageLinks}}<a class="lang-switch-link{{if .Current}} is-active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}</div>
			<div class="user-avatar" id="userAvatar"></div>
			<span id="dashboardUsername">{{.Messages.LoadingUser}}</span>
			<a class="admin-link" id="adminLink" href="/admin" hidden>{{.Messages.AdminLink}}</a>
		</div>
	</div>
	<main class="main">
		<div class="container dashboard-container">
			<h1 class="title">{{.Messages.Heading}}</h1>
			<p class="subtitle">{{.Messages.Subtitle}}</p>
			<div class="success" id="dashboardSuccess"></div>
			<div class="error" id="dashboardError"></div>
			<div class="empty-state" id="entriesLoading">{{.Messages.LoadingEntries}}</div>
			<div class="empty-state" id="entriesEmpty" hidden>{{.Messages.EmptyState}}</div>
			<div class="entries-grid" id="entriesGrid"></div>
		</div>
	</main>
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
	<link rel="icon" type="image/svg+xml" href="/assets/icon.svg">
</head>
<body>
	<div class="header">
		<div class="logo">
			<img src="/assets/icon.svg" alt="Logo">
			RDP Gateway
		</div>
		<div class="user-info">
			<div class="lang-switch" aria-label="Language switcher">{{range .LanguageLinks}}<a class="lang-switch-link{{if .Current}} is-active{{end}}" href="{{.URL}}">{{.Label}}</a>{{end}}</div>
			<span id="adminUsername">{{.Messages.LoadingUser}}</span>
			<a class="admin-link" href="/">{{.Messages.BackToDashboard}}</a>
		</div>
	</div>
	<main class="main">
		<div class="container admin-container">
			<h1 class="title">{{.Messages.Heading}}</h1>
			<p class="subtitle">{{.Messages.Subtitle}}</p>
			<div class="success" id="adminSuccess"></div>
			<div class="error" id="adminError"></div>
			<div class="admin-grid">
			<section class="admin-panel">
				<h2>{{.Messages.CreateHostEntryHeading}}</h2>
				<form id="hostForm" class="stack-form">
					<label>{{.Messages.NameLabel}}<input type="text" name="name" required></label>
					<label>{{.Messages.DescriptionLabel}}<input type="text" name="description"></label>
					<label>{{.Messages.AllowedGroupsLabel}}<input type="text" name="allowedGroups" placeholder="{{.Messages.AllowedGroupsPlaceholder}}" required></label>
					<label>{{.Messages.HostLabel}}<input type="text" name="host" placeholder="{{.Messages.HostPlaceholder}}" required></label>
					<button type="submit" class="primary-button">{{.Messages.CreateHostEntryButton}}</button>
				</form>
			</section>
			<section class="admin-panel">
				<h2>{{.Messages.CreateTemplateEntryHeading}}</h2>
				<form id="templateForm" class="stack-form" enctype="multipart/form-data">
					<label>{{.Messages.NameLabel}}<input type="text" name="name" required></label>
					<label>{{.Messages.DescriptionLabel}}<input type="text" name="description"></label>
					<label>{{.Messages.AllowedGroupsLabel}}<input type="text" name="allowedGroups" placeholder="{{.Messages.AllowedGroupsPlaceholder}}" required></label>
					<label>{{.Messages.TargetHostOverrideLabel}}<input type="text" name="targetHostOverride" placeholder="{{.Messages.TargetHostOverridePlaceholder}}"></label>
					<label>{{.Messages.RdpTemplateFileLabel}}<input type="file" name="template" accept=".rdp" required></label>
					<button type="submit" class="primary-button">{{.Messages.UploadTemplateEntryButton}}</button>
				</form>
			</section>
			<section class="admin-panel">
				<h2>{{.Messages.CreateDirectAuthUserHeading}}</h2>
				<form id="authUserForm" class="stack-form">
					<label>{{.Messages.UsernameLabel}}<input type="text" name="username" required></label>
					<label>{{.Messages.PasswordLabel}}<input type="password" name="password" required></label>
					<label><input type="checkbox" name="enabled" checked> {{.Messages.EnabledLabel}}</label>
					<button type="submit" class="primary-button">{{.Messages.CreateDirectAuthUserButton}}</button>
				</form>
			</section>
			</div>
			<section class="admin-panel">
				<h2>{{.Messages.CurrentEntriesHeading}}</h2>
				<div id="adminEntries"></div>
			</section>
			<section class="admin-panel">
				<h2>{{.Messages.CurrentDirectAuthUsersHeading}}</h2>
				<div id="adminAuthUsers"></div>
			</section>
		</div>
	</main>
	<script>window.adminMessages = {{.MessagesJSON}};</script>
	<script src="/static/admin.js"></script>
</body>
</html>
`
