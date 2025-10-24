#!/bin/sh
if [ "$DOCKER_NONROOT" = "true" ]; then
	# TODO for the next major upgrade version:
	# 	- Remove this code block and leave only exec /app/gokapi "@"
	# 	- Remove gokapi user / group creation in Dockerfile
	# 	- Remove su-exec installation from the Dockerfile

	echo "WARNING:"
	echo "	Usage of DOCKER_NONROOT is deprecated in favor of docker --user option!"
	echo "	Please visit the Gokapi docker setup documentation page for more details."
	echo
	echo "Setting permissions" && \
	chown -R gokapi:gokapi /app && \
	chmod -R 700 /app && \
	echo "Starting application" && \
	exec su-exec gokapi:gokapi /app/gokapi "$@"
else
	exec /app/gokapi "$@"
fi

