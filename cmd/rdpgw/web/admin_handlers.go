package web

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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

type adminEntryResponse struct {
	ID                  string    `json:"id"`
	Type                string    `json:"type"`
	Name                string    `json:"name"`
	Description         string    `json:"description"`
	AllowedGroups       []string  `json:"allowedGroups"`
	Enabled             bool      `json:"enabled"`
	Host                string    `json:"host"`
	TargetHostOverride  string    `json:"targetHostOverride"`
	HasUploadedTemplate bool      `json:"hasUploadedTemplate"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type adminCreateAuthUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Enabled  *bool  `json:"enabled"`
}

type adminUpdateAuthUserRequest struct {
	Password *string `json:"password"`
	Enabled  *bool   `json:"enabled"`
}

type adminAuthUserResponse struct {
	Username  string    `json:"username"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	_ = json.NewEncoder(w).Encode(adminEntryResponses(entries))
}

func (h *Handler) HandleAdminCreateHostEntry(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	var req adminCreateHostEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	id, err := newEntryID()
	if err != nil {
		http.Error(w, "unable to generate entry id", http.StatusInternalServerError)
		return
	}

	entry := dashboard.Entry{
		ID:            id,
		Type:          dashboard.EntryTypeHost,
		Name:          strings.TrimSpace(req.Name),
		Description:   strings.TrimSpace(req.Description),
		AllowedGroups: req.AllowedGroups,
		Enabled:       true,
		Host:          strings.TrimSpace(req.Host),
	}

	if err := h.dashboardStore.Put(entry); err != nil {
		http.Error(w, err.Error(), statusForStorePutError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newAdminEntryResponse(entry))
}

func (h *Handler) HandleAdminCreateTemplateEntry(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
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
	uploadFilePath := h.dashboardStore.ResolveUpload(uploadedPath)

	id, err := newEntryID()
	if err != nil {
		_ = os.Remove(uploadFilePath)
		http.Error(w, "unable to generate entry id", http.StatusInternalServerError)
		return
	}

	entry := dashboard.Entry{
		ID:                   id,
		Type:                 dashboard.EntryTypeTemplate,
		Name:                 strings.TrimSpace(r.FormValue("name")),
		Description:          strings.TrimSpace(r.FormValue("description")),
		AllowedGroups:        splitCSV(r.FormValue("allowedGroups")),
		Enabled:              true,
		UploadedTemplatePath: uploadedPath,
		TargetHostOverride:   strings.TrimSpace(r.FormValue("targetHostOverride")),
	}

	if err := h.dashboardStore.Put(entry); err != nil {
		if removeErr := os.Remove(uploadFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			http.Error(w, "unable to persist template entry", http.StatusInternalServerError)
			return
		}
		http.Error(w, err.Error(), statusForStorePutError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newAdminEntryResponse(entry))
}

func (h *Handler) HandleAdminUpdateEntry(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
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
		http.Error(w, err.Error(), statusForStorePutError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(newAdminEntryResponse(entry))
}

func (h *Handler) HandleAdminDeleteEntry(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	entryID := mux.Vars(r)["id"]
	if strings.TrimSpace(entryID) == "" {
		http.Error(w, "missing entry id", http.StatusBadRequest)
		return
	}

	if _, err := h.dashboardStore.Get(entryID); err != nil {
		if errors.Is(err, dashboard.ErrEntryNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "unable to load dashboard entry", http.StatusInternalServerError)
		return
	}

	if err := h.dashboardStore.Delete(entryID); err != nil {
		http.Error(w, "unable to delete dashboard entry", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) HandleAdminListAuthUsers(w http.ResponseWriter, r *http.Request) {
	if h.dashboardAuthUserStore == nil {
		http.Error(w, "auth user store not configured", http.StatusServiceUnavailable)
		return
	}

	users, err := h.dashboardAuthUserStore.List()
	if err != nil {
		http.Error(w, "unable to list auth users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminAuthUserResponses(users))
}

func (h *Handler) HandleAdminCreateAuthUser(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardAuthUserStore == nil || h.authHelperConfigPath == "" {
		http.Error(w, "auth user management not configured", http.StatusServiceUnavailable)
		return
	}

	var req adminCreateAuthUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	user := dashboard.AuthUser{
		Username: strings.TrimSpace(req.Username),
		Password: req.Password,
		Enabled:  enabled,
	}
	if err := h.dashboardAuthUserStore.Put(user); err != nil {
		http.Error(w, err.Error(), statusForStorePutError(err))
		return
	}
	if err := h.regenerateAuthHelperConfig(); err != nil {
		http.Error(w, "unable to write auth helper config", http.StatusInternalServerError)
		return
	}

	stored, err := h.dashboardAuthUserStore.Get(user.Username)
	if err != nil {
		http.Error(w, "unable to load auth user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newAdminAuthUserResponse(stored))
}

func (h *Handler) HandleAdminUpdateAuthUser(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardAuthUserStore == nil || h.authHelperConfigPath == "" {
		http.Error(w, "auth user management not configured", http.StatusServiceUnavailable)
		return
	}

	username := strings.TrimSpace(mux.Vars(r)["username"])
	if username == "" {
		http.Error(w, "missing username", http.StatusBadRequest)
		return
	}

	user, err := h.dashboardAuthUserStore.Get(username)
	if err != nil {
		if errors.Is(err, dashboard.ErrAuthUserNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "unable to load auth user", http.StatusInternalServerError)
		return
	}

	var req adminUpdateAuthUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Password != nil {
		user.Password = *req.Password
	}
	if req.Enabled != nil {
		user.Enabled = *req.Enabled
	}

	if err := h.dashboardAuthUserStore.Put(user); err != nil {
		http.Error(w, err.Error(), statusForStorePutError(err))
		return
	}
	if err := h.regenerateAuthHelperConfig(); err != nil {
		http.Error(w, "unable to write auth helper config", http.StatusInternalServerError)
		return
	}

	updated, err := h.dashboardAuthUserStore.Get(username)
	if err != nil {
		http.Error(w, "unable to load auth user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(newAdminAuthUserResponse(updated))
}

func (h *Handler) HandleAdminDeleteAuthUser(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardAuthUserStore == nil || h.authHelperConfigPath == "" {
		http.Error(w, "auth user management not configured", http.StatusServiceUnavailable)
		return
	}

	username := strings.TrimSpace(mux.Vars(r)["username"])
	if username == "" {
		http.Error(w, "missing username", http.StatusBadRequest)
		return
	}

	if _, err := h.dashboardAuthUserStore.Get(username); err != nil {
		if errors.Is(err, dashboard.ErrAuthUserNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "unable to load auth user", http.StatusInternalServerError)
		return
	}
	if err := h.dashboardAuthUserStore.Delete(username); err != nil {
		http.Error(w, "unable to delete auth user", http.StatusInternalServerError)
		return
	}
	if err := h.regenerateAuthHelperConfig(); err != nil {
		http.Error(w, "unable to write auth helper config", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func newEntryID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

func statusForStorePutError(err error) int {
	if dashboard.IsValidationError(err) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func sameOriginAdminRequest(r *http.Request) bool {
	if target, ok := requestOriginURL(r); ok {
		return sameOriginHost(target, r.Host)
	}
	if target, ok := requestRefererURL(r); ok {
		return sameOriginHost(target, r.Host)
	}
	return false
}

func requestOriginURL(r *http.Request) (*url.URL, bool) {
	rawOrigin := strings.TrimSpace(r.Header.Get("Origin"))
	if rawOrigin == "" || rawOrigin == "null" {
		return nil, false
	}
	parsed, err := url.Parse(rawOrigin)
	if err != nil || parsed.Host == "" {
		return nil, false
	}
	return parsed, true
}

func requestRefererURL(r *http.Request) (*url.URL, bool) {
	rawReferer := strings.TrimSpace(r.Referer())
	if rawReferer == "" {
		return nil, false
	}
	parsed, err := url.Parse(rawReferer)
	if err != nil || parsed.Host == "" {
		return nil, false
	}
	return parsed, true
}

func sameOriginHost(target *url.URL, requestHost string) bool {
	if target == nil {
		return false
	}
	return strings.EqualFold(target.Host, requestHost)
}

func adminEntryResponses(entries []dashboard.Entry) []adminEntryResponse {
	out := make([]adminEntryResponse, 0, len(entries))
	for _, entry := range entries {
		out = append(out, newAdminEntryResponse(entry))
	}
	return out
}

func adminAuthUserResponses(users []dashboard.AuthUser) []adminAuthUserResponse {
	out := make([]adminAuthUserResponse, 0, len(users))
	for _, user := range users {
		out = append(out, newAdminAuthUserResponse(user))
	}
	return out
}

func newAdminEntryResponse(entry dashboard.Entry) adminEntryResponse {
	return adminEntryResponse{
		ID:                  entry.ID,
		Type:                string(entry.Type),
		Name:                entry.Name,
		Description:         entry.Description,
		AllowedGroups:       entry.AllowedGroups,
		Enabled:             entry.Enabled,
		Host:                entry.Host,
		TargetHostOverride:  entry.TargetHostOverride,
		HasUploadedTemplate: strings.TrimSpace(entry.UploadedTemplatePath) != "",
		CreatedAt:           entry.CreatedAt,
		UpdatedAt:           entry.UpdatedAt,
	}
}

func newAdminAuthUserResponse(user dashboard.AuthUser) adminAuthUserResponse {
	return adminAuthUserResponse{
		Username:  user.Username,
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

func (h *Handler) regenerateAuthHelperConfig() error {
	users, err := h.dashboardAuthUserStore.List()
	if err != nil {
		return err
	}
	return dashboard.WriteAuthHelperConfig(h.authHelperConfigPath, users)
}
