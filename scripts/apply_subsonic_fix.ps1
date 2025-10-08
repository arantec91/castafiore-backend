# Script to apply Subsonic authentication fix
# This script applies the database migration and updates existing users

Write-Host "=== Castafiore Backend - Subsonic Authentication Fix ===" -ForegroundColor Cyan
Write-Host ""

# Check if .env file exists
if (-not (Test-Path ".env")) {
    Write-Host "ERROR: .env file not found. Please create it from .env.example" -ForegroundColor Red
    exit 1
}

# Load database connection from .env
$envContent = Get-Content ".env"
$dbUrl = ($envContent | Where-Object { $_ -match "^DATABASE_URL=" }) -replace "^DATABASE_URL=", ""

if (-not $dbUrl) {
    Write-Host "ERROR: DATABASE_URL not found in .env file" -ForegroundColor Red
    exit 1
}

Write-Host "Database URL found: $dbUrl" -ForegroundColor Green
Write-Host ""

# Parse PostgreSQL connection string
# Format: postgres://user:password@host:port/database
if ($dbUrl -match "postgres://([^:]+):([^@]+)@([^:]+):(\d+)/(.+)") {
    $dbUser = $matches[1]
    $dbPass = $matches[2]
    $dbHost = $matches[3]
    $dbPort = $matches[4]
    $dbName = $matches[5]
    
    Write-Host "Parsed connection details:" -ForegroundColor Yellow
    Write-Host "  Host: $dbHost"
    Write-Host "  Port: $dbPort"
    Write-Host "  Database: $dbName"
    Write-Host "  User: $dbUser"
    Write-Host ""
} else {
    Write-Host "ERROR: Could not parse DATABASE_URL" -ForegroundColor Red
    exit 1
}

# Set PostgreSQL password environment variable
$env:PGPASSWORD = $dbPass

Write-Host "Step 1: Applying migration 004_add_subsonic_password.sql..." -ForegroundColor Cyan
$migrationFile = "migrations\004_add_subsonic_password.sql"

if (-not (Test-Path $migrationFile)) {
    Write-Host "ERROR: Migration file not found: $migrationFile" -ForegroundColor Red
    exit 1
}

# Apply migration using psql
$psqlCmd = "psql -h $dbHost -p $dbPort -U $dbUser -d $dbName -f `"$migrationFile`""
Write-Host "Executing: $psqlCmd" -ForegroundColor Gray

try {
    Invoke-Expression $psqlCmd
    Write-Host "Migration applied successfully!" -ForegroundColor Green
    Write-Host ""
} catch {
    Write-Host "ERROR applying migration: $_" -ForegroundColor Red
    Write-Host "You may need to install PostgreSQL client tools (psql)" -ForegroundColor Yellow
    Write-Host "Or apply the migration manually using your preferred database tool" -ForegroundColor Yellow
    Write-Host ""
}

Write-Host "Step 2: Updating existing users..." -ForegroundColor Cyan
$updateScript = "scripts\update_subsonic_passwords.sql"

if (-not (Test-Path $updateScript)) {
    Write-Host "ERROR: Update script not found: $updateScript" -ForegroundColor Red
    exit 1
}

$psqlCmd = "psql -h $dbHost -p $dbPort -U $dbUser -d $dbName -f `"$updateScript`""
Write-Host "Executing: $psqlCmd" -ForegroundColor Gray

try {
    Invoke-Expression $psqlCmd
    Write-Host ""
} catch {
    Write-Host "ERROR updating users: $_" -ForegroundColor Red
    Write-Host ""
}

Write-Host "=== Fix Applied ===" -ForegroundColor Green
Write-Host ""
Write-Host "IMPORTANT: You need to set the subsonic_password for each user:" -ForegroundColor Yellow
Write-Host "  1. For user 'antonio', run this SQL:" -ForegroundColor White
Write-Host "     UPDATE users SET subsonic_password = 'actual_password' WHERE username = 'antonio';" -ForegroundColor Gray
Write-Host ""
Write-Host "  2. For any other users, set their subsonic_password to their actual password" -ForegroundColor White
Write-Host ""
Write-Host "After setting passwords, restart the server:" -ForegroundColor Yellow
Write-Host "  go run cmd/server/main.go" -ForegroundColor Gray
Write-Host ""

# Clean up
Remove-Item Env:\PGPASSWORD