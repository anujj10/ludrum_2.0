param(
[Parameter(Mandatory = $true)]
[string]$BackupFile,
[string]$HostName = "localhost",
[string]$Port = "5432",
[string]$UserName = "postgres",
[string]$Password = "password",
[string]$Database = "ludrum"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path $BackupFile)) {
  throw "Backup file not found: $BackupFile"
}

if (-not (Get-Command pg_restore -ErrorAction SilentlyContinue)) {
  throw "pg_restore was not found in PATH. Install PostgreSQL client tools first."
}

$env:PGPASSWORD = $Password

try {
  pg_restore `
    --host $HostName `
    --port $Port `
    --username $UserName `
    --dbname $Database `
    --clean `
    --if-exists `
    --no-owner `
    --no-privileges `
    $BackupFile

  Write-Host "Restore completed from:" $BackupFile
}
finally {
  Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
}
