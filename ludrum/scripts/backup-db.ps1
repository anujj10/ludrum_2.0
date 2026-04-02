param(
  [string]$HostName = "localhost",
  [string]$Port = "5432",
  [string]$UserName = "postgres",
  [string]$Password = "password",
  [string]$Database = "ludrum",
  [string]$OutputDir = ".\\backups",
  [string]$LogDir = ".\\backups\\logs"
)

$ErrorActionPreference = "Stop"
$PSNativeCommandUseErrorActionPreference = $false

if (-not (Get-Command pg_dump -ErrorAction SilentlyContinue)) {
  throw "pg_dump was not found in PATH. Install PostgreSQL client tools first."
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"
$fileName = "${Database}-${timestamp}.dump"
$outputPath = Join-Path $OutputDir $fileName
$metaPath = Join-Path $OutputDir "${Database}-${timestamp}.meta.txt"
$logPath = Join-Path $LogDir "backup-${timestamp}.log"

$env:PGPASSWORD = $Password

try {
  $rowCounts = & psql `
    --host $HostName `
    --port $Port `
    --username $UserName `
    --dbname $Database `
    --tuples-only `
    --no-align `
    --command "select 'market_snapshots=' || (select count(*) from market_snapshots); select 'option_chain=' || (select count(*) from option_chain); select 'option_features=' || (select count(*) from option_features);" 2>&1

  if ($LASTEXITCODE -ne 0) {
    throw "Failed to query row counts before backup: $rowCounts"
  }

  $stdoutPath = Join-Path $LogDir "backup-${timestamp}.stdout.log"
  $stderrPath = Join-Path $LogDir "backup-${timestamp}.stderr.log"
  $argList = @(
    "--host", $HostName,
    "--port", $Port,
    "--username", $UserName,
    "--format", "custom",
    "--file", $outputPath,
    $Database
  )

  $dumpProcess = Start-Process `
    -FilePath "pg_dump" `
    -ArgumentList $argList `
    -NoNewWindow `
    -Wait `
    -PassThru `
    -RedirectStandardOutput $stdoutPath `
    -RedirectStandardError $stderrPath

  @(
    "stdout_log=$stdoutPath"
    "stderr_log=$stderrPath"
    (Get-Content $stdoutPath -ErrorAction SilentlyContinue)
    (Get-Content $stderrPath -ErrorAction SilentlyContinue)
  ) | Set-Content -Path $logPath

  if ($dumpProcess.ExitCode -ne 0) {
    throw "pg_dump failed. See log: $logPath"
  }

  @(
    "created_at=$timestamp"
    "database=$Database"
    "host=$HostName"
    "port=$Port"
    "dump_file=$outputPath"
    "log_file=$logPath"
    $rowCounts
  ) | Set-Content -Path $metaPath

  Write-Host "Backup created:" $outputPath
  Write-Host "Metadata saved:" $metaPath
  Write-Host "Log saved:" $logPath
}
finally {
  Remove-Item Env:PGPASSWORD -ErrorAction SilentlyContinue
}
