#!/bin/ash
# Default values if env vars are not set
export PROXY_LISTEN=${PROXY_LISTEN:-tcp://:80}
export WEB_LISTEN=${WEB_LISTEN:-tcp://:8080}
export REVERSE=${REVERSE_PROXY:+true}
export REVERSE=${REVERSE:-false}
export PROXY_AUTHORIZATION=${PROXY_AUTHORIZATION:-true}
export LOG_LEVEL=${LOG_LEVEL:-0}
export ADMIN_PASSWORD=${ADMIN_PASSWORD:-proxy}
export PROXY_USER=${PROXY_USER:-{}}

envsubst < config.template > config.json
exec ./server -c config.json