#!/bin/bash

set -euo pipefail

# Restic backup script for 'etc' profile
# Safely executes initialization, check, backup, and cleanup commands
# Outputs JSON logs to temporary directory

# Configuration variables
RESTIC="/usr/local/bin/restic"
RESTIC_HOOKS="/usr/local/bin/restic-kit"
RESTIC_REPOSITORY="s3:s3.us-west-002.backblazeb2.com/my-backup-bucket"

export RESTIC_REPOSITORY

# Create temporary directory for logs
TEMP_DIR=$(mktemp -d)

# Cleanup function to send email notification and remove temp dir
cleanup() {
    # First send email notification with backup summary
    $RESTIC_HOOKS notify-email \
        --smtp-host "smtp.test.com" \
        --smtp-port "465" \
        --smtp-username "restic@test.com" \
        --smtp-password "${RESTIC_HOOKS_EMAIL_PASSWORD}" \
        --from "restic@test.com" \
        --to "nobody@gmail.com" \
        "$TEMP_DIR" || true

    # Send HTTP notification
    $RESTIC_HOOKS notify-http \
        --url "https://hc-ping.com/<uuid>" \
        "$TEMP_DIR" || true

    # Clean up log directory only if all backups were successful
    $RESTIC_HOOKS cleanup "$TEMP_DIR" || true
}
trap cleanup EXIT


# Perform backup
echo "Starting backup for etc..."
$RESTIC backup \
    --exclude=/etc/foobar \
    --one-file-system \
    --verbose=2 \
    --json \
    /etc 2> >(tee "$TEMP_DIR/backup.etc.err" >&2) | tee "$TEMP_DIR/backup.etc.out"
echo $? > "$TEMP_DIR/backup.etc.exitcode"

# Check repository consistency
echo "Checking repository consistency..."
$RESTIC check \
    --json > "$TEMP_DIR/check.out" 2> "$TEMP_DIR/check.err"
echo $? > "$TEMP_DIR/check.exitcode"

# Get repository snapshots
echo "Getting repository snapshots..."
$RESTIC snapshots \
    --group-by=paths \
    --json > "$TEMP_DIR/snapshots.out" 2> "$TEMP_DIR/snapshots.err"
echo $? > "$TEMP_DIR/snapshots.exitcode"

echo "Backup process completed successfully."
