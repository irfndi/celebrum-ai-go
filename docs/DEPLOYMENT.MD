THIS IS UNEDIT VERSION, FROM OTHER PROJECT THAT SHARING SOME VPS, AFTER WE DEPLOY UPDATE/ADJUST THE CONTENT TO USE THIS PROJECT

# Deployment Guide for MeetsMatch (VPS)

## Overview
**Target Server:** `217.216.35.77`
**User:** `root` (for deployment ops), `meetsmatch` (runtime user)
**App Directory:** `/opt/apps/meetsmatch/code`
**Service Name:** `meetsmatch.service`

## Prerequisites
- SSH access to `root@217.216.35.77`
- Local installation of `rsync`
- Local project with `Makefile`

## Deployment Steps

### 1. Sync Code
Use `rsync` to upload the code. **Important:** Exclude local config files (`.env`) and virtual environments.

```bash
rsync -avz \
  --exclude '.venv' \
  --exclude 'venv' \
  --exclude '.git' \
  --exclude '__pycache__' \
  --exclude '.DS_Store' \
  --exclude '.pytest_cache' \
  --exclude '.ruff_cache' \
  --exclude '.mypy_cache' \
  --exclude '.env' \
  . root@217.216.35.77:/opt/apps/meetsmatch/code/
```

### 2. Fix Permissions
Ensure the runtime user owns the files.

```bash
ssh root@217.216.35.77 "chown -R meetsmatch:meetsmatch /opt/apps/meetsmatch/code"
```

### 3. Update Dependencies
Run `uv sync` as the `meetsmatch` user.

```bash
ssh root@217.216.35.77 "cd /opt/apps/meetsmatch/code && su -s /bin/bash meetsmatch -c 'UV_CACHE_DIR=/opt/apps/meetsmatch/.cache/uv uv sync'"
```

### 4. Run Migrations
Apply database changes using `alembic` (via `Makefile` or direct command).

```bash
ssh root@217.216.35.77 "cd /opt/apps/meetsmatch/code && su -s /bin/bash meetsmatch -c 'UV_CACHE_DIR=/opt/apps/meetsmatch/.cache/uv make migrate'"
# OR directly:
ssh root@217.216.35.77 "cd /opt/apps/meetsmatch/code && su -s /bin/bash meetsmatch -c 'UV_CACHE_DIR=/opt/apps/meetsmatch/.cache/uv uv run alembic upgrade head'"
```

### 5. Restart Service
Restart the systemd service to pick up changes.

```bash
ssh root@217.216.35.77 "systemctl restart meetsmatch.service"
```

### 6. Verify Deployment
Check the service status and logs.

```bash
ssh root@217.216.35.77 "systemctl status meetsmatch.service"
ssh root@217.216.35.77 "journalctl -u meetsmatch.service -f"
```

## Environment Variables
The `.env` file is located at `/opt/apps/meetsmatch/code/.env` on the server.
**DO NOT** overwrite it with your local `.env`.
If you need to add variables, edit it manually on the server:

```bash
ssh root@217.216.35.77 "nano /opt/apps/meetsmatch/code/.env"
# Then restart the service
ssh root@217.216.35.77 "systemctl restart meetsmatch.service"
```

## Troubleshooting
- **Database logs:** `journalctl -u meetsmatch-postgres.service`
- **Redis logs:** `journalctl -u meetsmatch-redis.service`
- **App logs:** `journalctl -u meetsmatch.service`

## Backup & Recovery
Backups are configured to run daily at 03:00 AM.

- **Script Location:** `/opt/apps/meetsmatch/scripts/backup.sh`
- **Backup Location:** `/opt/apps/meetsmatch/backups/`
- **Retention:** 7 days
- **Log File:** `/var/log/meetsmatch-backup.log`

### Manual Backup
To trigger a backup manually:

```bash
ssh root@217.216.35.77 "/opt/apps/meetsmatch/scripts/backup.sh"
```

### Restore
To restore from a backup:

```bash
# 1. Stop the application
ssh root@217.216.35.77 "systemctl stop meetsmatch.service"

# 2. Drop and recreate database (caution!)
# ssh root@217.216.35.77 "dropdb -h localhost -p 5433 -U meetsmatch meetsmatch && createdb -h localhost -p 5433 -U meetsmatch meetsmatch"

# 3. Import the backup
# ssh root@217.216.35.77 "gunzip -c /opt/apps/meetsmatch/backups/meetsmatch_db_YYYYMMDD_HHMMSS.sql.gz | psql -h localhost -p 5433 -U meetsmatch meetsmatch"

# 4. Restart application
ssh root@217.216.35.77 "systemctl start meetsmatch.service"
```

## Logging
Logging is handled by systemd journal.

- **View real-time logs:** `journalctl -u meetsmatch.service -f`
- **View logs since boot:** `journalctl -u meetsmatch.service -b`
- **View specific time range:** `journalctl -u meetsmatch.service --since "1 hour ago"`

