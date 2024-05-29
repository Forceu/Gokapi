#!/bin/sh
if [ "$DOCKER_NONROOT" = "true" ]; then
	echo "Setting permissions" && \
	chown -R gokapi:gokapi /app && \
	chmod -R 700 /app && \
	echo "Starting application" && \
	exec su-exec gokapi:gokapi /app/gokapi "$@"
else
	exec /app/gokapi "$@"
fi

