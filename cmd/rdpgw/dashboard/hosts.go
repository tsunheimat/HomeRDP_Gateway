package dashboard

func EnabledHostAddresses(entries []Entry) []string {
	hosts := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type != EntryTypeHost || !entry.Enabled {
			continue
		}
		hosts = append(hosts, entry.Host)
	}
	return hosts
}
