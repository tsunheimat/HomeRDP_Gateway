package dashboard

import "strings"

const (
	EntryIconWindow     = "window"
	EntryIconBrowser    = "browser"
	EntryIconTerminal   = "terminal"
	EntryIconFolder     = "folder"
	EntryIconDatabase   = "database"
	EntryIconWord       = "word"
	EntryIconExcel      = "excel"
	EntryIconPowerPoint = "powerpoint"
	EntryIconOutlook    = "outlook"
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
	_, ok := allowedEntryIcons[normalized]
	return ok
}
