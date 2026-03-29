package web

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/rdp"
	"github.com/gorilla/mux"
)

func (h *Handler) HandleEntryDownload(w http.ResponseWriter, r *http.Request) {
	id := identity.FromRequestCtx(r)
	if id == nil || !id.Authenticated() {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if h.dashboardStore == nil {
		http.Error(w, "dashboard store not configured", http.StatusServiceUnavailable)
		return
	}

	entryID := mux.Vars(r)["id"]
	if entryID == "" {
		http.Error(w, "missing entry id", http.StatusBadRequest)
		return
	}

	entry, err := h.dashboardStore.Get(entryID)
	if err != nil {
		if errors.Is(err, dashboard.ErrEntryNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("Cannot load dashboard entry %s due to %s", entryID, err)
		http.Error(w, "unable to load entry", http.StatusInternalServerError)
		return
	}

	visible := dashboard.VisibleEntries([]dashboard.Entry{entry}, id.Groups())
	if len(visible) != 1 {
		http.NotFound(w, r)
		return
	}

	builder, host, err := h.buildEntryBuilder(id, entry)
	if err != nil {
		log.Printf("Cannot build RDP for entry %s due to %s", entryID, err)
		http.Error(w, "unable to build RDP file", http.StatusInternalServerError)
		return
	}

	username := id.UserName()
	token, err := h.paaTokenGenerator(r.Context(), username, host)
	if err != nil {
		log.Printf("Cannot generate PAA token for user %s due to %s", username, err)
		http.Error(w, "unable to generate gateway credentials", http.StatusInternalServerError)
		return
	}

	if !h.rdpOpts.NoUsername {
		renderedUsername := username
		if h.rdpOpts.UsernameTemplate != "" {
			renderedUsername = h.rdpOpts.UsernameTemplate
			renderedUsername = strings.Replace(renderedUsername, "{{ username }}", username, 1)
			if renderedUsername == h.rdpOpts.UsernameTemplate {
				http.Error(w, "invalid server configuration", http.StatusInternalServerError)
				return
			}
		}
		builder.Settings.Username = renderedUsername
	}

	builder.Settings.FullAddress = host
	builder.Settings.GatewayHostname = h.gatewayAddress.Host
	builder.Settings.GatewayCredentialsSource = rdp.SourceCookie
	builder.Settings.GatewayAccessToken = token
	builder.Settings.GatewayCredentialMethod = 1
	builder.Settings.GatewayUsageMethod = 1

	h.serveBuiltRDP(w, r, entry.ID, builder)
}

func (h *Handler) buildEntryBuilder(id identity.Identity, entry dashboard.Entry) (*rdp.Builder, string, error) {
	switch entry.Type {
	case dashboard.EntryTypeHost:
		host := strings.Replace(entry.Host, "{{ preferred_username }}", id.UserName(), 1)
		if h.rdpDefaults == "" {
			return rdp.NewBuilder(), host, nil
		}
		builder, err := rdp.NewBuilderFromFile(h.rdpDefaults)
		if err != nil {
			return nil, "", err
		}
		return builder, host, nil
	case dashboard.EntryTypeTemplate:
		uploadedPath := h.dashboardStore.ResolveUpload(entry.UploadedTemplatePath)
		builder, err := rdp.NewBuilderFromFile(uploadedPath)
		if err != nil {
			return nil, "", err
		}
		host := builder.Settings.FullAddress
		if strings.TrimSpace(entry.TargetHostOverride) != "" {
			host = strings.TrimSpace(entry.TargetHostOverride)
		}
		return builder, host, nil
	default:
		return nil, "", errors.New("unsupported dashboard entry type")
	}
}

func (h *Handler) serveBuiltRDP(w http.ResponseWriter, r *http.Request, entryID string, builder *rdp.Builder) {
	filename := filepath.Base(entryID)
	if !strings.HasSuffix(strings.ToLower(filename), ".rdp") {
		filename += ".rdp"
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filename)
	w.Header().Set("Content-Type", "application/x-rdp")

	if h.rdpSigner == nil {
		http.ServeContent(w, r, filename, time.Now(), strings.NewReader(builder.String()))
		return
	}

	signedContent, err := h.rdpSigner.Sign(builder.String())
	if err != nil {
		log.Printf("Could not sign RDP file due to %s", err)
		http.Error(w, "could not sign RDP file", http.StatusInternalServerError)
		return
	}
	http.ServeContent(w, r, filename, time.Now(), bytes.NewReader(signedContent))
}
