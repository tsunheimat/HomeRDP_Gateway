#!/bin/sh

USER=rdpgw

cd /opt/rdpgw || exit 1

. /run.lib.sh

AUTH_SOCKET=${RDPGW_SERVER__AUTH_SOCKET:-/tmp/rdpgw-auth.sock}
AUTH_CONFIG=${RDPGW_AUTH_HELPER_CONFIG:-${RDPGW_DASHBOARD__AUTHHELPERCONFIGPATH:-/opt/rdpgw/data/dashboard/rdpgw-auth.yaml}}

start_helper=$(rdpgw_should_start_auth_helper "$@")

if [ "${start_helper}" = "true" ]; then
  echo "Starting rdpgw-auth (socket: ${AUTH_SOCKET})"
  AUTH_CMD="/opt/rdpgw/rdpgw-auth -s ${AUTH_SOCKET}"
  if [ -f "${AUTH_CONFIG}" ]; then
    echo "Using auth helper config ${AUTH_CONFIG}"
    AUTH_CMD="${AUTH_CMD} -c ${AUTH_CONFIG}"
  else
    echo "Auth helper config ${AUTH_CONFIG} not found yet; starting helper and waiting for generated config"
  fi
  sh -c "${AUTH_CMD}" &
fi

# drop privileges and run the application
su -c /opt/rdpgw/rdpgw "${USER}" -- "$@" &
wait
exit $?
