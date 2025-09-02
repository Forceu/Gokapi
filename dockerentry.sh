#!/bin/sh

echo "Setting permissions"
# chmod 750 for directories, 640 for files; does not remove existing u+x or g+x file permissions
find -P /app -type d -exec chmod 750 -- {} + -o \
	-type f -exec chmod u+rw,g+r,o-rwx -- {} +

echo "Setting ownership"
if [ "$DOCKER_NONROOT" = "true" ]; then
	chown -R gokapi:gokapi /app
	echo "Starting application"
	exec su-exec gokapi:gokapi /app/gokapi "$@"
else
	chown -R root:root /app  # Restore permissions if previously NONROOT was used
	echo "Starting application"
	exec /app/gokapi "$@"
fi
