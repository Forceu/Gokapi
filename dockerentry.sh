#!/bin/sh
if [ -n "$UMASK" ]; then
	umask "$UMASK"
fi && \
if [ -n "$PUID$PGID" ] || [ "$DOCKER_NONROOT" = "true" ]; then
	echo "Setting permissions" && \
	PUID="${PUID-$(id -u gokapi)}" && \
	PGID="${PGID-$(id -g gokapi)}" && \
	sed -E 's/^(gokapi:x):([0-9]+)(.*)$/\1:'"$PGID"'\3/g' -i /etc/group && \
	sed -E 's/^(gokapi:x):([0-9]+:[0-9]+)(.*)$/\1:'"$PUID:$PGID"'\3/' -i /etc/passwd && \
	chown -R gokapi:gokapi /app && \
	chmod -R 700 /app && \
	echo "Starting application" && \
	exec su-exec gokapi:gokapi /app/gokapi "$@"
else
	exec /app/gokapi "$@"
fi

