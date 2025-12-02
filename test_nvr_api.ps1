#!/usr/bin/env pwsh
# ICCTV HTTP Service - Complete API Test Script
# 测试所有 API 接口

param(
  [string]$BaseUrl = "http://127.0.0.1:8080",
  [string]$Username = "admin",
  [string]$Password = "admin123",
  [int]$RemoteOrangePiId = 2
)

# 设置 UTF-8 编码以正确显示中文
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8
if ($PSVersionTable.PSVersion.Major -ge 6) {
  [Console]::InputEncoding = [System.Text.Encoding]::UTF8
}
# Windows PowerShell 兼容性处理
if ($PSVersionTable.PSVersion.Major -lt 6) {
  chcp 65001 | Out-Null
}

$PASSED = 0
$FAILED = 0
$TOTAL = 0

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
  if ($Headers -and $Headers.Count -gt 0) {
    $headersJson = ($Headers | ConvertTo-Json -Depth 5)
    Write-Host "    Headers: $headersJson" -ForegroundColor DarkGray
  }
  if ($Body) {
    Write-Host "    Body:" -ForegroundColor DarkGray
    Write-Host $Body -ForegroundColor DarkGray
  }
    
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
    Write-Host "    ✅ SUCCESS" -ForegroundColor Green
    $respJson = ($response | ConvertTo-Json -Depth 10)
    Write-Host "    Response:" -ForegroundColor DarkGray
    Write-Host $respJson -ForegroundColor DarkGray
    $script:PASSED++
        
    return $response
  }
  catch {
    Write-Host "    ❌ FAILED: $($_.Exception.Message)" -ForegroundColor Red
    if ($_.Exception.Response -and $_.Exception.Response.ContentLength -gt 0) {
      try {
        $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
        $errorBody = $reader.ReadToEnd()
        if ($errorBody) {
          Write-Host "    Error Response:" -ForegroundColor Red
          Write-Host $errorBody -ForegroundColor Red
        }
      }
      catch {}
    }
    $script:FAILED++
    return $null
  }
}

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "ICCTV HTTP Service - Complete API Test" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan
Write-Host ""

# ==================== 认证接口 (Auth) ====================
Write-Host "========== 认证接口 ==========" -ForegroundColor Yellow
$loginBody = @{username = $Username; password = $Password } | ConvertTo-Json
$loginResp = Test-Endpoint "Admin Login" "POST" "$BaseUrl/api/auth/login" @{} $loginBody
if ($null -eq $loginResp) { Write-Host "❌ Login failed, abort tests!" -ForegroundColor Red; exit 1 }
$jwtToken = $loginResp.data.accessToken
$authHeaders = @{"Authorization" = "Bearer $jwtToken" }
$OutputEncoding = [System.Text.Encoding]::UTF8

# ==================== 管理员接口 (Admin) ====================
Write-Host "========== 管理员账户管理 ==========" -ForegroundColor Yellow
$testId = Get-Random -Minimum 1000 -Maximum 9999
$adminBody = @{username = "test_admin_$testId"; password = "testpwd123" } | ConvertTo-Json
$adminResp = Test-Endpoint "Create Admin" "POST" "$BaseUrl/api/admin" $authHeaders $adminBody
$adminId = if ($adminResp) { $adminResp.data.id } else { 0 }
Test-Endpoint "查询管理员" "GET" "$BaseUrl/api/admin" $authHeaders
$updateAdminBody = @{id = $adminId; username = "updated_admin_$testId" } | ConvertTo-Json
Test-Endpoint "更新管理员" "PUT" "$BaseUrl/api/admin" $authHeaders $updateAdminBody
$delAdminBody = @{id = $adminId } | ConvertTo-Json
Test-Endpoint "删除管理员" "DELETE" "$BaseUrl/api/admin" $authHeaders $delAdminBody

# ==================== 建筑信息管理 (Building) ====================
Write-Host "========== 建筑信息管理 ==========" -ForegroundColor Yellow
$buildId = Get-Random -Minimum 10000 -Maximum 90000
$buildBody = @{ismartid = "test_building_$buildId"; name = "测试楼栋_$buildId"; remark = "备注_$buildId" } | ConvertTo-Json
$buildResp = Test-Endpoint "创建建筑" "POST" "$BaseUrl/api/building" $authHeaders $buildBody
$buildingId = if ($buildResp) { $buildResp.data.id }else { 0 }
Test-Endpoint "查询建筑列表" "GET" "$BaseUrl/api/building" $authHeaders
$updateBuildBody = @{name = "已更名楼栋_$buildId" } | ConvertTo-Json
Test-Endpoint "更新建筑" "PUT" "$BaseUrl/api/building?id=$buildingId" $authHeaders $updateBuildBody
$delBuildBody = @{id = $buildingId } | ConvertTo-Json
Write-Host "    Building 已创建，ID: $buildingId，用于后续绑定测试`n" -ForegroundColor DarkGray

# ==================== OrangePi 设备 (Device) ====================
Write-Host "========== OrangePi 设备管理 ==========" -ForegroundColor Yellow
$deviceId = Get-Random -Minimum 100 -Maximum 999
$devBody = @{ismartid = "test_building_$buildId"; name = "TestDevice_$deviceId"; icctv_auth_service_remote_port = 30000 + $deviceId; ssh_remote_port = 20000 + $deviceId; is_active = $true } | ConvertTo-Json
$devResp = Test-Endpoint "创建设备" "POST" "$BaseUrl/api/device" $authHeaders $devBody
$devID = if ($devResp) { $devResp.data.id }else { 0 }
Test-Endpoint "查询设备" "GET" "$BaseUrl/api/device" $authHeaders
$updateDevBody = @{name = "UpdatedDevice_$deviceId" } | ConvertTo-Json
Test-Endpoint "更新设备" "PUT" "$BaseUrl/api/device?id=$devID" $authHeaders $updateDevBody
$delDevBody = @{id = $devID } | ConvertTo-Json
Write-Host "    OrangePi 已创建，ID: $devID，将参与绑定/远程测试`n" -ForegroundColor DarkGray

# ==================== OrangePi 远程接口 ====================
Write-Host "========== OrangePi 远程接口 ==========" -ForegroundColor Yellow
# 如需测试真实在线 Orangepi，可通过 RemoteOrangePiId 指定已有设备，默认为 2
$remoteTargetId = if ($RemoteOrangePiId -gt 0) { $RemoteOrangePiId } else { $devID }
Write-Host "    远程接口使用的 OrangePi ID: $remoteTargetId" -ForegroundColor DarkGray
Test-Endpoint "查询远程设备信息" "GET" "$BaseUrl/api/orangepi/remote/info?id=$remoteTargetId" $authHeaders
Test-Endpoint "远程健康检查" "GET" "$BaseUrl/api/orangepi/remote/health?id=$remoteTargetId" $authHeaders
$remoteBody = @{id = $remoteTargetId; ssh_remote_port = 22022; icctv_auth_service_remote_port = 33022 } | ConvertTo-Json
Test-Endpoint "远程端口更新" "POST" "$BaseUrl/api/orangepi/remote/ports" $authHeaders $remoteBody

# ==================== NVR接口 ====================
Write-Host "========== NVR管理 ==========" -ForegroundColor Yellow
$nvrBody = @{name = "测试NVR设备"; url = "192.168.1.100:8080"; building_id = $buildingId; admin_user = @{name = "admin"; password = "admin123" }; users = @(@{name = "u1"; password = "p1" }, @{name = "u2"; password = "p2" }); rtsp_urls = @(@{channel = 1; url = "rtsp://...1" }) } | ConvertTo-Json -Depth 10
$nvrResp = Test-Endpoint "创建NVR" "POST" "$BaseUrl/api/nvr" $authHeaders $nvrBody
$nvrID = if ($nvrResp) { $nvrResp.data.id }else { 0 }
Test-Endpoint "查询NVR" "GET" "$BaseUrl/api/nvr" $authHeaders
$updateNvrBody = @{name = "更新NVR设备" } | ConvertTo-Json
Test-Endpoint "更新NVR" "PUT" "$BaseUrl/api/nvr?id=$nvrID" $authHeaders $updateNvrBody
$delNvrBody = @{id = $nvrID } | ConvertTo-Json
Write-Host "    NVR 已创建，ID: $nvrID，将用于绑定测试`n" -ForegroundColor DarkGray

# ==================== Building-OrangePi Bind接口 ====================
Write-Host "========== Building-OrangePi 绑定 ==========" -ForegroundColor Yellow
$bindBody = @{building_id = $buildingId; orangepi_id = $devID } | ConvertTo-Json
Test-Endpoint "绑定设备到建筑" "POST" "$BaseUrl/api/bind/building-orangepi" $authHeaders $bindBody
$getBindUrl = "$BaseUrl/api/bind/building-orangepi/$buildingId"
Test-Endpoint "获取建筑OrangePi绑定" "GET" $getBindUrl $authHeaders
$unbindBody = @{orangepi_id = $devID } | ConvertTo-Json
Test-Endpoint "解绑OrangePi" "DELETE" "$BaseUrl/api/bind/building-orangepi" $authHeaders $unbindBody

# ==================== Building-NVR Bind接口 ====================
Write-Host "========== Building-NVR 绑定 ==========" -ForegroundColor Yellow
$bindNvrBody = @{building_id = $buildingId; nvr_id = $nvrID } | ConvertTo-Json
Test-Endpoint "绑定NVR到建筑" "POST" "$BaseUrl/api/bind/building-nvr" $authHeaders $bindNvrBody
$getBindNvrUrl = "$BaseUrl/api/bind/building-nvr/$buildingId"
Test-Endpoint "获取建筑NVR绑定" "GET" $getBindNvrUrl $authHeaders
$unbindNvrBody = @{nvr_id = $nvrID } | ConvertTo-Json
Test-Endpoint "解绑NVR" "DELETE" "$BaseUrl/api/bind/building-nvr" $authHeaders $unbindNvrBody

# ==================== 设备Info/网络配置 ====================
Write-Host "========== 设备汇总&公网配置 ==========" -ForegroundColor Yellow
Test-Endpoint "设备汇总信息" "GET" "$BaseUrl/api/device/info" $authHeaders
$pubCfgBody = @{external_ip = "203.0.113.77" } | ConvertTo-Json
Test-Endpoint "修改公网配置" "PUT" "$BaseUrl/api/publicnet/config" $authHeaders $pubCfgBody

# ==================== 清理测试数据 ====================
Write-Host "========== 清理测试数据 ==========" -ForegroundColor Yellow
if ($devID) { Test-Endpoint "删除设备" "DELETE" "$BaseUrl/api/device" $authHeaders $delDevBody }
if ($nvrID) { Test-Endpoint "删除NVR" "DELETE" "$BaseUrl/api/nvr" $authHeaders $delNvrBody }
if ($buildingId) { Test-Endpoint "删除建筑" "DELETE" "$BaseUrl/api/building" $authHeaders $delBuildBody }
if ($adminId) { Test-Endpoint "删除管理员" "DELETE" "$BaseUrl/api/admin" $authHeaders $delAdminBody }

