#!/bin/sh

rdpgw_config_path() {
	while [ $# -gt 0 ]; do
		case "$1" in
			-c|--conf)
				shift
				if [ $# -gt 0 ]; then
					printf '%s\n' "$1"
					return 0
				fi
				;;
			--conf=*)
				printf '%s\n' "${1#--conf=}"
				return 0
				;;
		esac
		shift
	done

	printf '%s\n' "rdpgw.yaml"
}

rdpgw_auth_modes_from_config() {
	config_path=$1
	[ -f "${config_path}" ] || return 0

	awk '
		BEGIN {
			in_server = 0
			in_auth = 0
		}
		/^[[:space:]]*Server:[[:space:]]*$/ {
			in_server = 1
			in_auth = 0
			next
		}
		in_server && /^[^[:space:]]/ {
			in_server = 0
			in_auth = 0
		}
		!in_server {
			next
		}
		in_server && /^[[:space:]]+Authentication:[[:space:]]*\[[^]]+\][[:space:]]*$/ {
			line = $0
			sub(/^[[:space:]]+Authentication:[[:space:]]*\[/, "", line)
			sub(/\][[:space:]]*$/, "", line)
			gsub(/[[:space:]]*,[[:space:]]*/, "\n", line)
			print line
			next
		}
		in_server && /^[[:space:]]+Authentication:[[:space:]]*$/ {
			in_auth = 1
			next
		}
		in_auth && /^[[:space:]]+-[[:space:]]*/ {
			line = $0
			sub(/^[[:space:]]+-[[:space:]]*/, "", line)
			gsub(/[[:space:]]+$/, "", line)
			if (line != "") {
				print line
			}
			next
		}
		in_auth {
			in_auth = 0
		}
	' "${config_path}"
}

rdpgw_auth_modes() {
	if [ -n "${RDPGW_SERVER__AUTHENTICATION}" ]; then
		printf '%s\n' "${RDPGW_SERVER__AUTHENTICATION}" | tr ',;' ' '
		return 0
	fi

	rdpgw_auth_modes_from_config "$(rdpgw_config_path "$@")"
}

rdpgw_should_start_auth_helper() {
	for mode in $(rdpgw_auth_modes "$@"); do
		case "${mode}" in
			local|ntlm)
				printf '%s\n' "true"
				return 0
				;;
		esac
	done

	printf '%s\n' "false"
}
