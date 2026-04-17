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
	AdminLink        string            `json:"adminLink"`
	DownloadButton   string            `json:"downloadButton"`
	Downloading      string            `json:"downloading"`
	EmptyState       string            `json:"emptyState"`
	EntryTypes       map[string]string `json:"entryTypes"`
	Heading          string            `json:"heading"`
	LoadErrorPrefix  string            `json:"loadErrorPrefix"`
	LoadingEntries   string            `json:"loadingEntries"`
	LoadingUser      string            `json:"loadingUser"`
	NoDescription     string            `json:"noDescription"`
	RetryButton       string            `json:"retryButton"`
	SearchPlaceholder string            `json:"searchPlaceholder"`
	Subtitle          string            `json:"subtitle"`
	SummaryHosts     string            `json:"summaryHosts"`
	SummaryTemplates string            `json:"summaryTemplates"`
	SummaryTotal     string            `json:"summaryTotal"`
	UnknownUser      string            `json:"unknownUser"`
	UploadedTemplateLabel string       `json:"uploadedTemplateLabel"`
	WindowTitle      string            `json:"windowTitle"`
}

type adminMessages struct {
	AllowedGroupsLabel             string            `json:"allowedGroupsLabel"`
	AllowedGroupsPlaceholder       string            `json:"allowedGroupsPlaceholder"`
	BackToDashboard                string            `json:"backToDashboard"`
	CreateDirectAuthUserButton     string            `json:"createDirectAuthUserButton"`
	CreateDirectAuthUserHeading    string            `json:"createDirectAuthUserHeading"`
	CreateAuthUserErrorPrefix      string            `json:"createAuthUserErrorPrefix"`
	CreateDirectAuthUserSuccess    string            `json:"createDirectAuthUserSuccess"`
	CreateHostEntryButton          string            `json:"createHostEntryButton"`
	CreateHostEntryErrorPrefix     string            `json:"createHostEntryErrorPrefix"`
	CreateHostEntryHeading         string            `json:"createHostEntryHeading"`
	CreateHostEntrySuccess         string            `json:"createHostEntrySuccess"`
	CreateTemplateEntryHeading     string            `json:"createTemplateEntryHeading"`
	CurrentDirectAuthUsersHeading  string            `json:"currentDirectAuthUsersHeading"`
	CurrentEntriesHeading          string            `json:"currentEntriesHeading"`
	DeleteButton                   string            `json:"deleteButton"`
	DeleteConfirm                  string            `json:"deleteConfirm"`
	DeleteAuthUserErrorPrefix      string            `json:"deleteAuthUserErrorPrefix"`
	DeleteEntryErrorPrefix         string            `json:"deleteEntryErrorPrefix"`
	DeleteSuccess                  string            `json:"deleteSuccess"`
	DescriptionLabel               string            `json:"descriptionLabel"`
	DisabledStatus                 string            `json:"disabledStatus"`
	EditButton                     string            `json:"editButton"`
	EnabledLabel                   string            `json:"enabledLabel"`
	EnabledStatus                  string            `json:"enabledStatus"`
	EntryTypes                     map[string]string `json:"entryTypes"`
	Heading                        string            `json:"heading"`
	HostLabel                      string            `json:"hostLabel"`
	HostPlaceholder                string            `json:"hostPlaceholder"`
	LoadAuthUsersErrorPrefix       string            `json:"loadAuthUsersErrorPrefix"`
	LoadEntriesErrorPrefix         string            `json:"loadEntriesErrorPrefix"`
	LoadUserErrorPrefix            string            `json:"loadUserErrorPrefix"`
	LoadingUser                    string            `json:"loadingUser"`
	NameLabel                      string            `json:"nameLabel"`
	NewPasswordLabel               string            `json:"newPasswordLabel"`
	NoAuthUsersConfigured          string            `json:"noAuthUsersConfigured"`
	NoDescription                  string            `json:"noDescription"`
	NoEntriesConfigured            string            `json:"noEntriesConfigured"`
	PasswordKeepPlaceholder        string            `json:"passwordKeepPlaceholder"`
	PasswordLabel                  string            `json:"passwordLabel"`
	RdpTemplateFileLabel           string            `json:"rdpTemplateFileLabel"`
	SaveButton                     string            `json:"saveButton"`
	SaveAuthUserErrorPrefix        string            `json:"saveAuthUserErrorPrefix"`
	SaveEntryErrorPrefix           string            `json:"saveEntryErrorPrefix"`
	SaveSuccess                    string            `json:"saveSuccess"`
	Subtitle                       string            `json:"subtitle"`
	TabDirectAuthUsers             string            `json:"tabDirectAuthUsers"`
	TabPublishedEntries            string            `json:"tabPublishedEntries"`
	TargetHostOverrideLabel        string            `json:"targetHostOverrideLabel"`
	TargetHostOverridePlaceholder  string            `json:"targetHostOverridePlaceholder"`
	ToggleEnabledButton            string            `json:"toggleEnabledButton"`
	UnknownUser                    string            `json:"unknownUser"`
	UploadedTemplateLabel          string            `json:"uploadedTemplateLabel"`
	UpdateAuthUserErrorPrefix      string            `json:"updateAuthUserErrorPrefix"`
	UpdateEntryErrorPrefix         string            `json:"updateEntryErrorPrefix"`
	UpdateSuccess                  string            `json:"updateSuccess"`
	UploadTemplateEntryButton      string            `json:"uploadTemplateEntryButton"`
	UploadTemplateEntryErrorPrefix string            `json:"uploadTemplateEntryErrorPrefix"`
	UploadTemplateEntrySuccess     string            `json:"uploadTemplateEntrySuccess"`
	UsernameLabel                  string            `json:"usernameLabel"`
	WindowTitle                    string            `json:"windowTitle"`
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
			LoadingEntries:    "正在載入儀表板項目...",
			LoadingUser:       "載入中...",
			NoDescription:     "未提供描述。",
			RetryButton:       "重試",
			SearchPlaceholder: "搜尋項目...",
			Subtitle:          "選取已發布的項目並下載其 RDP 檔案。",
			SummaryHosts:      "主機",
			SummaryTemplates:  "範本",
			SummaryTotal:      "總項目",
			UnknownUser:       "未知使用者",
			UploadedTemplateLabel: "已上傳範本",
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
		Heading:          "Connection Dashboard",
		LoadErrorPrefix:  "Unable to load dashboard:",
		LoadingEntries:    "Loading dashboard entries...",
		LoadingUser:       "Loading...",
		NoDescription:     "No description provided.",
		RetryButton:       "Retry",
		SearchPlaceholder: "Search entries...",
		Subtitle:          "Select a published entry and download its RDP file.",
		SummaryHosts:     "Hosts",
		SummaryTemplates: "Templates",
		SummaryTotal:     "Total entries",
		UnknownUser:      "Unknown User",
		UploadedTemplateLabel: "Uploaded template",
		WindowTitle:      "RDP Gateway Dashboard",
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

func adminMessagesForLocale(locale string) adminMessages {
	if locale == dashboardLocaleTraditionalChinese {
		return adminMessages{
			AllowedGroupsLabel:            "允許的群組（以逗號分隔）",
			AllowedGroupsPlaceholder:      "homelab-users,rdpgw-admins",
			BackToDashboard:               "返回儀表板",
			CreateDirectAuthUserButton:    "建立直接驗證使用者",
			CreateAuthUserErrorPrefix:     "無法建立直接驗證使用者：",
			CreateDirectAuthUserHeading:   "建立直接驗證使用者",
			CreateDirectAuthUserSuccess:   "已建立直接驗證使用者。",
			CreateHostEntryButton:         "建立主機項目",
			CreateHostEntryErrorPrefix:    "無法建立主機項目：",
			CreateHostEntryHeading:        "建立主機項目",
			CreateHostEntrySuccess:        "已建立主機項目。",
			CreateTemplateEntryHeading:    "建立範本項目",
			CurrentDirectAuthUsersHeading: "目前的直接驗證使用者",
			CurrentEntriesHeading:         "目前的項目",
			DeleteButton:                  "刪除",
			DeleteConfirm:                 "確定要刪除 %s 嗎？",
			DeleteAuthUserErrorPrefix:     "無法刪除直接驗證使用者：",
			DeleteEntryErrorPrefix:        "無法刪除項目：",
			DeleteSuccess:                 "已刪除 %s。",
			DescriptionLabel:              "描述",
			DisabledStatus:                "已停用",
			EditButton:                    "編輯",
			EnabledLabel:                  "啟用",
			EnabledStatus:                 "已啟用",
			EntryTypes: map[string]string{
				"host":     "主機",
				"template": "範本",
			},
			Heading:                        "項目管理",
			HostLabel:                      "主機（host:port）",
			HostPlaceholder:                "lab.internal:3389",
			LoadAuthUsersErrorPrefix:       "無法載入直接驗證使用者：",
			LoadEntriesErrorPrefix:         "無法載入項目：",
			LoadUserErrorPrefix:            "無法載入使用者資訊：",
			LoadingUser:                    "載入中...",
			NameLabel:                      "名稱",
			NewPasswordLabel:               "新密碼",
			NoAuthUsersConfigured:          "目前沒有已設定的直接驗證使用者。",
			NoDescription:                  "未提供描述。",
			NoEntriesConfigured:            "目前沒有已設定的項目。",
			PasswordKeepPlaceholder:        "留空以保留目前密碼",
			PasswordLabel:                  "密碼",
			RdpTemplateFileLabel:           "RDP 範本檔案",
			SaveButton:                     "儲存",
			SaveAuthUserErrorPrefix:        "無法儲存直接驗證使用者：",
			SaveEntryErrorPrefix:           "無法儲存項目：",
			SaveSuccess:                    "已儲存 %s。",
			Subtitle:                       "建立並管理已發布的主機與範本式項目。",
			TabDirectAuthUsers:             "直接驗證使用者",
			TabPublishedEntries:            "已發布項目",
			TargetHostOverrideLabel:        "目標主機覆寫（可選）",
			TargetHostOverridePlaceholder:  "app.internal:3389",
			ToggleEnabledButton:            "切換啟用狀態",
			UnknownUser:                    "未知使用者",
			UploadedTemplateLabel:          "已上傳範本",
			UpdateAuthUserErrorPrefix:      "無法更新直接驗證使用者：",
			UpdateEntryErrorPrefix:         "無法更新項目：",
			UpdateSuccess:                  "已更新 %s。",
			UploadTemplateEntryButton:      "上傳範本項目",
			UploadTemplateEntryErrorPrefix: "無法上傳範本項目：",
			UploadTemplateEntrySuccess:     "已上傳範本項目。",
			UsernameLabel:                  "使用者名稱",
			WindowTitle:                    "RDP Gateway 管理",
		}
	}

	return adminMessages{
		AllowedGroupsLabel:            "Allowed Groups (comma separated)",
		AllowedGroupsPlaceholder:      "homelab-users,rdpgw-admins",
		BackToDashboard:               "Back to Dashboard",
		CreateDirectAuthUserButton:    "Create RDP Gateway User",
		CreateAuthUserErrorPrefix:     "Unable to create auth user:",
		CreateDirectAuthUserHeading:   "Create RDP Gateway User",
		CreateDirectAuthUserSuccess:   "Direct auth user created.",
		CreateHostEntryButton:         "Create Host Entry",
		CreateHostEntryErrorPrefix:    "Unable to create host entry:",
		CreateHostEntryHeading:        "Create Host Entry",
		CreateHostEntrySuccess:        "Host entry created.",
		CreateTemplateEntryHeading:    "Create Template Entry",
		CurrentDirectAuthUsersHeading: "Current Direct Auth Users",
		CurrentEntriesHeading:         "Current Entries",
		DeleteButton:                  "Delete",
		DeleteConfirm:                 "Are you sure you want to delete %s?",
		DeleteAuthUserErrorPrefix:     "Unable to delete auth user:",
		DeleteEntryErrorPrefix:        "Unable to delete entry:",
		DeleteSuccess:                 "Deleted %s.",
		DescriptionLabel:              "Description",
		DisabledStatus:                "disabled",
		EditButton:                    "Edit",
		EnabledLabel:                  "Enabled",
		EnabledStatus:                 "enabled",
		EntryTypes: map[string]string{
			"host":     "Host",
			"template": "Template",
		},
		Heading:                        "Entry Administration",
		HostLabel:                      "Host (host:port)",
		HostPlaceholder:                "lab.internal:3389",
		LoadAuthUsersErrorPrefix:       "Unable to load auth users:",
		LoadEntriesErrorPrefix:         "Unable to load entries:",
		LoadUserErrorPrefix:            "Unable to load user information:",
		LoadingUser:                    "Loading...",
		NameLabel:                      "Name",
		NewPasswordLabel:               "New Password",
		NoAuthUsersConfigured:          "No direct auth users configured yet.",
		NoDescription:                  "No description provided.",
		NoEntriesConfigured:            "No entries configured yet.",
		PasswordKeepPlaceholder:        "Leave blank to keep current password",
		PasswordLabel:                  "Password",
		RdpTemplateFileLabel:           "RDP Template File",
		SaveButton:                     "Save",
		SaveAuthUserErrorPrefix:        "Unable to save auth user:",
		SaveEntryErrorPrefix:           "Unable to save entry:",
		SaveSuccess:                    "Saved %s.",
		Subtitle:                       "Create and manage published hosts and template-based entries.",
		TabDirectAuthUsers:             "Direct Auth Users",
		TabPublishedEntries:            "Published Entries",
		TargetHostOverrideLabel:        "Target Host Override (optional)",
		TargetHostOverridePlaceholder:  "app.internal:3389",
		ToggleEnabledButton:            "Toggle Enabled",
		UnknownUser:                    "Unknown User",
		UploadedTemplateLabel:          "Uploaded template",
		UpdateAuthUserErrorPrefix:      "Unable to update auth user:",
		UpdateEntryErrorPrefix:         "Unable to update entry:",
		UpdateSuccess:                  "Updated %s.",
		UploadTemplateEntryButton:      "Upload Template Entry",
		UploadTemplateEntryErrorPrefix: "Unable to upload template entry:",
		UploadTemplateEntrySuccess:     "Template entry uploaded.",
		UsernameLabel:                  "Username",
		WindowTitle:                    "RDP Gateway Admin",
	}
}

func adminMessagesJSON(messages adminMessages) template.JS {
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
