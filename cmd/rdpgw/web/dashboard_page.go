package web

import (
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
		http.Redirect(w, r, "/connect", http.StatusFound)
		return
	}

	h.renderTemplatePage(w, "dashboard.html", fallbackDashboardTemplate, map[string]any{
		"Title": "RDP Gateway Dashboard",
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

	h.renderTemplatePage(w, "admin.html", fallbackAdminTemplate, map[string]any{
		"Title": "RDP Gateway Admin",
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
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Failed to execute template %s: %v", filename, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
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
<html lang="en">
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
			<span id="dashboardUser">Loading...</span>
			<a class="admin-link" id="adminLink" href="/admin" hidden>Admin</a>
		</div>
	</div>
	<main class="main">
		<div class="container">
			<h1 class="title">Connection Dashboard</h1>
			<div class="error" id="dashboardError"></div>
			<div class="entries-grid" id="entriesGrid"></div>
		</div>
	</main>
	<script src="/static/dashboard.js"></script>
</body>
</html>
`

const fallbackAdminTemplate = `<!DOCTYPE html>
<html lang="en">
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
			RDP Gateway Admin
		</div>
		<div class="user-info">
			<a class="admin-link" href="/">Dashboard</a>
		</div>
	</div>
	<main class="main">
		<div class="container">
			<h1 class="title">Admin</h1>
			<div class="error" id="adminError"></div>
			<section class="admin-panel">
				<h2>Host Entry</h2>
				<form id="hostForm"></form>
			</section>
			<section class="admin-panel">
				<h2>Template Entry</h2>
				<form id="templateForm"></form>
			</section>
			<section class="admin-panel">
				<h2>Entries</h2>
				<div id="adminEntries"></div>
			</section>
		</div>
	</main>
	<script src="/static/admin.js"></script>
</body>
</html>
`
