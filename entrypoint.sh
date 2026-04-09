#!/bin/sh

# Default to 1000 if variables are not set
USER_ID=${UID:-1000}
GROUP_ID=${GID:-1000}

echo "Starting with UID: $USER_ID, GID: $GROUP_ID"

# Update the appuser's UID and GID to match the env vars
# We use shadow (usermod/groupmod) or modify /etc/passwd manually if shadow is missing
if [ "$(id -u appuser)" -ne "$USER_ID" ] || [ "$(id -g appuser)" -ne "$GROUP_ID" ]; then
    # Handle alpine specifically (which uses usermod/groupmod from the 'shadow' package)
    # You might need to install 'shadow' in your Dockerfile
    usermod -u $USER_ID -o appuser
    groupmod -g $GROUP_ID -o appuser

    # Fix permissions for the app directory so the new UID owns it
    chown -R appuser:appuser /app
    chown -R appuser:appuser /appdata
fi

# Switch to the user and run the main command
# 'exec' replaces the shell process with your binary
exec su-exec appuser "$@"
