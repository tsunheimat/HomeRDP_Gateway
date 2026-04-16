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

rdpgw_split_enabled_from_config() {
	config_path=$1
	[ -f "${config_path}" ] || return 0

	awk '
		function indent_of(line,    match_len) {
			match(line, /^[[:space:]]*/)
			return RLENGTH
		}
		BEGIN {
			in_split = 0
			split_indent = -1
		}
		/^[[:space:]]*GatewaySplit:[[:space:]]*$/ {
			in_split = 1
			split_indent = indent_of($0)
			next
		}
		in_split && indent_of($0) <= split_indent && $0 !~ /^[[:space:]]*$/ {
			in_split = 0
		}
		in_split && /^[[:space:]]+Enabled:[[:space:]]*/ {
			line = $0
			sub(/^[[:space:]]+Enabled:[[:space:]]*/, "", line)
			gsub(/[[:space:]]+$/, "", line)
			print line
			exit
		}
	' "${config_path}"
}

rdpgw_split_enabled() {
	if [ -n "${RDPGW_GATEWAYSPLIT__ENABLED}" ]; then
		printf '%s\n' "${RDPGW_GATEWAYSPLIT__ENABLED}"
		return 0
	fi

	rdpgw_split_enabled_from_config "$(rdpgw_config_path "$@")"
}

rdpgw_split_listener_value_from_config() {
	config_path=$1
	listener=$2
	field=$3
	[ -f "${config_path}" ] || return 0

	awk -v listener="${listener}" -v field="${field}" '
		function indent_of(line,    match_len) {
			match(line, /^[[:space:]]*/)
			return RLENGTH
		}
		BEGIN {
			in_split = 0
			in_listener = 0
			split_indent = -1
			listener_indent = -1
		}
		/^[[:space:]]*GatewaySplit:[[:space:]]*$/ {
			in_split = 1
			in_listener = 0
			split_indent = indent_of($0)
			next
		}
		in_split && indent_of($0) <= split_indent && $0 !~ /^[[:space:]]*$/ {
			in_split = 0
			in_listener = 0
		}
		in_split && $0 ~ "^[[:space:]]+" listener ":[[:space:]]*$" {
			in_listener = 1
			listener_indent = indent_of($0)
			next
		}
		in_listener && indent_of($0) <= listener_indent && $0 !~ /^[[:space:]]*$/ {
			in_listener = 0
		}
		in_listener && $0 ~ "^[[:space:]]+" field ":[[:space:]]*" {
			line = $0
			sub("^[[:space:]]+" field ":[[:space:]]*", "", line)
			sub(/[[:space:]]+#.*$/, "", line)
			gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
			print line
			exit
		}
	' "${config_path}"
}

rdpgw_split_listener_auth_modes_from_config() {
	config_path=$1
	listener=$2
	[ -f "${config_path}" ] || return 0

	awk -v listener="${listener}" '
		function indent_of(line,    match_len) {
			match(line, /^[[:space:]]*/)
			return RLENGTH
		}
		BEGIN {
			in_split = 0
			in_listener = 0
			in_auth = 0
			split_indent = -1
			listener_indent = -1
			auth_indent = -1
		}
		/^[[:space:]]*GatewaySplit:[[:space:]]*$/ {
			in_split = 1
			in_listener = 0
			in_auth = 0
			split_indent = indent_of($0)
			next
		}
		in_split && indent_of($0) <= split_indent && $0 !~ /^[[:space:]]*$/ {
			in_split = 0
			in_listener = 0
			in_auth = 0
		}
		in_split && $0 ~ "^[[:space:]]+" listener ":[[:space:]]*$" {
			in_listener = 1
			in_auth = 0
			listener_indent = indent_of($0)
			next
		}
		in_listener && indent_of($0) <= listener_indent && $0 !~ /^[[:space:]]*$/ {
			in_listener = 0
			in_auth = 0
		}
		in_listener && /^[[:space:]]+Authentication:[[:space:]]*\[[^]]+\][[:space:]]*$/ {
			line = $0
			sub(/^[[:space:]]+Authentication:[[:space:]]*\[/, "", line)
			sub(/\][[:space:]]*$/, "", line)
			gsub(/[[:space:]]*,[[:space:]]*/, " ", line)
			gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
			print line
			exit
		}
		in_listener && /^[[:space:]]+Authentication:[[:space:]]*$/ {
			in_auth = 1
			auth_indent = indent_of($0)
			next
		}
		in_auth && indent_of($0) <= auth_indent && $0 !~ /^[[:space:]]*$/ {
			in_auth = 0
		}
		in_auth && /^[[:space:]]+-[[:space:]]*/ {
			line = $0
			sub(/^[[:space:]]+-[[:space:]]*/, "", line)
			gsub(/[[:space:]]+$/, "", line)
			if (line != "") {
				modes = modes (modes == "" ? "" : " ") line
			}
			next
		}
		END {
			if (modes != "") {
				print modes
			}
		}
	' "${config_path}"
}

rdpgw_split_listener_value() {
	listener=$1
	field=$2
	shift 2
	rdpgw_split_listener_value_from_config "$(rdpgw_config_path "$@")" "${listener}" "${field}"
}

rdpgw_split_listener_auth_modes() {
	listener=$1
	shift
	rdpgw_split_listener_auth_modes_from_config "$(rdpgw_config_path "$@")" "${listener}"
}

rdpgw_shell_quote() {
	printf "%s" "$1" | sed "s/'/'\\\\''/g"
}

rdpgw_dashboard_value_from_config() {
	config_path=$1
	field=$2
	[ -f "${config_path}" ] || return 0

	awk -v field="${field}" '
		BEGIN {
			in_dashboard = 0
		}
		/^[[:space:]]*Dashboard:[[:space:]]*$/ {
			in_dashboard = 1
			next
		}
		in_dashboard && /^[^[:space:]]/ {
			in_dashboard = 0
		}
		!in_dashboard {
			next
		}
		{
			pattern = "^[[:space:]]+" field ":[[:space:]]*"
			if ($0 ~ pattern) {
				line = $0
				sub(pattern, "", line)
				sub(/[[:space:]]+#.*$/, "", line)
				gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
				print line
				exit
			}
		}
	' "${config_path}"
}

rdpgw_derived_auth_helper_config_path() {
	store_path=$1
	if [ -z "${store_path}" ]; then
		printf '%s\n' "/var/lib/rdpgw/dashboard/rdpgw-auth.yaml"
		return 0
	fi

	case "${store_path}" in
		*/)
			printf '%s\n' "${store_path}rdpgw-auth.yaml"
			;;
		*)
			printf '%s\n' "${store_path}/rdpgw-auth.yaml"
			;;
	esac
}

rdpgw_auth_helper_config_path() {
	if [ -n "${RDPGW_AUTH_HELPER_CONFIG}" ]; then
		printf '%s\n' "${RDPGW_AUTH_HELPER_CONFIG}"
		return 0
	fi

	if [ -n "${RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH}" ]; then
		printf '%s\n' "${RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH}"
		return 0
	fi

	config_path=$(rdpgw_config_path "$@")
	configured_path=$(rdpgw_dashboard_value_from_config "${config_path}" "AuthHelperConfigPath")
	if [ -n "${configured_path}" ]; then
		printf '%s\n' "${configured_path}"
		return 0
	fi

	if [ -n "${RDPGW_DASHBOARD__STOREPATH}" ]; then
		rdpgw_derived_auth_helper_config_path "${RDPGW_DASHBOARD__STOREPATH}"
		return 0
	fi

	configured_store=$(rdpgw_dashboard_value_from_config "${config_path}" "StorePath")
	rdpgw_derived_auth_helper_config_path "${configured_store}"
}

rdpgw_should_start_auth_helper() {
	if [ "$(rdpgw_split_enabled "$@")" = "true" ]; then
		for mode in $(rdpgw_split_listener_auth_modes Direct "$@"); do
			case "${mode}" in
				local|ntlm)
					printf '%s\n' "true"
					return 0
					;;
			esac
		done

		printf '%s\n' "false"
		return 0
	fi

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

rdpgw_runtime_exports() {
	mode=$1
	shift

	if [ "$(rdpgw_split_enabled "$@")" != "true" ]; then
		return 0
	fi

	case "${mode}" in
		oidc)
			port=$(rdpgw_split_listener_value OIDC Port "$@")
			host=$(rdpgw_split_listener_value OIDC Hostname "$@")
			auth="openid"
			token_auth="true"
			;;
		direct)
			port=$(rdpgw_split_listener_value Direct Port "$@")
			host=$(rdpgw_split_listener_value Direct Hostname "$@")
			auth=$(rdpgw_split_listener_auth_modes Direct "$@")
			token_auth="false"
			;;
		*)
			return 1
			;;
	esac

	if [ -n "${port}" ]; then
		printf "export RDPGW_SERVER__PORT='%s'\n" "$(rdpgw_shell_quote "${port}")"
	fi
	if [ -n "${host}" ]; then
		printf "export RDPGW_SERVER__GATEWAYADDRESS='%s'\n" "$(rdpgw_shell_quote "${host}")"
	fi
	if [ -n "${auth}" ]; then
		printf "export RDPGW_SERVER__AUTHENTICATION='%s'\n" "$(rdpgw_shell_quote "${auth}")"
	fi
	printf "export RDPGW_CAPS__TOKENAUTH='%s'\n" "$(rdpgw_shell_quote "${token_auth}")"
}
