package dashboard

import (
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

type Entry struct {
	ID                   string    `json:"id"`
	Type                 EntryType `json:"type"`
	Name                 string    `json:"name"`
	Description          string    `json:"description"`
	AllowedGroups        []string  `json:"allowedGroups"`
	Enabled              bool      `json:"enabled"`
	Host                 string    `json:"host"`
	UploadedTemplatePath string    `json:"uploadedTemplatePath"`
	TargetHostOverride   string    `json:"targetHostOverride"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

func (e Entry) Validate() error {
	if strings.TrimSpace(e.ID) == "" {
		return fmt.Errorf("id is required")
	}
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("name is required")
	}

	validGroups := 0
	for _, group := range e.AllowedGroups {
		if strings.TrimSpace(group) != "" {
			validGroups++
		}
	}
	if validGroups == 0 {
		return fmt.Errorf("at least one allowed group is required")
	}

	switch e.Type {
	case EntryTypeHost:
		host, port, err := net.SplitHostPort(strings.TrimSpace(e.Host))
		if err != nil || strings.TrimSpace(host) == "" {
			return fmt.Errorf("host entry requires a valid host:port")
		}
		portNum, err := strconv.Atoi(port)
		if err != nil || portNum < 1 || portNum > 65535 {
			return fmt.Errorf("host entry requires a valid host:port")
		}
	case EntryTypeTemplate:
		if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(e.UploadedTemplatePath)), ".rdp") {
			return fmt.Errorf("template entry requires an .rdp uploaded template path")
		}
	default:
		return fmt.Errorf("invalid entry type %q", e.Type)
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
