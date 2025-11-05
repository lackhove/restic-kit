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