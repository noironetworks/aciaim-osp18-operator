#!/bin/sh
# This script initializes the AIM service.
set -e

{{if .LogToDisk}}STATE_DIR="/var/log/aim"
DONE_FILE="$STATE_DIR/init_done"

if [ -f "$DONE_FILE" ]; then
    echo "Initialization already completed. Exiting."
    exit 0
fi

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

aimctl $CONFIG_FILES config update

# infra create may fail if the VMM domain already exists with EPGs (APIC error 1579).
# This is expected on redeployments, so allow it to fail.
aimctl $CONFIG_FILES infra create || echo "Warning: infra create failed (may already exist)"

aimctl $CONFIG_FILES manager load-domains

{{if .LogToDisk}}touch "$DONE_FILE"
{{end}}
