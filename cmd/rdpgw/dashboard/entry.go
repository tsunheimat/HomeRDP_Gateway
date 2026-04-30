package dashboard

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type EntryType string

const (
	EntryTypeHost     EntryType = "host"
	EntryTypeTemplate EntryType = "template"
)

var errValidation = errors.New("dashboard validation error")

type Entry struct {
	ID                    string    `json:"id"`
	Type                  EntryType `json:"type"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	Icon                  string    `json:"icon"`
	AllowedGroups         []string  `json:"allowedGroups"`
	Enabled               bool      `json:"enabled"`
	Host                  string    `json:"host"`
	UploadedTemplatePath  string    `json:"uploadedTemplatePath"`
	TargetHostOverride    string    `json:"targetHostOverride"`
	TargetIPOverride      string    `json:"targetIPOverride"`
	ForceTargetIPOverride bool      `json:"forceTargetIPOverride"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

func (e Entry) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return validationError("id is required")
	}
	if strings.TrimSpace(e.Name) == "" {
		return validationError("name is required")
	}
	if !IsValidEntryIcon(e.Icon) {
		return validationError(fmt.Sprintf("invalid entry icon %q", e.Icon))
	}

	validGroups := 0
	for _, group := range e.AllowedGroups {
		if strings.TrimSpace(group) != "" {
			validGroups++
		}
	}
	if validGroups == 0 {
		return validationError("at least one allowed group is required")
	}
	if override := strings.TrimSpace(e.TargetIPOverride); override != "" && net.ParseIP(override) == nil {
		return validationError("target IP override must be an IP literal")
	}

	switch e.Type {
	case EntryTypeHost:
		host, port, err := net.SplitHostPort(strings.TrimSpace(e.Host))
		if err != nil || strings.TrimSpace(host) == "" {
			return validationError("host entry requires a valid host:port")
		}
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return validationError("host entry requires a valid host:port")
		}
	case EntryTypeTemplate:
		if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(e.UploadedTemplatePath)), ".rdp") {
			return validationError("template entry requires an .rdp uploaded template path")
		}
	default:
		return validationError(fmt.Sprintf("invalid entry type %q", e.Type))
	}

	return nil
}

func VisibleEntries(entries []Entry, groups []string) []Entry {
	allowed := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		allowed[group] = struct{}{}
	}

	visible := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if !entry.Enabled {
			continue
		}
		for _, group := range entry.AllowedGroups {
			group = strings.TrimSpace(group)
			if group == "" {
				continue
			}
			if _, ok := allowed[group]; ok {
				visible = append(visible, entry)
				break
			}
		}
	}
	return visible
}

func IsValidationError(err error) bool {
	return errors.Is(err, errValidation)
}

func validationError(message string) error {
	return fmt.Errorf("%w: %s", errValidation, message)
}
