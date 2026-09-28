#!/bin/sh
# This script initializes AIM whenever the aimctl configuration changes.

# Exit immediately if any command fails.
set -e

STATE_DIR="/var/log/aim"
CONFIG_HASH_FILE="$STATE_DIR/aimctl.conf.sha256"

{{if .LogToDisk}}STATE_DIR="/var/log/aim"

mkdir -p "$STATE_DIR"
{{else}}POD_ORDINAL="${POD_NAME##*-}"
if [ "$POD_ORDINAL" != "0" ]; then
    echo "Skipping AIM initialization on pod ordinal $POD_ORDINAL."
    exit 0
fi
{{end}}
# Use the source config path since postStart may run before kolla copies configs to /etc/aim/.
CONFIG_DIR="/var/lib/kolla/config_files/src/etc/aim"
CONFIG_FILES="--config-file=$CONFIG_DIR/aim.conf --config-file=$CONFIG_DIR/aimctl.conf"
mkdir -p "$STATE_DIR"

CONFIG_HASH=$(sha256sum "$CONFIG_DIR/aimctl.conf" | cut -d ' ' -f 1)
if [ -f "$CONFIG_HASH_FILE" ] && [ "$(cat "$CONFIG_HASH_FILE")" = "$CONFIG_HASH" ]; then
    echo "AIM CLI configuration unchanged. Exiting."
    exit 0
fi

# Apply AIM configuration when the aimctl configuration changes.
aimctl $CONFIG_FILES config update

# infra create may fail if the VMM domain already exists with EPGs (APIC error 1579).
# This is expected on redeployments, so allow it to fail.
aimctl $CONFIG_FILES infra create || echo "Warning: infra create failed (may already exist)"

aimctl $CONFIG_FILES manager load-domains

# Record the applied configuration only after all setup commands succeed.
printf '%s\n' "$CONFIG_HASH" > "$CONFIG_HASH_FILE"
