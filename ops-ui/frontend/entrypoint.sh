#!/usr/bin/env bash

set -e

if [ -f /etc/nginx/security-headers.conf ]; then
  cp -v /etc/nginx/security-headers.conf /etc/nginx/default.d/
fi

echo "APLAT_SECURITY_ENABLED=${APLAT_SECURITY_ENABLED}"
if [ "${APLAT_SECURITY_ENABLED:-"false"}" = "true" ] && [ -f /etc/nginx/security-enable-aplat.conf ]; then
  cp -v /etc/nginx/security-enable-aplat.conf /etc/nginx/default.d/
fi

echo "loaded modules: "
ls -la /usr/share/nginx/modules/

echo "loaded defaults: "
ls -la /usr/share/nginx/default.d/

echo "current config: "
cat /etc/nginx/nginx.conf

echo "executing command: "
exec "$@"
