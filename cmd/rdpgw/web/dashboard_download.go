package web

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/dashboard"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/identity"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/rdp"
	"github.com/bolkedebruin/rdpgw/cmd/rdpgw/security"
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

	user := id.UserName()
	domain := ""
	if h.rdpOpts.SplitUserDomain {
		parts := strings.SplitN(id.UserName(), "@", 2)
		user = parts[0]
		if len(parts) > 1 {
			domain = parts[1]
		}
	}

	render := user
	if h.rdpOpts.UsernameTemplate != "" {
		render = fmt.Sprint(h.rdpOpts.UsernameTemplate)
		render = strings.Replace(render, "{{ username }}", user, 1)
		if render == h.rdpOpts.UsernameTemplate {
			http.Error(w, "invalid server configuration", http.StatusInternalServerError)
			return
		}
	}

	token, err := h.paaTokenGenerator(r.Context(), user, host)
	if err != nil {
		log.Printf("Cannot generate PAA token for user %s due to %s", user, err)
		http.Error(w, "unable to generate gateway credentials", http.StatusInternalServerError)
		return
	}

	if h.enableUserToken {
		userToken, err := h.userTokenGenerator(r.Context(), user)
		if err != nil {
			log.Printf("Cannot generate token for user %s due to %s", user, err)
			http.Error(w, "unable to generate gateway credentials", http.StatusInternalServerError)
			return
		}
		render = strings.Replace(render, "{{ token }}", userToken, 1)
	}

	if !h.rdpOpts.NoUsername {
		builder.Settings.Username = render
		builder.Settings.Domain = domain
	} else {
		builder.Settings.Username = ""
		builder.Settings.Domain = ""
	}

	builder.Settings.FullAddress = host
	builder.Settings.GatewayHostname = h.gatewayAddress.Host
	builder.Settings.GatewayCredentialsSource = rdp.SourceCookie
	builder.Settings.GatewayAccessToken = token
	builder.Settings.GatewayCredentialMethod = 1
	builder.Settings.GatewayUsageMethod = 1
	h.applyRdpRedirectionPolicy(builder)

	h.serveBuiltRDP(w, r, entry.ID, builder)
}

func (h *Handler) buildEntryBuilder(id identity.Identity, entry dashboard.Entry) (*rdp.Builder, string, error) {
	switch entry.Type {
	case dashboard.EntryTypeHost:
		host, err := security.ResolvePreferredUsernameHost(entry.Host, id.UserName())
		if err != nil {
			return nil, "", err
		}
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
		host, err = security.ResolvePreferredUsernameHost(host, id.UserName())
		if err != nil {
			if strings.TrimSpace(host) == "" {
				return nil, "", errors.New("template entry does not resolve to a target host")
			}
			return nil, "", err
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
