# Deployment Guide for Celebrum AI (VPS)

## Overview

**User:** `root` (for deployment ops), `celebrum` (runtime user)
**App Directory:** `/root/apps/celebrum-ai/`
**Service Name:** `celebrum-ai.service`

## Prerequisites

- SSH access to your VPS server
- Local installation of `rsync`
- Go 1.24+ installed locally
- Local project with `Makefile`

## Deployment Steps

### 1. Build Binary Locally

Build the Go binary before deployment:

```bash
make build
```

### 2. Sync Code

Use `rsync` to upload the code. **Important:** Exclude local config files (`.env`) and development artifacts.

```bash
rsync -avz \
  --exclude '.git' \
  --exclude 'bin/' \
  --exclude '.cache' \
  --exclude 'coverage.out' \
  --exclude 'coverage.html' \
  --exclude '.env' \
  --exclude 'node_modules' \
  . root@YOUR_SERVER:/root/apps/celebrum-ai/code/
```

### 3. Fix Permissions

Ensure the runtime user owns the files.

```bash
ssh root@YOUR_SERVER "chown -R celebrum:celebrum /root/apps/celebrum-ai"
```

### 4. Run Migrations

Apply database changes using the migration script.

```bash
ssh root@YOUR_SERVER "cd /root/apps/celebrum-ai/code && make migrate"
```

### 5. Restart Service

Restart the systemd service to pick up changes.

```bash
ssh root@YOUR_SERVER "systemctl restart celebrum-ai.service"
```

### 6. Verify Deployment

Check the service status and logs.

```bash
ssh root@YOUR_SERVER "systemctl status celebrum-ai.service"
ssh root@YOUR_SERVER "journalctl -u celebrum-ai.service -f"
```

## Environment Variables

The `.env` file is located at `/root/apps/celebrum-ai/code/.env` on the server.
**DO NOT** overwrite it with your local `.env`.
If you need to add variables, edit it manually on the server:

```bash
ssh root@YOUR_SERVER "nano /root/apps/celebrum-ai/code/.env"
# Then restart the service
ssh root@YOUR_SERVER "systemctl restart celebrum-ai.service"
```

## Troubleshooting

- **Database logs:** `journalctl -u celebrum-ai-postgres.service`
- **Redis logs:** `journalctl -u celebrum-ai-redis.service`
- **App logs:** `journalctl -u celebrum-ai.service`

## Backup & Recovery

Backups are configured to run daily at 03:00 AM.

- **Script Location:** `/root/apps/celebrum-ai/scripts/backup.sh`
- **Backup Location:** `/root/apps/celebrum-ai/backups/`
- **Retention:** 7 days
- **Log File:** `/var/log/celebrum-ai-backup.log`

### Manual Backup

To trigger a backup manually:

```bash
ssh root@YOUR_SERVER "/root/apps/celebrum-ai/scripts/backup.sh"
```

### Restore

To restore from a backup:

```bash
# 1. Stop the application
ssh root@YOUR_SERVER "systemctl stop celebrum-ai.service"

# 2. Drop and recreate database (caution!)
# ssh root@YOUR_SERVER "dropdb -h localhost -p 5433 -U celebrum celebrum_db && createdb -h localhost -p 5433 -U celebrum celebrum_db"

# 3. Import the backup
# ssh root@YOUR_SERVER "gunzip -c /root/apps/celebrum-ai/backups/celebrum_db_YYYYMMDD_HHMMSS.sql.gz | psql -h localhost -p 5433 -U celebrum celebrum_db"

# 4. Restart application
ssh root@YOUR_SERVER "systemctl start celebrum-ai.service"
```

## Logging

Logging is handled by systemd journal.

- **View real-time logs:** `journalctl -u celebrum-ai.service -f`
- **View logs since boot:** `journalctl -u celebrum-ai.service -b`
- **View specific time range:** `journalctl -u celebrum-ai.service --since "1 hour ago"`

## See Also

- [VPS Systemd Guide](./vps-systemd-guide.md) - Comprehensive systemd setup guide
- [Deployment Best Practices](./DEPLOYMENT_BEST_PRACTICES.md) - Best practices for deployment

