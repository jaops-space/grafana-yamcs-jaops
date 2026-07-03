#!/bin/sh

# Runtime mode selection: DEVELOPMENT takes precedence, then falls back to image DEV.
mode="${DEVELOPMENT:-${DEV}}"

if [ "${mode}" = "true" ]; then
    export NODE_ENV="development"
    export GF_LOG_LEVEL="info"
    export GF_LOG_FILTERS="plugin.jaops-yamcs-app:debug, plugin.jaops-yamcs-datasource:debug"
    echo "Starting development mode"
else
    export NODE_ENV="production"
    export GF_LOG_LEVEL="critical"
    unset GF_LOG_FILTERS
    export GF_DEFAULT_APP_MODE="production"
    echo "Starting production mode"
fi

if [ "${DEV}" = "false" ]; then
    exec /run.sh
fi

if grep -i -q alpine /etc/issue; then
    exec /usr/bin/supervisord -c /etc/supervisord.conf
elif grep -i -q ubuntu /etc/issue; then
    exec /usr/bin/supervisord -c /etc/supervisor/supervisord.conf
else
    echo 'ERROR: Unsupported base image'
    exit 1
fi

