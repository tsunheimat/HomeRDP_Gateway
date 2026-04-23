package dashboard

import "strings"

const (
	EntryIconWindow         = "window"
	EntryIconBrowser        = "browser"
	EntryIconTerminal       = "terminal"
	EntryIconFolder         = "folder"
	EntryIconDatabase       = "database"
	EntryIconWord           = "word"
	EntryIconExcel          = "excel"
	EntryIconPowerPoint     = "powerpoint"
	EntryIconOutlook        = "outlook"
	EntryIconUploadedPrefix = "uploaded:"
)

var allowedEntryIcons = map[string]struct{}{
	EntryIconWindow:     {},
	EntryIconBrowser:    {},
	EntryIconTerminal:   {},
	EntryIconFolder:     {},
	EntryIconDatabase:   {},
	EntryIconWord:       {},
	EntryIconExcel:      {},
	EntryIconPowerPoint: {},
	EntryIconOutlook:    {},
}

func NormalizeEntryIcon(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if IsUploadedEntryIcon(normalized) {
		return normalized
	}
	if _, ok := allowedEntryIcons[normalized]; ok {
		return normalized
	}
	return EntryIconWindow
}

func IsValidEntryIcon(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return true
	}
	if IsUploadedEntryIcon(normalized) {
		return true
	}
	_, ok := allowedEntryIcons[normalized]
	return ok
}

func IsUploadedEntryIcon(value string) bool {
	_, ok := UploadedEntryIconID(value)
	return ok
}

func UploadedEntryIconID(value string) (string, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if !strings.HasPrefix(normalized, EntryIconUploadedPrefix) {
		return "", false
	}
	id := strings.TrimPrefix(normalized, EntryIconUploadedPrefix)
	if len(id) != 32 {
		return "", false
	}
	for _, ch := range id {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return "", false
		}
	}
	return id, true
}
