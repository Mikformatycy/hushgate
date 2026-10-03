#!/bin/sh
# Simulates MDM installing the company root CA into the system trust store.
set -e
if [ "${INSTALL_CORP_CA:-0}" = 1 ] && [ -f /certs/corp-ca.crt ]; then
  cp /certs/corp-ca.crt /usr/local/share/ca-certificates/corp-ca.crt
  update-ca-certificates >/dev/null 2>&1
fi
exec "$@"
