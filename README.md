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

Send an email notification with backup report details from JSON logs in a directory. The command parses JSON logs from the directory and generates a formatted email with backup summaries, snapshot tables, error attachments, and repository check status.

### notify-http

Perform a single HTTP GET request to notify an external service.

### wait-online

Wait for network connectivity by checking if a URL is reachable with exponential backoff.

### audit

Audit restic snapshots for size anomalies. Checks for unusual size changes between the two most recent snapshots per path. Sends email notifications for any failures.

## Remote Backup Execution

This section describes how to set up secure remote backup execution where the backup script runs on a remote host but executes the actual backup via SSH on the source system.

### Architecture

- **Remote Host**: Runs the backup script, handles notifications, repository checks
- **Source Host**: Only executes the `restic backup` command via SSH
- **Self-Contained**: The restic binary is automatically copied to the source system, eliminating the need to install restic on source hosts
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
RESTIC="/usr/local/bin/restic"                    # Local restic path (for check/snapshots)
REMOTE_RESTIC_SRC="/usr/local/bin/restic"         # Path to restic binary on remote host (copied to source)
SSH_HOST="backup-source"                          # SSH config host alias
SSH_PORT="22"                                     # SSH port for source system
```

**Note:** `REMOTE_RESTIC_SRC` specifies the restic binary on the remote host that gets copied to the source system. This allows the remote and source hosts to have different architectures (e.g., x86 remote host backing up ARM source systems).

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