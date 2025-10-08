# Script to test Subsonic authentication
param(
    [string]$Username = "admin",
    [string]$Password = "admin123",
    [string]$ServerUrl = "http://localhost:8080"
)

Write-Host "=== Testing Subsonic Authentication ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Server: $ServerUrl" -ForegroundColor Yellow
Write-Host "Username: $Username" -ForegroundColor Yellow
Write-Host ""

# Generate random salt
$salt = -join ((48..57) + (97..102) | Get-Random -Count 6 | ForEach-Object {[char]$_})
Write-Host "Generated salt: $salt" -ForegroundColor Gray

# Calculate MD5 token
$md5 = [System.Security.Cryptography.MD5]::Create()
$hash = $md5.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($Password + $salt))
$token = [System.BitConverter]::ToString($hash).Replace("-", "").ToLower()

Write-Host "Calculated token: $token" -ForegroundColor Gray
Write-Host ""

# Test 1: Ping (no auth required)
Write-Host "Test 1: Ping endpoint (no auth required)..." -ForegroundColor Cyan
$pingUrl = "$ServerUrl/rest/ping.view?u=$Username`&t=$token`&s=$salt`&v=1.16.1`&c=TestClient`&f=json"
try {
    $response = Invoke-RestMethod -Uri $pingUrl -Method Get
    if ($response.'subsonic-response'.status -eq "ok") {
        Write-Host "✓ Ping successful!" -ForegroundColor Green
    } else {
        Write-Host "✗ Ping failed: $($response.'subsonic-response'.error.message)" -ForegroundColor Red
    }
} catch {
    Write-Host "✗ Ping failed: $_" -ForegroundColor Red
}
Write-Host ""

# Test 2: Get License (no auth required)
Write-Host "Test 2: Get License endpoint (no auth required)..." -ForegroundColor Cyan
$licenseUrl = "$ServerUrl/rest/getLicense.view?u=$Username`&t=$token`&s=$salt`&v=1.16.1`&c=TestClient`&f=json"
try {
    $response = Invoke-RestMethod -Uri $licenseUrl -Method Get
    if ($response.'subsonic-response'.status -eq "ok") {
        Write-Host "✓ License check successful!" -ForegroundColor Green
        Write-Host "  License valid: $($response.'subsonic-response'.license.valid)" -ForegroundColor Gray
    } else {
        Write-Host "✗ License check failed: $($response.'subsonic-response'.error.message)" -ForegroundColor Red
    }
} catch {
    Write-Host "✗ License check failed: $_" -ForegroundColor Red
}
Write-Host ""

# Test 3: Get Music Folders (requires auth)
Write-Host "Test 3: Get Music Folders endpoint (requires auth)..." -ForegroundColor Cyan
$foldersUrl = "$ServerUrl/rest/getMusicFolders.view?u=$Username&t=$token&s=$salt&v=1.16.1&c=TestClient&f=json"
try {
    $response = Invoke-RestMethod -Uri $foldersUrl -Method Get
    if ($response.'subsonic-response'.status -eq "ok") {
        Write-Host "✓ Authentication successful!" -ForegroundColor Green
        $folders = $response.'subsonic-response'.musicFolders.musicFolder
        if ($folders) {
            Write-Host "  Found $($folders.Count) music folder(s):" -ForegroundColor Gray
            foreach ($folder in $folders) {
                Write-Host "    - ID: $($folder.id), Name: $($folder.name)" -ForegroundColor Gray
            }
        }
    } else {
        Write-Host "✗ Authentication failed: $($response.'subsonic-response'.error.message)" -ForegroundColor Red
        Write-Host "  Error code: $($response.'subsonic-response'.error.code)" -ForegroundColor Red
    }
} catch {
    Write-Host "✗ Request failed: $_" -ForegroundColor Red
}
Write-Host ""

# Test 4: Get Artists (requires auth)
Write-Host "Test 4: Get Artists endpoint (requires auth)..." -ForegroundColor Cyan
$artistsUrl = "$ServerUrl/rest/getArtists.view?u=$Username&t=$token&s=$salt&v=1.16.1&c=TestClient&f=json"
try {
    $response = Invoke-RestMethod -Uri $artistsUrl -Method Get
    if ($response.'subsonic-response'.status -eq "ok") {
        Write-Host "✓ Get Artists successful!" -ForegroundColor Green
        $indexes = $response.'subsonic-response'.artists.index
        if ($indexes) {
            $totalArtists = 0
            foreach ($index in $indexes) {
                if ($index.artist) {
                    $totalArtists += $index.artist.Count
                }
            }
            Write-Host "  Found $totalArtists artist(s)" -ForegroundColor Gray
        }
    } else {
        Write-Host "✗ Get Artists failed: $($response.'subsonic-response'.error.message)" -ForegroundColor Red
    }
} catch {
    Write-Host "✗ Request failed: $_" -ForegroundColor Red
}
Write-Host ""

Write-Host "=== Test Complete ===" -ForegroundColor Green
Write-Host ""
Write-Host "To test with other users:" -ForegroundColor Yellow
Write-Host "  .\test_subsonic_auth.ps1 -Username antonio -Password 150291" -ForegroundColor Gray
Write-Host "  .\test_subsonic_auth.ps1 -Username fredyaran -Password 'Aleida2001+'" -ForegroundColor Gray