#!/bin/sh

USER=rdpgw

cd /opt/rdpgw || exit 1

. /run.lib.sh

AUTH_SOCKET=${RDPGW_SERVER__AUTH_SOCKET:-/tmp/rdpgw-auth.sock}
AUTH_CONFIG=$(rdpgw_auth_helper_config_path "$@")
SPLIT_MODE=$(rdpgw_split_enabled "$@")

start_helper=$(rdpgw_should_start_auth_helper "$@")
pids=""

start_rdpgw_instance() {
  mode=$1
  shift

  (
    eval "$(rdpgw_runtime_exports "${mode}" "$@")"
    echo "Starting rdpgw-${mode} (port: ${RDPGW_SERVER__PORT:-default}, gateway: ${RDPGW_SERVER__GATEWAYADDRESS:-default})"
    exec su -c /opt/rdpgw/rdpgw "${USER}" -- "$@"
  ) &

  pids="${pids} $!"
}

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
  pids="${pids} $!"
fi

if [ "${SPLIT_MODE}" = "true" ]; then
  echo "Split gateway mode enabled"
  start_rdpgw_instance oidc "$@"
  start_rdpgw_instance direct "$@"
else
  # drop privileges and run the application
  su -c /opt/rdpgw/rdpgw "${USER}" -- "$@" &
  pids="${pids} $!"
fi

while :; do
  for pid in ${pids}; do
    if ! kill -0 "${pid}" 2>/dev/null; then
      wait "${pid}"
      status=$?
      for other_pid in ${pids}; do
        if [ "${other_pid}" != "${pid}" ]; then
          kill "${other_pid}" 2>/dev/null || true
        fi
      done
      wait 2>/dev/null || true
      exit "${status}"
    fi
  done
  sleep 1
done
