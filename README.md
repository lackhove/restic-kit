# restic-kit

A collection of useful restic helpers and a dead-simple, bash-based orchestration toolkit. It includes a small Go CLI for executing hooks (email/http/wait-online), a few helper scripts, and minimal examples to compose and run restic backups from Bash.

## Installation

```bash
go install github.com/yourusername/restic-kit@latest
```

Or download the binary from the releases page.

## Usage

Configuration is provided via command-line parameters for each subcommand.

## Actions

### notify-email

Send an email notification with backup report details from JSON logs in a directory.

```bash
restic-kit notify-email /path/to/logs \
  --smtp-host smtp.gmail.com \
  --smtp-port 587 \
  --smtp-username your-email@gmail.com \
  --smtp-password "your-app-password" \
  --from your-email@gmail.com \
  --to recipient@example.com
```

**Options:**
- `--smtp-host` (required): SMTP server hostname
- `--smtp-port`: SMTP server port (default: 587)
- `--smtp-username` (required): SMTP username
- `--smtp-password` (required): SMTP password
- `--from` (required): From email address
- `--to` (required): To email address

The command parses JSON logs from the directory and generates a formatted email with:
- Backup summary for each action
- Snapshot tables with file changes and data sizes
- Error attachments if any backups failed
- Repository check status

### notify-http

Perform a single HTTP GET request to notify an external service.

```bash
restic-kit notify-http --url https://api.example.com/notify
```

**Options:**
- `--url` (required): HTTP URL to send the notification to

### wait-online

Wait for network connectivity by checking if a URL is reachable with exponential backoff.

```bash
restic-kit wait-online \
  --url https://www.google.com \
  --timeout 5m \
  --initial-delay 1s \
  --max-delay 30s
```

**Options:**
- `--url`: URL to check for connectivity (default: https://www.google.com)
- `--timeout`: Total timeout for waiting (default: 5m)
- `--initial-delay`: Initial delay between retries (default: 1s)
- `--max-delay`: Maximum delay between retries (default: 30s)

### audit

Audit restic snapshots for size anomalies and retention policy compliance. Checks for unusual size changes between the two most recent snapshots per path and verifies snapshot counts against retention policies. Sends email notifications for any failures.

```bash
restic-kit audit /path/to/logs \
  --grow-threshold 20.0 \
  --shrink-threshold 5.0 \
  --keep-daily 7 \
  --smtp-host smtp.gmail.com \
  --smtp-port 587 \
  --smtp-username your-email@gmail.com \
  --smtp-password "your-app-password" \
  --from your-email@gmail.com \
  --to recipient@example.com
```

**Options:**
- `--grow-threshold`: Maximum allowed growth percentage between snapshots (default: 20.0)
- `--shrink-threshold`: Maximum allowed shrink percentage between snapshots (default: 5.0)
- `--keep-hourly`: Number of hourly snapshots to keep
- `--keep-daily`: Number of daily snapshots to keep
- `--keep-weekly`: Number of weekly snapshots to keep
- `--keep-monthly`: Number of monthly snapshots to keep
- `--keep-yearly`: Number of yearly snapshots to keep
- `--smtp-host`: SMTP server hostname (for email notifications)
- `--smtp-port`: SMTP server port (default: 587)
- `--smtp-username`: SMTP username (for email notifications)
- `--smtp-password`: SMTP password (for email notifications)
- `--from`: From email address (for email notifications)
- `--to`: To email address (for email notifications)

The command performs two types of checks:
1. **Size Change Detection**: Compares the two most recent snapshots per backup path for unusual size changes
2. **Retention Policy Validation**: Verifies snapshot counts against configured retention policies (hourly/daily/weekly/monthly/yearly)

If any checks fail and email configuration is provided, a detailed email report is sent with violation details.

## Remote Backup Execution

This section describes how to set up secure remote backup execution where the backup script runs on a remote host but executes the actual backup via SSH on the source system.

### Architecture

- **Remote Host**: Runs the backup script, handles notifications, repository checks
- **Source Host**: Only executes the `restic backup` command via SSH
- **Security**: Credentials forwarded via SSH SendEnv (whitelist-only)

### Setup Requirements

#### 1. SSH Key Authentication

Generate SSH key pair on remote host:
```bash
ssh-keygen -t ed25519 -f ~/.ssh/backup_id_rsa -C "backup-system"
```

Copy public key to source host:
```bash
ssh-copy-id -i ~/.ssh/backup_id_rsa.pub user@source.example.com
```

#### 2. SSH Configuration (on Remote Host)

Create or modify `~/.ssh/config`:

```bash
Host backup-source
    HostName source.example.com
    User backupuser
    IdentityFile ~/.ssh/backup_id_rsa
    # Forward RESTIC_*, B2_*, and AWS_* environment variables only to this host
    SendEnv RESTIC_*
    SendEnv B2_*
    SendEnv AWS_*
    # Only applies to connections to backup-source host
```

#### 3. Configuration Variables

Set these variables in the `backup.sh` script or as environment variables on the remote host:

**Required Variables:**
```bash
RESTIC="/usr/local/bin/restic"           # Local restic path (for check/snapshots)
REMOTE_RESTIC="/usr/local/bin/restic"    # Remote restic path (for backup on source)
SSH_HOST="backup-source"                 # SSH config host alias
SSH_PORT="22"                            # SSH port for source system
```

**Environment Variables:**
```bash
export RESTIC_REPOSITORY="s3:s3.us-west-002.backblazeb2.com/my-backup-bucket"
export B2_ACCOUNT_ID="your_backblaze_account_id"
export B2_ACCOUNT_KEY="your_backblaze_account_key"
export RESTIC_PASSWORD="your_restic_repository_password"
export RESTIC_HOOKS_EMAIL_PASSWORD="your_email_password"
```

#### 4. SSH Daemon Configuration (on Source Host)

Ensure `/etc/ssh/sshd_config` includes explicit environment variable acceptance:
```bash
AcceptEnv RESTIC_REPOSITORY
AcceptEnv RESTIC_PASSWORD
AcceptEnv B2_ACCOUNT_ID
AcceptEnv B2_ACCOUNT_KEY
AcceptEnv AWS_ACCESS_KEY_ID
AcceptEnv AWS_SECRET_ACCESS_KEY
```

Restart SSH service after changes:
```bash
sudo systemctl restart sshd
```

### Security Benefits

1. **Explicit Whitelisting**: Only approved environment variables are forwarded
2. **No Command-Line Exposure**: Credentials never appear in process arguments
3. **SSH Encryption**: All credential transmission is encrypted
4. **Minimal Attack Surface**: Only backup command runs on source system
5. **Key-Based Authentication**: Strong authentication without passwords

### Testing Remote Setup

Use the provided test script to validate your remote backup setup:

```bash
./test_remote_setup.sh
```

Test the setup step by step:

1. **SSH connectivity**:
   ```bash
   ssh backup-source "echo 'SSH works'"
   ```

2. **Environment forwarding**:
   ```bash
   export TEST_RESTIC_VAR="secret"
   ssh -o SendEnv=TEST_RESTIC_VAR backup-source "echo \$TEST_RESTIC_VAR"
   ```

3. **Restic access**:
   ```bash
   ssh backup-source "restic version"
   ```

4. **Full backup test** (dry run):
   ```bash
   export RESTIC_PASSWORD="test"
   ssh backup-source "restic backup --dry-run /etc"
   ```

### Troubleshooting

- **"Environment variable not set"**: Check SSH config SendEnv and server AcceptEnv
- **"Permission denied"**: Verify SSH key is properly installed
- **"Command not found"**: Ensure restic is installed on source system
- **"Repository access failed"**: Verify credentials are properly forwarded

### Alternative Approaches

If SendEnv is not feasible, consider:
- SSH with here documents (credentials in script memory)
- Base64 encoded credential bundles
- Temporary credential files with automatic cleanup

But SendEnv provides the best balance of security and usability.

## Restic Integration and Bash orchestration

Use restic-kit in your restic backup commands. This repository contains a tiny, example Bash runner (`backup.sh`) that demonstrates how to combine the provided helpers and the CLI into simple, repeatable backup flows. The scripts are intentionally minimal so you can adapt them to your environment.

Use the CLI and scripts together like this:

```bash
# Wait for network before backup
restic-kit wait-online && restic backup /data

# Send email notification after backup with JSON logs
restic backup /data --json > /tmp/logs.json
restic-kit notify-email /tmp \
  --smtp-host smtp.gmail.com \
  --smtp-port 587 \
  --smtp-username user@gmail.com \
  --smtp-password "password" \
  --from user@gmail.com \
  --to admin@example.com

# Send HTTP notification
restic backup /data && restic-kit notify-http --url https://api.example.com/notify
```

## Building

```bash
go build -o restic-kit .
```

## Testing

```bash
go test ./...
```

## License

MIT License - see LICENSE file for details.