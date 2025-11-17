#!/bin/sh

USER=rdpgw

file="/root/createusers.txt"
if [ -f $file ]
  then
    while IFS=: read -r username password is_sudo
        do
            echo "Username: $username, Password: **** , Sudo: $is_sudo"

            if getent passwd "$username" > /dev/null 2>&1
              then
                echo "User Exists"
              else
                adduser -s /sbin/nologin "$username"
                echo "$username:$password" | chpasswd
            fi
    done <"$file"
fi

cd /opt/rdpgw || exit 1

AUTH_MODES=$(echo "${RDPGW_SERVER__AUTHENTICATION}" | tr ',;' ' ')
AUTH_SOCKET=${RDPGW_SERVER__AUTH_SOCKET:-/tmp/rdpgw-auth.sock}
AUTH_CONFIG=${RDPGW_AUTH_HELPER_CONFIG:-/opt/rdpgw/rdpgw-auth.yaml}

start_helper=false
for mode in ${AUTH_MODES}; do
  case "${mode}" in
    local|ntlm)
      start_helper=true
      ;;
  esac
done

if [ "${start_helper}" = "true" ]; then
  echo "Starting rdpgw-auth (socket: ${AUTH_SOCKET})"
  AUTH_CMD="/opt/rdpgw/rdpgw-auth -s ${AUTH_SOCKET}"
  if [ -f "${AUTH_CONFIG}" ]; then
    echo "Using auth helper config ${AUTH_CONFIG}"
    AUTH_CMD="${AUTH_CMD} -c ${AUTH_CONFIG}"
  else
    echo "Auth helper config ${AUTH_CONFIG} not found, proceeding without -c"
  fi
  sh -c "${AUTH_CMD}" &
fi

# drop privileges and run the application
su -c /opt/rdpgw/rdpgw "${USER}" -- "$@" &
wait
exit $?
