#!/bin/sh
echo "Setting permissions" && \
chown -R gokapi:gokapi /app && \
chmod -R 700 /app && \
echo "Starting application" && \
exec su-exec gokapi:gokapi /app/gokapi "$@"
