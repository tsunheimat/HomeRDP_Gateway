package web

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/gorilla/mux"
)

type adminIconResponse struct {
	ID           string    `json:"id"`
	Filename     string    `json:"filename"`
	OriginalName string    `json:"originalName"`
	ContentType  string    `json:"contentType"`
	Active       bool      `json:"active"`
	URL          string    `json:"url"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type adminSelectIconRequest struct {
	ID string `json:"id"`
}

func (h *Handler) HandleAdminListIcons(w http.ResponseWriter, r *http.Request) {
	if h.dashboardIconStore == nil {
		http.Error(w, "dashboard icon store not configured", http.StatusServiceUnavailable)
		return
	}

	icons, err := h.dashboardIconStore.ListIcons()
	if err != nil {
		http.Error(w, "unable to list icons", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(adminIconResponses(icons))
}

func (h *Handler) HandleAdminUploadIcon(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardIconStore == nil {
		http.Error(w, "dashboard icon store not configured", http.StatusServiceUnavailable)
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

	upload, header, err := r.FormFile("icon")
	if err != nil {
		http.Error(w, "icon upload is required", http.StatusBadRequest)
		return
	}
	defer upload.Close()

	data, err := io.ReadAll(upload)
	if err != nil {
		http.Error(w, "unable to read icon upload", http.StatusBadRequest)
		return
	}

	icon, err := h.dashboardIconStore.SaveIcon(header.Filename, data)
	if err != nil {
		http.Error(w, err.Error(), statusForIconStoreError(err))
		return
	}
	if err := h.dashboardIconStore.SetActiveIcon(icon.ID); err != nil {
		http.Error(w, "unable to activate uploaded icon", statusForIconStoreError(err))
		return
	}
	icon.Active = true

	if wantsHTMLAdminResponse(r) {
		http.Redirect(w, r, "/admin?section=branding", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(newAdminIconResponse(icon))
}

func (h *Handler) HandleAdminSelectIcon(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardIconStore == nil {
		http.Error(w, "dashboard icon store not configured", http.StatusServiceUnavailable)
		return
	}

	var req adminSelectIconRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.dashboardIconStore.SetActiveIcon(req.ID); err != nil {
		http.Error(w, err.Error(), statusForIconStoreError(err))
		return
	}

	active, err := h.dashboardIconStore.ActiveIcon()
	if err != nil {
		http.Error(w, "unable to load active icon", statusForIconStoreError(err))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(newAdminIconResponse(active))
}

func (h *Handler) HandleAdminDeleteIcon(w http.ResponseWriter, r *http.Request) {
	if !sameOriginAdminRequest(r) {
		http.Error(w, "cross-origin admin request forbidden", http.StatusForbidden)
		return
	}
	if h.dashboardIconStore == nil {
		http.Error(w, "dashboard icon store not configured", http.StatusServiceUnavailable)
		return
	}

	iconID := mux.Vars(r)["id"]
	if strings.TrimSpace(iconID) == "" {
		http.Error(w, "missing icon id", http.StatusBadRequest)
		return
	}

	if err := h.dashboardIconStore.DeleteIcon(iconID); err != nil {
		http.Error(w, err.Error(), statusForIconStoreError(err))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ServeAppIcon(w http.ResponseWriter, r *http.Request) {
	if h.dashboardIconStore != nil {
		icon, err := h.dashboardIconStore.ActiveIcon()
		if err == nil {
			path := h.dashboardIconStore.ResolveIcon(icon.Filename)
			if _, statErr := os.Stat(path); statErr == nil {
				w.Header().Set("Content-Type", icon.ContentType)
				w.Header().Set("Cache-Control", "no-cache")
				http.ServeFile(w, r, path)
				return
			}
		} else if !errors.Is(err, dashboard.ErrIconNotFound) {
			log.Printf("Failed to load active app icon: %v", err)
		}
	}

	h.ServeAssetFile("icon.svg").ServeHTTP(w, r)
}

func (h *Handler) ServeUploadedIcon(w http.ResponseWriter, r *http.Request) {
	if h.dashboardIconStore == nil {
		http.NotFound(w, r)
		return
	}

	iconID := strings.TrimSpace(mux.Vars(r)["id"])
	if iconID == "" {
		http.NotFound(w, r)
		return
	}

	icons, err := h.dashboardIconStore.ListIcons()
	if err != nil {
		http.Error(w, "unable to list icons", http.StatusInternalServerError)
		return
	}

	for _, icon := range icons {
		if icon.ID != iconID {
			continue
		}
		path := h.dashboardIconStore.ResolveIcon(icon.Filename)
		if _, statErr := os.Stat(path); statErr != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", icon.ContentType)
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeFile(w, r, path)
		return
	}

	http.NotFound(w, r)
}

func statusForIconStoreError(err error) int {
	if errors.Is(err, dashboard.ErrIconNotFound) {
		return http.StatusNotFound
	}
	if dashboard.IsValidationError(err) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func wantsHTMLAdminResponse(r *http.Request) bool {
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/html") && !strings.Contains(accept, "application/json")
}

func adminIconResponses(icons []dashboard.Icon) []adminIconResponse {
	out := make([]adminIconResponse, 0, len(icons))
	for _, icon := range icons {
		out = append(out, newAdminIconResponse(icon))
	}
	return out
}

func newAdminIconResponse(icon dashboard.Icon) adminIconResponse {
	return adminIconResponse{
		ID:           icon.ID,
		Filename:     icon.Filename,
		OriginalName: icon.OriginalName,
		ContentType:  icon.ContentType,
		Active:       icon.Active,
		URL:          "/assets/icons/" + icon.ID,
		CreatedAt:    icon.CreatedAt,
		UpdatedAt:    icon.UpdatedAt,
	}
}
