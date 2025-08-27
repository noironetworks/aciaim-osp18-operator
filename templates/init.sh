#!/bin/sh
# This script is run once to initialize the AIM service.
# It uses a "done file" on a persistent volume to ensure it only runs once.

# Exit immediately if any command fails.
set -e

# Define the location for our "done file" on the persistent volume.
STATE_DIR="/var/log/aim"
DONE_FILE="$STATE_DIR/init_done"

# Check if the initialization has already been completed.
if [ -f "$DONE_FILE" ]; then
    echo "Initialization already completed. Exiting."
    exit 0
fi

# If the done file doesn't exist, run the initialization commands.
# Ensure the state directory exists before trying to write to it.
mkdir -p $STATE_DIR

# Run the initialization commands.
aimctl config update

aimctl infra create

aimctl manager load-domains

# As the very last step, create the "done file".
touch "$DONE_FILE"
