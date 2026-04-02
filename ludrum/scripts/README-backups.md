# Database Backups

These scripts back up and restore the local `ludrum` PostgreSQL database.

## Create a backup

From the repo root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\backup-db.ps1
```

This creates a timestamped dump in:

```text
.\backups\
```

## Restore a backup

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\restore-db.ps1 -BackupFile .\backups\ludrum-YYYYMMDD-HHMMSS.dump
```

## Current defaults

- Host: `localhost`
- Port: `5433`
- User: `postgres`
- Password: `password`
- Database: `ludrum`

## Override defaults

Example:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\backup-db.ps1 -Port 5432 -Password mysecret
```

## Important

- `restore-db.ps1` restores into the target database and uses `--clean`, so it replaces existing objects.
- Test a restore on a non-production database before relying on backups.
