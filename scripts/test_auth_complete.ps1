# Complete Authentication Test Script for Castafiore Backend
# Tests all authentication scenarios

$ServerUrl = "http://localhost:8080"
$ValidUser = "antonio"
$ValidPassword = "150291"
$InvalidUser = "invaliduser"
$InvalidPassword = "wrongpassword"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Castafiore Authentication Test Suite" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Function to calculate MD5 token
function Get-SubsonicToken {
    param(
        [string]$Password,
        [string]$Salt
    )
    $md5 = [System.Security.Cryptography.MD5]::Create()
    $hash = $md5.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($Password + $Salt))
    return [System.BitConverter]::ToString($hash).Replace("-", "").ToLower()
}

# Function to test endpoint
function Test-SubsonicEndpoint {
    param(
        [string]$TestName,
        [string]$Endpoint,
        [string]$Username,
        [string]$Password,
        [bool]$ShouldSucceed
    )
    
    Write-Host "Test: $TestName" -ForegroundColor Yellow
    
    $salt = [System.Guid]::NewGuid().ToString().Substring(0, 6)
    $token = Get-SubsonicToken -Password $Password -Salt $salt
    
    $url = "$ServerUrl$Endpoint`?u=$Username`&t=$token`&s=$salt`&v=1.16.1`&c=TestClient`&f=json"
    
    try {
        $response = Invoke-RestMethod -Uri $url -Method Get -ErrorAction Stop
        
        if ($response.'subsonic-response'.status -eq "ok") {
            if ($ShouldSucceed) {
                Write-Host "  [PASS] Authentication succeeded (expected)" -ForegroundColor Green
                return $true
            } else {
                Write-Host "  [FAIL] Authentication succeeded (should have failed)" -ForegroundColor Red
                return $false
            }
        } else {
            if (-not $ShouldSucceed) {
                Write-Host "  [PASS] Authentication failed (expected)" -ForegroundColor Green
                Write-Host "    Error: $($response.'subsonic-response'.error.message)" -ForegroundColor Gray
                return $true
            } else {
                Write-Host "  [FAIL] Authentication failed (should have succeeded)" -ForegroundColor Red
                Write-Host "    Error: $($response.'subsonic-response'.error.message)" -ForegroundColor Gray
                return $false
            }
        }
    } catch {
        Write-Host "  [FAIL] Request error: $_" -ForegroundColor Red
        return $false
    }
}

# Test counters
$totalTests = 0
$passedTests = 0

Write-Host "Testing Ping Endpoint" -ForegroundColor Cyan
Write-Host "---------------------" -ForegroundColor Cyan

# Test 1: Ping with valid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "Ping with valid credentials" -Endpoint "/rest/ping.view" -Username $ValidUser -Password $ValidPassword -ShouldSucceed $true) {
    $passedTests++
}
Write-Host ""

# Test 2: Ping with invalid user
$totalTests++
if (Test-SubsonicEndpoint -TestName "Ping with invalid user" -Endpoint "/rest/ping.view" -Username $InvalidUser -Password $ValidPassword -ShouldSucceed $false) {
    $passedTests++
}
Write-Host ""

# Test 3: Ping with invalid password
$totalTests++
if (Test-SubsonicEndpoint -TestName "Ping with invalid password" -Endpoint "/rest/ping.view" -Username $ValidUser -Password $InvalidPassword -ShouldSucceed $false) {
    $passedTests++
}
Write-Host ""

Write-Host "Testing GetLicense Endpoint" -ForegroundColor Cyan
Write-Host "---------------------------" -ForegroundColor Cyan

# Test 4: GetLicense with valid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "GetLicense with valid credentials" -Endpoint "/rest/getLicense.view" -Username $ValidUser -Password $ValidPassword -ShouldSucceed $true) {
    $passedTests++
}
Write-Host ""

# Test 5: GetLicense with invalid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "GetLicense with invalid credentials" -Endpoint "/rest/getLicense.view" -Username $InvalidUser -Password $InvalidPassword -ShouldSucceed $false) {
    $passedTests++
}
Write-Host ""

Write-Host "Testing GetMusicFolders Endpoint" -ForegroundColor Cyan
Write-Host "--------------------------------" -ForegroundColor Cyan

# Test 6: GetMusicFolders with valid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "GetMusicFolders with valid credentials" -Endpoint "/rest/getMusicFolders.view" -Username $ValidUser -Password $ValidPassword -ShouldSucceed $true) {
    $passedTests++
}
Write-Host ""

# Test 7: GetMusicFolders with invalid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "GetMusicFolders with invalid credentials" -Endpoint "/rest/getMusicFolders.view" -Username $InvalidUser -Password $InvalidPassword -ShouldSucceed $false) {
    $passedTests++
}
Write-Host ""

Write-Host "Testing GetArtists Endpoint" -ForegroundColor Cyan
Write-Host "---------------------------" -ForegroundColor Cyan

# Test 8: GetArtists with valid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "GetArtists with valid credentials" -Endpoint "/rest/getArtists.view" -Username $ValidUser -Password $ValidPassword -ShouldSucceed $true) {
    $passedTests++
}
Write-Host ""

# Test 9: GetArtists with invalid credentials
$totalTests++
if (Test-SubsonicEndpoint -TestName "GetArtists with invalid credentials" -Endpoint "/rest/getArtists.view" -Username $InvalidUser -Password $InvalidPassword -ShouldSucceed $false) {
    $passedTests++
}
Write-Host ""

# Summary
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Total Tests: $totalTests" -ForegroundColor White
Write-Host "Passed: $passedTests" -ForegroundColor Green
Write-Host "Failed: $($totalTests - $passedTests)" -ForegroundColor Red

if ($passedTests -eq $totalTests) {
    Write-Host ""
    Write-Host "ALL TESTS PASSED!" -ForegroundColor Green
    Write-Host "Authentication is working correctly." -ForegroundColor Green
    exit 0
} else {
    Write-Host ""
    Write-Host "SOME TESTS FAILED" -ForegroundColor Red
    Write-Host "Please review the authentication implementation." -ForegroundColor Red
    exit 1
}