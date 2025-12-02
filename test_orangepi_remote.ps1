#!/usr/bin/env pwsh
# OrangePi Remote API Test Script

param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$Username = "admin",
  [string]$Password = "admin123",
  [string]$BuildingISmartID = "0314100",
  [int]$OrangePiId = 61
)

[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

$PASSED = 0
$FAILED = 0
$TOTAL = 0
$staffToken = $null

function Test-Endpoint {
  param(
    [string]$Name,
    [string]$Method,
    [string]$Uri,
    [hashtable]$Headers = @{},
    [string]$Body = $null
  )
    
  $script:TOTAL++
  Write-Host "[$script:TOTAL] $Name" -ForegroundColor Cyan
  Write-Host "    -> $Method $Uri" -ForegroundColor DarkGray
    
  try {
    $params = @{
      Uri         = $Uri
      Method      = $Method
      ContentType = "application/json"
      ErrorAction = "Stop"
    }
        
    if ($Headers -and $Headers.Count -gt 0) {
      $params["Headers"] = $Headers
    }
        
    if ($Body) {
      $params["Body"] = $Body
    }
        
    $response = Invoke-RestMethod @params
    Write-Host "    [OK] SUCCESS" -ForegroundColor Green
    
    # 显示响应数据
    $respJson = ($response | ConvertTo-Json -Depth 10 -Compress)
    if ($respJson.Length -gt 800) {
      Write-Host "    Response: $($respJson.Substring(0, 800))..." -ForegroundColor DarkGray
    }
    else {
      Write-Host "    Response: $respJson" -ForegroundColor DarkGray
    }
    
    $script:PASSED++
        
    return $response
  }
  catch {
    Write-Host "    [FAIL] $($_.Exception.Message)" -ForegroundColor Red
    if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
      Write-Host "    Error: $($_.ErrorDetails.Message)" -ForegroundColor Red
    }
    $script:FAILED++
    return $null
  }
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "OrangePi Remote API Test" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Base URL: $BaseUrl" -ForegroundColor Gray
Write-Host "Building: $BuildingISmartID" -ForegroundColor Gray
Write-Host "OrangePi ID: $OrangePiId" -ForegroundColor Gray
Write-Host ""

# Step 1: Admin Login
Write-Host "========== Step 1: Admin Login ==========" -ForegroundColor Yellow
$loginBody = @{username = $Username; password = $Password } | ConvertTo-Json
$loginResp = Test-Endpoint "Admin Login" "POST" "$BaseUrl/api/auth/login" @{} $loginBody

if ($null -eq $loginResp) {
  Write-Host "[ERROR] Login failed!" -ForegroundColor Red
  exit 1
}

$jwtToken = $loginResp.data.accessToken
$authHeaders = @{"Authorization" = "Bearer $jwtToken" }
Write-Host ""

# Step 2: Generate Staff Token
Write-Host "========== Step 2: Generate Staff Token ==========" -ForegroundColor Yellow
$staffTokenBody = @{ismartid = $BuildingISmartID; is_staff = $true } | ConvertTo-Json
$staffTokenResp = Test-Endpoint "Staff Token" "POST" "$BaseUrl/api/auth/public" @{} $staffTokenBody

if ($staffTokenResp -and $staffTokenResp.data) {
  $staffToken = $staffTokenResp.data.token
  Write-Host "  Token: $($staffToken.Substring(0, 50))..." -ForegroundColor Cyan
  Write-Host "  OrangePis: $($staffTokenResp.data.orangepis.Count)" -ForegroundColor Cyan
  
  foreach ($opi in $staffTokenResp.data.orangepis) {
    Write-Host "  [OrangePi] $($opi.orangepi_name) (ID: $($opi.orangepi_id))" -ForegroundColor Yellow
    Write-Host "    URLs: $($opi.urls.Count)" -ForegroundColor Gray
    foreach ($url in $opi.urls) {
      Write-Host "      - $url" -ForegroundColor DarkGray
    }
  }
}
Write-Host ""

# Step 3: Generate User Token
Write-Host "========== Step 3: Generate User Token ==========" -ForegroundColor Yellow
$userTokenBody = @{ismartid = $BuildingISmartID; is_staff = $false } | ConvertTo-Json
$userTokenResp = Test-Endpoint "User Token" "POST" "$BaseUrl/api/auth/public" @{} $userTokenBody
Write-Host ""

# Step 4: Remote Health Check
Write-Host "========== Step 4: Remote Health Check ==========" -ForegroundColor Yellow
$healthUri = "$BaseUrl/api/orangepi/remote/health?id=$OrangePiId"
$healthResp = Test-Endpoint "Health Check" "GET" $healthUri $authHeaders

if ($healthResp -and $healthResp.data) {
  Write-Host "  Status: $($healthResp.data.status)" -ForegroundColor Cyan
  Write-Host "  Service: $($healthResp.data.service)" -ForegroundColor Gray
  if ($healthResp.data.docker_services) {
    Write-Host "  Docker Services:" -ForegroundColor Gray
    $healthResp.data.docker_services.PSObject.Properties | ForEach-Object {
      $statusIcon = if ($_.Value) { "[OK]" } else { "[FAIL]" }
      Write-Host "    $statusIcon $($_.Name)" -ForegroundColor DarkGray
    }
  }
}
Write-Host ""

# Step 5: Remote Device Info
Write-Host "========== Step 5: Remote Device Info ==========" -ForegroundColor Yellow
if ($staffToken) {
  $deviceInfoUri = "$BaseUrl/api/orangepi/remote/info?id=$OrangePiId" + "&" + "token=$staffToken"
  $deviceInfoResp = Test-Endpoint "Device Info" "GET" $deviceInfoUri $authHeaders

  if ($deviceInfoResp -and $deviceInfoResp.data) {
    Write-Host "  Device ID: $($deviceInfoResp.data.device_id)" -ForegroundColor Cyan
    Write-Host "  MediaMTX Version: $($deviceInfoResp.data.mediamtx_version)" -ForegroundColor Gray
    Write-Host "  FRP Server: $($deviceInfoResp.data.frpc_server)" -ForegroundColor Gray
    Write-Host "  Auth Port: $($deviceInfoResp.data.frpc_auth_remote_port)" -ForegroundColor Gray
    Write-Host "  SSH Port: $($deviceInfoResp.data.frpc_ssh_remote_port)" -ForegroundColor Gray
    Write-Host "  Channels: $($deviceInfoResp.data.available_channels -join ', ')" -ForegroundColor Gray
    Write-Host "  Status: $($deviceInfoResp.data.status)" -ForegroundColor Gray
  }
}
else {
  Write-Host "  [SKIP] Need Staff Token" -ForegroundColor Yellow
}
Write-Host ""

# Step 6: List MediaMTX Paths
if ($staffToken) {
  Write-Host "========== Step 6: List MediaMTX Paths ==========" -ForegroundColor Yellow
  $pathsUri = "$BaseUrl/api/orangepi/remote/paths?id=$OrangePiId" + "&" + "token=$staffToken" + "&" + "page=0" + "&" + "items_per_page=50"
  $pathsResp = Test-Endpoint "List Paths" "GET" $pathsUri $authHeaders
  
  if ($pathsResp -and $pathsResp.data) {
    Write-Host "  Total: $($pathsResp.data.itemsTotal)" -ForegroundColor Cyan
    Write-Host "  Page: $($pathsResp.data.itemsPage)" -ForegroundColor Gray
    Write-Host "  Paths:" -ForegroundColor Gray
    $pathsResp.data.items | ForEach-Object {
      $statusIcon = if ($_.ready) { "[OK]" } else { "[FAIL]" }
      Write-Host "    $statusIcon $($_.name) - Ready: $($_.ready)" -ForegroundColor DarkGray
    }
  }
  Write-Host ""
}
else {
  Write-Host "========== Step 6: List MediaMTX Paths ==========" -ForegroundColor Yellow
  Write-Host "  [SKIP] Need Staff Token" -ForegroundColor Yellow
  Write-Host ""
}

# Step 7: Get Path Detail
if ($staffToken) {
  Write-Host "========== Step 7: Get Path Detail ==========" -ForegroundColor Yellow
  $pathName = "channel1"
  $pathDetailUri = "$BaseUrl/api/orangepi/remote/paths/detail?id=$OrangePiId" + "&" + "token=$staffToken" + "&" + "name=$pathName"
  $pathDetailResp = Test-Endpoint "Get Path: $pathName" "GET" $pathDetailUri $authHeaders
  
  if ($pathDetailResp -and $pathDetailResp.data) {
    Write-Host "  Name: $($pathDetailResp.data.name)" -ForegroundColor Cyan
    Write-Host "  Config:" -ForegroundColor Gray
    if ($pathDetailResp.data.conf.source) {
      Write-Host "    Source: $($pathDetailResp.data.conf.source)" -ForegroundColor DarkGray
    }
    if ($null -ne $pathDetailResp.data.conf.record) {
      Write-Host "    Record: $($pathDetailResp.data.conf.record)" -ForegroundColor DarkGray
    }
  }
  Write-Host ""
}
else {
  Write-Host "========== Step 7: Get Path Detail ==========" -ForegroundColor Yellow
  Write-Host "  [SKIP] Need Staff Token" -ForegroundColor Yellow
  Write-Host ""
}

# Step 8-10: Add, Update, Delete Path
if ($staffToken) {
  Write-Host "========== Step 8: Add Path ==========" -ForegroundColor Yellow
  $testPathName = "test_path_$(Get-Random -Minimum 1000 -Maximum 9999)"
  $addPathBody = @{
    name   = $testPathName
    config = @{
      source             = "rtsp://admin:pwd@192.168.1.100:554/test"
      sourceOnDemand     = $false
      record             = $true
      recordPath         = "./recordings/%path/%Y-%m-%d_%H-%M-%S"
      recordPartDuration = "30m"
    }
  } | ConvertTo-Json -Depth 4
  
  $addPathUri = "$BaseUrl/api/orangepi/remote/paths?id=$OrangePiId" + "&" + "token=$staffToken"
  $addPathResp = Test-Endpoint "Add Path: $testPathName" "POST" $addPathUri $authHeaders $addPathBody
  
  if ($addPathResp -and $addPathResp.data) {
    Write-Host "  Action: $($addPathResp.data.action)" -ForegroundColor Cyan
    Write-Host "  Name: $($addPathResp.data.name)" -ForegroundColor Gray
    Write-Host "  Status Code: $($addPathResp.data.status_code)" -ForegroundColor Gray
  }
  Write-Host ""
  
  if ($addPathResp) {
    Write-Host "========== Step 9: Update Path ==========" -ForegroundColor Yellow
    $updatePathBody = @{config = @{record = $true; recordPartDuration = "10m" } } | ConvertTo-Json -Depth 4
    $updatePathUri = "$BaseUrl/api/orangepi/remote/paths?id=$OrangePiId" + "&" + "token=$staffToken" + "&" + "name=$testPathName"
    $updatePathResp = Test-Endpoint "Update Path: $testPathName" "PATCH" $updatePathUri $authHeaders $updatePathBody
    
    if ($updatePathResp -and $updatePathResp.data) {
      Write-Host "  Action: $($updatePathResp.data.action)" -ForegroundColor Cyan
      Write-Host "  Status Code: $($updatePathResp.data.status_code)" -ForegroundColor Gray
    }
    Write-Host ""
    
    Write-Host "========== Step 10: Delete Path ==========" -ForegroundColor Yellow
    $deletePathUri = "$BaseUrl/api/orangepi/remote/paths?id=$OrangePiId" + "&" + "token=$staffToken" + "&" + "name=$testPathName"
    $deletePathResp = Test-Endpoint "Delete Path: $testPathName" "DELETE" $deletePathUri $authHeaders
    
    if ($deletePathResp -and $deletePathResp.data) {
      Write-Host "  Action: $($deletePathResp.data.action)" -ForegroundColor Cyan
      Write-Host "  Status Code: $($deletePathResp.data.status_code)" -ForegroundColor Gray
    }
    Write-Host ""
  }
}
else {
  Write-Host "========== Step 8-10: Path Management ==========" -ForegroundColor Yellow
  Write-Host "  [SKIP] Need Staff Token" -ForegroundColor Yellow
  Write-Host ""
}

# Summary
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Test Summary" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "Total: $TOTAL" -ForegroundColor White
Write-Host "Passed: $PASSED" -ForegroundColor Green
Write-Host "Failed: $FAILED" -ForegroundColor Red

if ($FAILED -eq 0) {
  Write-Host "`n[SUCCESS] All tests passed!" -ForegroundColor Green
}
else {
  Write-Host "`n[WARNING] Some tests failed" -ForegroundColor Yellow
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

exit $FAILED
