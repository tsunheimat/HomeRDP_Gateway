package web

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	rdpparser "github.com/bolkedebruin/rdpgw/cmd/rdpgw/rdp/koanf/parsers/rdp"
	"github.com/gorilla/mux"
)

type adminCreateHostEntryRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	AllowedGroups []string `json:"allowedGroups"`
	Host          string   `json:"host"`
}

type adminUpdateEntryRequest struct {
	Name               *string   `json:"name"`
	Description        *string   `json:"description"`
	AllowedGroups      *[]string `json:"allowedGroups"`
	Enabled            *bool     `json:"enabled"`
	Host               *string   `json:"host"`
	TargetHostOverride *string   `json:"targetHostOverride"`
}

func (h *Handler) HandleAdminListEntries(w http.ResponseWriter, r *http.Request) {
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	entries, err := h.dashboardStore.List()
	if err != nil {
		http.Error(w, "unable to list dashboard entries", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entries)
}

func (h *Handler) HandleAdminCreateHostEntry(w http.ResponseWriter, r *http.Request) {
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	var req adminCreateHostEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	entry := dashboard.Entry{
		ID:            newEntryID(),
		Type:          dashboard.EntryTypeHost,
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		AllowedGroups: req.AllowedGroups,
		Enabled:       true,
		Host:          strings.TrimSpace(req.Host),
	}

	if err := h.dashboardStore.Put(entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) HandleAdminCreateTemplateEntry(w http.ResponseWriter, r *http.Request) {
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	maxUploadBytes := h.dashboardMaxUploadBytes
	if maxUploadBytes <= 0 {
		maxUploadBytes = 10 << 20
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		http.Error(w, "invalid multipart form", http.StatusBadRequest)
		return
	}

	upload, header, err := r.FormFile("template")
	if err != nil {
		http.Error(w, "template upload is required", http.StatusBadRequest)
		return
	}
	defer upload.Close()

	if strings.ToLower(filepath.Ext(header.Filename)) != ".rdp" {
		http.Error(w, "template must be an .rdp file", http.StatusBadRequest)
		return
	}

	data, err := io.ReadAll(upload)
	if err != nil {
		http.Error(w, "unable to read template upload", http.StatusBadRequest)
		return
	}

	if _, err := rdpparser.Parser().Unmarshal(data); err != nil {
		http.Error(w, "invalid rdp template upload", http.StatusBadRequest)
		return
	}

	uploadedPath, err := h.dashboardStore.SaveUpload(bytes.NewReader(data))
	if err != nil {
		http.Error(w, "unable to save template upload", http.StatusInternalServerError)
		return
	}

	entry := dashboard.Entry{
		ID:                   newEntryID(),
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 strings.TrimSpace(r.FormValue("name")),
		Description:          strings.TrimSpace(r.FormValue("description")),
		AllowedGroups:        splitCSV(r.FormValue("allowedGroups")),
		Enabled:              true,
		UploadedTemplatePath: uploadedPath,
		TargetHostOverride:   strings.TrimSpace(r.FormValue("targetHostOverride")),
	}

	if err := h.dashboardStore.Put(entry); err != nil {
		uploadFilePath := h.dashboardStore.ResolveUpload(uploadedPath)
		if removeErr := os.Remove(uploadFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			http.Error(w, "unable to persist template entry", http.StatusInternalServerError)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) HandleAdminUpdateEntry(w http.ResponseWriter, r *http.Request) {
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	entryID := mux.Vars(r)["id"]
	if strings.TrimSpace(entryID) == "" {
		http.Error(w, "missing entry id", http.StatusBadRequest)
		return
	}

	entry, err := h.dashboardStore.Get(entryID)
	if err != nil {
		if errors.Is(err, dashboard.ErrEntryNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "unable to load dashboard entry", http.StatusInternalServerError)
		return
	}

	var req adminUpdateEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name != nil {
		entry.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		entry.Description = strings.TrimSpace(*req.Description)
	}
	if req.AllowedGroups != nil {
		entry.AllowedGroups = *req.AllowedGroups
	}
	if req.Enabled != nil {
		entry.Enabled = *req.Enabled
	}
	if req.Host != nil && entry.Type == dashboard.EntryTypeHost {
		entry.Host = strings.TrimSpace(*req.Host)
	}
	if req.TargetHostOverride != nil && entry.Type == dashboard.EntryTypeTemplate {
		entry.TargetHostOverride = strings.TrimSpace(*req.TargetHostOverride)
	}

	if err := h.dashboardStore.Put(entry); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(entry)
}

func (h *Handler) HandleAdminDeleteEntry(w http.ResponseWriter, r *http.Request) {
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	entryID := mux.Vars(r)["id"]
	if strings.TrimSpace(entryID) == "" {
		http.Error(w, "missing entry id", http.StatusBadRequest)
		return
	}

	if err := h.dashboardStore.Delete(entryID); err != nil {
		http.Error(w, "unable to delete dashboard entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func newEntryID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}
