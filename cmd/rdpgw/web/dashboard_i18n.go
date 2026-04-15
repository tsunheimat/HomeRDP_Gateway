package web

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
)

const (
	dashboardLocaleEnglish            = "en"
	dashboardLocaleTraditionalChinese = "zh-Hant"
)

type dashboardMessages struct {
	AdminLink       string            `json:"adminLink"`
	DownloadButton  string            `json:"downloadButton"`
	Downloading     string            `json:"downloading"`
	EmptyState      string            `json:"emptyState"`
	EntryTypes      map[string]string `json:"entryTypes"`
	Heading         string            `json:"heading"`
	LoadErrorPrefix string            `json:"loadErrorPrefix"`
	LoadingEntries  string            `json:"loadingEntries"`
	LoadingUser     string            `json:"loadingUser"`
	NoDescription   string            `json:"noDescription"`
	Subtitle        string            `json:"subtitle"`
	UnknownUser     string            `json:"unknownUser"`
	WindowTitle     string            `json:"windowTitle"`
}

func dashboardLocaleFromRequest(r *http.Request) string {
	if locale, ok := normalizeDashboardLocale(r.URL.Query().Get("lang")); ok {
		return locale
	}

	return dashboardLocaleEnglish
}

func normalizeDashboardLocale(value string) (string, bool) {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), "_", "-"))
	switch {
	case normalized == "":
		return "", false
	case strings.HasPrefix(normalized, "en"):
		return dashboardLocaleEnglish, true
	case strings.HasPrefix(normalized, "zh-hant"),
		strings.HasPrefix(normalized, "zh-tw"),
		strings.HasPrefix(normalized, "zh-hk"),
		strings.HasPrefix(normalized, "zh-mo"):
		return dashboardLocaleTraditionalChinese, true
	default:
		return "", false
	}
}

func dashboardMessagesForLocale(locale string) dashboardMessages {
	if locale == dashboardLocaleTraditionalChinese {
		return dashboardMessages{
			AdminLink:      "管理",
			DownloadButton: "下載 RDP",
			Downloading:    "正在下載 %s。",
			EmptyState:     "目前沒有可供您群組使用的項目。",
			EntryTypes: map[string]string{
				"host":     "主機",
				"template": "範本",
			},
			Heading:         "連線儀表板",
			LoadErrorPrefix: "無法載入儀表板：",
			LoadingEntries:  "正在載入儀表板項目...",
			LoadingUser:     "載入中...",
			NoDescription:   "未提供描述。",
			Subtitle:        "選取已發布的項目並下載其 RDP 檔案。",
			UnknownUser:     "未知使用者",
			WindowTitle:     "RDP Gateway 儀表板",
		}
	}

	return dashboardMessages{
		AdminLink:      "Admin",
		DownloadButton: "Download RDP",
		Downloading:    "Downloading %s.",
		EmptyState:     "No entries are currently available for your groups.",
		EntryTypes: map[string]string{
			"host":     "Host",
			"template": "Template",
		},
		Heading:         "Connection Dashboard",
		LoadErrorPrefix: "Unable to load dashboard:",
		LoadingEntries:  "Loading dashboard entries...",
		LoadingUser:     "Loading...",
		NoDescription:   "No description provided.",
		Subtitle:        "Select a published entry and download its RDP file.",
		UnknownUser:     "Unknown User",
		WindowTitle:     "RDP Gateway Dashboard",
	}
}

func dashboardMessagesJSON(messages dashboardMessages) template.JS {
	encoded, err := json.Marshal(messages)
	if err != nil {
		return template.JS(`{}`)
	}

	encoded = bytes.ReplaceAll(encoded, []byte("</"), []byte("<\\/"))
	encoded = bytes.ReplaceAll(encoded, []byte("\u2028"), []byte("\\u2028"))
	encoded = bytes.ReplaceAll(encoded, []byte("\u2029"), []byte("\\u2029"))
	return template.JS(encoded)
}

func dashboardLanguageURL(r *http.Request, locale string) string {
	query := r.URL.Query()
	query.Set("lang", locale)
	encoded := query.Encode()
	if encoded == "" {
		return r.URL.Path
	}
	return r.URL.Path + "?" + encoded
}

func dashboardLanguageLinks(r *http.Request, activeLocale string) []map[string]any {
	return []map[string]any{
		{
			"Current": activeLocale == dashboardLocaleEnglish,
			"Label":   "EN",
			"URL":     dashboardLanguageURL(r, dashboardLocaleEnglish),
		},
		{
			"Current": activeLocale == dashboardLocaleTraditionalChinese,
			"Label":   "繁中",
			"URL":     dashboardLanguageURL(r, dashboardLocaleTraditionalChinese),
		},
	}
}
