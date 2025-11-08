#!/bin/bash

set -euo pipefail

# Restic backup script for 'etc' profile - Remote Execution Version
# Runs on remote host, executes backup via SSH on source system
# Safely executes check, snapshots, notifications, and cleanup commands
# Outputs JSON logs to temporary directory
#
# Security: Uses SSH SendEnv to forward RESTIC_*, B2_*, and AWS_* environment variables
# containing credentials. SSH config must include SendEnv directives.

# Configuration variables
RESTIC="/usr/local/bin/restic"
REMOTE_RESTIC_SRC="/usr/local/bin/restic"  # Path to restic on source system
RESTIC_HOOKS="/usr/local/bin/restic-kit"
RESTIC_REPOSITORY="s3:s3.us-west-002.backblazeb2.com/my-backup-bucket"
SSH_HOST="backup-source"  # SSH config host alias for source system
SSH_PORT="22"  # SSH port for source system

# Export repository and credentials for SSH forwarding
export RESTIC_REPOSITORY
export B2_ACCOUNT_ID="${B2_ACCOUNT_ID:-}"
export B2_ACCOUNT_KEY="${B2_ACCOUNT_KEY:-}"
export RESTIC_PASSWORD="${RESTIC_PASSWORD:-}"
export AWS_ACCESS_KEY_ID="${AWS_ACCESS_KEY_ID:-}"
export AWS_SECRET_ACCESS_KEY="${AWS_SECRET_ACCESS_KEY:-}"

# Create temporary directory for logs
TEMP_DIR=$(mktemp -d)

# Setup remote restic binary
echo "Setting up restic binary on remote host..."
REMOTE_RESTIC=$(ssh -p "$SSH_PORT" "$SSH_HOST" "mktemp")
echo "Created remote temp file: $REMOTE_RESTIC"

# Copy restic binary to remote host
echo "Copying restic binary to remote host..."
scp -P "$SSH_PORT" "$REMOTE_RESTIC_SRC" "$SSH_HOST:$REMOTE_RESTIC"
ssh -p "$SSH_PORT" "$SSH_HOST" "chmod +x $REMOTE_RESTIC"

echo "Using restic binary at: $REMOTE_RESTIC"

# Cleanup function to send email notification and remove temp dir
cleanup() {
    # Clean up remote temporary restic binary
    if [ -n "$REMOTE_RESTIC" ]; then
        echo "Cleaning up remote temporary restic binary: $REMOTE_RESTIC"
        ssh -p "$SSH_PORT" "$SSH_HOST" "rm -f $REMOTE_RESTIC" || true
    fi

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


# Perform backup via SSH on source system
echo "Starting backup for etc on source system via SSH..."
ssh -p "$SSH_PORT" "$SSH_HOST" "
$REMOTE_RESTIC backup \
    --exclude=/etc/foobar \
    --one-file-system \
    --verbose=2 \
    --json \
    /etc
" 2> >(tee "$TEMP_DIR/backup.etc.err" >&2) | tee "$TEMP_DIR/backup.etc.out"
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
