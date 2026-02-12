#!/bin/sh
set -eu

DOMAIN="deletebadzim.aurorass.art"
CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
FULLCHAIN="${CERT_DIR}/fullchain.pem"
PRIVKEY="${CERT_DIR}/privkey.pem"

if [ -f "$FULLCHAIN" ] && [ -f "$PRIVKEY" ]; then
  echo "[nginx] TLS cert found for ${DOMAIN}; enabling HTTPS"
  cp /etc/nginx/templates/nginx.https.conf /etc/nginx/nginx.conf
else
  echo "[nginx] TLS cert not found for ${DOMAIN}; starting HTTP-only (ACME bootstrap)"
  cp /etc/nginx/templates/nginx.http.conf /etc/nginx/nginx.conf
fi

nginx -t
