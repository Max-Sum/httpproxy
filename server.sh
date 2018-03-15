#!/bin/bash
# Default values if env vars are not set
PROXY_LISTEN=${PROXY_LISTEN:-tcp://:80}
WEB_LISTEN=${WEB_LISTEN:-tcp://:8080}
REVERSE=${REVERSE_PROXY:+true}
REVERSE=${REVERSE:-false}
PROXY_AUTHORIZATION=${PROXY_AUTHORIZATION:-true}
LOG_LEVEL=${LOG_LEVEL:-0}
ADMIN_PASSWORD=${ADMIN_PASSWORD:-proxy}
PROXY_USER=${PROXY_USER:-{}}

envsubst < config/config.template > config/config.json
exec ./server -c config/config.json