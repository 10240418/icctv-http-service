# OrangePi 后台管理系统 - HTTP Service API 文档

## 📋 完整API接口列表

| 编号 | 接口 | 方法 | 简介/功能 | 权限 |
| --- | --- | --- | --- | --- |
| 1 | /api/auth/public | POST | 获取公开Token(24小时有效) | 无 |
| 2 | /api/auth/permanent | POST | 获取永久Token | 无 |
| 3 | /api/auth/login | POST | 管理员登录 | 无 |

| 4 | /api/admin | GET | 查询管理员列表 | 管理员 |
| 5 | /api/admin | POST | 创建管理员 | 管理员 |
| 6 | /api/admin | PUT | 更新管理员 | 管理员 |
| 7 | /api/admin | DELETE | 删除管理员 | 管理员 |

| 8 | /api/building | GET | 查询建筑列表 | 管理员 |
| 9 | /api/building | POST | 创建建筑信息 | 管理员 |
| 10 | /api/building | PUT | 更新建筑信息 | 管理员 |
| 11 | /api/building | DELETE | 删除建筑信息 | 管理员 |

| 12 | /api/device | GET | 查询OrangePi设备 | 管理员 |
| 13 | /api/device | POST | 创建OrangePi设备 | 管理员 |
| 14 | /api/device | PUT | 更新OrangePi设备 | 管理员 |
| 15 | /api/device | DELETE | 删除OrangePi设备 | 管理员 |

| 16 | /api/orangepi/remote/ports | POST | 远程端口更新 | 管理员 |
| 17 | /api/orangepi/remote/info | GET | 查询远程设备信息 | 管理员 |
| 18 | /api/orangepi/remote/health | GET | 远程健康检查 | 管理员 |

| 19 | /api/nvr | GET | 查询NVR列表 | 管理员 |
| 20 | /api/nvr | POST | 创建NVR信息 | 管理员 |
| 21 | /api/nvr | PUT | 更新NVR信息 | 管理员 |
| 22 | /api/nvr | DELETE | 删除NVR信息 | 管理员 |

| 23 | /api/bind/building-orangepi | POST | 绑定OrangePi到建筑 | 管理员 |
| 24 | /api/bind/building-orangepi | DELETE | 解绑OrangePi | 管理员 |
| 25 | /api/bind/building-orangepi/{building_id} | GET | 获取建筑关联的OrangePi | 管理员 |

| 26 | /api/bind/building-nvr | POST | 绑定NVR到建筑 | 管理员 |
| 27 | /api/bind/building-nvr | DELETE | 解绑NVR | 管理员 |
| 28 | /api/bind/building-nvr/{building_id} | GET | 获取建筑关联的NVR | 管理员 |

| 29 | /api/device/info | GET | 设备汇总信息 | 管理员 |
| 30 | /api/publicnet/config | GET | 获取公网配置 | 管理员 |
| 31 | /api/publicnet/config | PUT | 修改公网配置 | 管理员 |

---

## 详细接口参数/响应/脚本

### 1. /api/auth/public [POST]
- **简介**: 获取公开Token，有效期24小时
- **请求参数**
```json
{
  "building_id": "string", // 必填
  "channels": ["string"]   // 必填
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "token": "string" }
}
```
- **Powershell测试**
```powershell
$body = @{ismartid='0314100';is_staff=$false}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/auth/public" -Method POST -Body $body -ContentType "application/json"
```

---

### 2. /api/auth/permanent [POST]
- **简介**: 获取永久Token，永不过期
- **请求参数**
```json
{
  "ismartid": "string", // 必填，建筑ISmartID
  "is_staff": false     // 可选，是否员工(员工可访问所有频道)
}
```
- **响应参数**
```json
{
  "success": true,
  "data": {
    "orangepis": [
      {
        "orangepi_id": 1,
        "orangepi_name": "设备名称",
        "is_active": true,
        "token": "永久有效的token字符串",
        "urls": ["http://公网IP:端口/channel1?token=xxx"]
      }
    ]
  }
}
```
- **Powershell测试**
```powershell
$body = @{ismartid='0314100';is_staff=$false}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/auth/permanent" -Method POST -Body $body -ContentType "application/json"
```

---

### 3. /api/auth/login [POST]
- **请求参数**
```json
{
  "username": "string",
  "password": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "accessToken": "string", "expiresAt": "string" }
}
```
- **Powershell测试**
```powershell
$body = @{username='admin';password='123456'}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/auth/login" -Method POST -Body $body -ContentType "application/json"
```

---

### 4. /api/admin [GET]
- **请求参数**（示例。如果无查询参数可省略）
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": {...}
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/admin?id=1" -Headers $headers
```

---

### 5. /api/admin [POST]
- **请求参数**
```json
{
  "username": "string",
  "password": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "username": "string", "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{username='admin';password='123456'}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/admin" -Method POST -Headers $headers -Body $body
```

---

### 6. /api/admin [PUT]
- **请求参数**
```json
{
  "id": 1,
  "username": "string",
  "password": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "username": "string", "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1;username='admin';password='123456'}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/admin" -Method PUT -Headers $headers -Body $body
```

---

### 7. /api/admin [DELETE]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "deleted": true }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/admin" -Method DELETE -Headers $headers -Body $body
```

---

### 8. /api/building [GET]
- **请求参数**（示例。如果无查询参数可省略）
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": {...}
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/building?id=1" -Headers $headers
```

---

### 9. /api/building [POST]
- **请求参数**
```json
{
  "ismartid": "string",
  "name": "string",
  "remark": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "ismartid": "string", "name": "string", "remark": "string", "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{ismartid='xxx';name='xxx';remark='xxx'}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/building" -Method POST -Headers $headers -Body $body
```

---

### 10. /api/building [PUT]
- **请求参数**
```json
{
  "id": 1,
  "ismartid": "string",
  "name": "string",
  "remark": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "ismartid": "string", "name": "string", "remark": "string", "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1;ismartid='xxx';name='xxx';remark='xxx'}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/building?id=1" -Method PUT -Headers $headers -Body $body
```

---

### 11. /api/building [DELETE]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "deleted": true }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/building" -Method DELETE -Headers $headers -Body $body
```

---

### 12. /api/device [GET]
- **请求参数**（示例。如果无查询参数可省略）
```json
{
  "ismartid": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": [...]
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/device?ismartid=ismart_001" -Headers $headers
```

---

### 13. /api/device [POST]
- **请求参数**
```json
{
  "ismartid": "string",
  "name": "string",
  "icctv_auth_service_remote_port": 123,
  "ssh_remote_port": 456,
  "is_active": true
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "ismartid": "string", "name": "string", "icctv_auth_service_remote_port": 123, "ssh_remote_port": 456, "is_active": true, "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{ismartid='xxx';name='xxx';icctv_auth_service_remote_port=123;ssh_remote_port=456;is_active=$true}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/device" -Method POST -Headers $headers -Body $body
```

---

### 14. /api/device [PUT]
- **请求参数**
```json
{
  "id": 1,
  "ismartid": "string",
  "name": "string",
  "icctv_auth_service_remote_port": 123,
  "ssh_remote_port": 456,
  "is_active": true
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "ismartid": "string", "name": "string", "icctv_auth_service_remote_port": 123, "ssh_remote_port": 456, "is_active": true, "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1;ismartid='xxx';name='xxx';icctv_auth_service_remote_port=123;ssh_remote_port=456;is_active=$true}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/device?id=1" -Method PUT -Headers $headers -Body $body
```

---

### 15. /api/device [DELETE]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "deleted": true }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/device" -Method DELETE -Headers $headers -Body $body
```

---

### 16. /api/orangepi/remote/ports [POST]
- **请求参数**
```json
{
  "id": 1,
  "ssh_remote_port": 123,
  "icctv_auth_service_remote_port": 456
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "success": true, "message": "string", "restarted": true }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1;ssh_remote_port=123;icctv_auth_service_remote_port=456}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/orangepi/remote/ports" -Method POST -Headers $headers -Body $body
```

---

### 17. /api/orangepi/remote/info [GET]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "totalDevices": 10, "activeDevices": 8, "buildingBounded": 5, "lastSync": "string" }
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/orangepi/remote/info?id=1" -Headers $headers
```

---

### 18. /api/orangepi/remote/health [GET]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "status": "string", "message": "string" }
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/orangepi/remote/health?id=1" -Headers $headers
```

---

### 19. /api/nvr [GET]
- **请求参数**（示例。如果无查询参数可省略）
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": [...]
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/nvr?id=1" -Headers $headers
```

---

### 20. /api/nvr [POST]
- **请求参数**
```json
{
  "name": "string",
  "url": "string",
  "building_id": 1,
  "admin_user": { "name": "string", "password": "string" },
  "users": [ { "name": "string", "password": "string" } ],
  "rtsp_urls": [ { "channel": 1, "url": "string" } ]
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "name": "string", "url": "string", "building_id": 1, "admin_user": { "name": "string", "password": "string" }, "users": [ { "name": "string", "password": "string" } ], "rtsp_urls": [ { "channel": 1, "url": "string" } ], "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{name='xxx';url='xxx';building_id=1;admin_user=@{"name"="admin";"password"="123456"};users=@(@{"name"="operator1";"password"="pass123"});rtsp_urls=@(@{"channel"=1;"url"="rtsp://192.168.1.100:554/stream1"})}|ConvertTo-Json -Depth 10
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/nvr" -Method POST -Headers $headers -Body $body
```

---

### 21. /api/nvr [PUT]
- **请求参数**
```json
{
  "id": 1,
  "name": "string",
  "url": "string",
  "building_id": 1,
  "admin_user": { "name": "string", "password": "string" },
  "users": [ { "name": "string", "password": "string" } ],
  "rtsp_urls": [ { "channel": 1, "url": "string" } ]
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "name": "string", "url": "string", "building_id": 1, "admin_user": { "name": "string", "password": "string" }, "users": [ { "name": "string", "password": "string" } ], "rtsp_urls": [ { "channel": 1, "url": "string" } ], "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1;name='xxx';url='xxx';building_id=1;admin_user=@{"name"="new_admin";"password"="newpass123"};users=@(@{"name"="operator3";"password"="pass789"});rtsp_urls=@(@{"channel"=1;"url"="rtsp://192.168.1.200:554/updated_stream1"})}|ConvertTo-Json -Depth 10
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/nvr?id=1" -Method PUT -Headers $headers -Body $body
```

---

### 22. /api/nvr [DELETE]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "deleted": true }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{id=1}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/nvr" -Method DELETE -Headers $headers -Body $body
```

---

### 23. /api/bind/building-orangepi [POST]
```json
{
  "building_id": $buildingId, // 必填
  "orangepi_id": $devID // 必填
}
```
- **响应**
```json
{"success":true, "data":{"bound":true}}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{building_id=$buildingId; orangepi_id=$devID}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/bind/building-orangepi" -Method POST -Headers $headers -Body $body
```

### 24. /api/bind/building-orangepi [DELETE]
```json
{
  "orangepi_id": $devID // 必填
}
```
- **响应**
```json
{"success":true,"data":{"unbound":true}}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token";"Content-Type"="application/json"}
$body=@{orangepi_id=$devID}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/bind/building-orangepi" -Method DELETE -Headers $headers -Body $body
```

### 25. /api/bind/building-orangepi/{building_id} [GET]
- **无Body参数**
- **响应**
```json
{"success":true,"data":[{...}]}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/bind/building-orangepi/$buildingId" -Headers $headers
```

### 26. /api/bind/building-nvr [POST]
```json
{
  "building_id": $buildingId, // 必填
  "nvr_id": $nvrID // 必填
}
```
- **响应**
```json
{"success":true, "data":{"bound":true}}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{building_id=$buildingId; nvr_id=$nvrID}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/bind/building-nvr" -Method POST -Headers $headers -Body $body
```

### 27. /api/bind/building-nvr [DELETE]
```json
{
  "nvr_id": $nvrID // 必填
}
```
- **响应**
```json
{"success":true,"data":{"unbound":true}}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token";"Content-Type"="application/json"}
$body=@{nvr_id=$nvrID}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/bind/building-nvr" -Method DELETE -Headers $headers -Body $body
```

### 28. /api/bind/building-nvr/{building_id} [GET]
- **无Body参数**
- **响应**
```json
{"success":true,"data":[{...}]}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/bind/building-nvr/$buildingId" -Headers $headers
```

### 29. /api/device/info [GET]
- **请求参数**
```json
{
  "id": 1
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "totalDevices": 10, "activeDevices": 8, "buildingBounded": 5, "lastSync": "string" }
}
```
- **Powershell测试**
```powershell
$headers=@{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/device/info?id=1" -Headers $headers
```

---

### 30. /api/publicnet/config [GET]
- **请求参数**
```json
{
  // 无请求体
}
```
- **响应参数**
```json
{
  "success": true, // 请求是否成功
  "data": {
    "id": 1, // 配置ID
    "external_ip": "203.0.113.100", // 当前公网IP
    "createdAt": "2025-11-24T10:00:00Z", // 创建时间
    "updatedAt": "2025-11-24T10:00:00Z"  // 更新时间
  }
}
```
- **Powershell测试**
```powershell
# 本地使用 127.0.0.1
$headers = @{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/publicnet/config" -Method GET -Headers $headers

# 香橙派示例 192.168.10.10
$headers = @{"Authorization"="Bearer $token"}
Invoke-RestMethod -Uri "http://192.168.10.10:8080/api/publicnet/config" -Method GET -Headers $headers
```

---

### 31. /api/publicnet/config [PUT]
- **请求参数**
```json
{
  "external_ip": "string"
}
```
- **响应参数**
```json
{
  "success": true,
  "data": { "id": 1, "external_ip": "string", "createdAt": "string", "updatedAt": "string" }
}
```
- **Powershell测试**
```powershell
$headers = @{"Authorization"="Bearer $token"; "Content-Type"="application/json"}
$body = @{external_ip='203.0.113.100'}|ConvertTo-Json
Invoke-RestMethod -Uri "http://127.0.0.1:8080/api/publicnet/config" -Method PUT -Headers $headers -Body $body
```

---

### 0. 通用字段 ModelFields

所有业务模型统一继承 `ModelFields`，用于存储主键与时间戳：

```go
type ModelFields struct {
    ID        int64          `gorm:"primary_key" json:"id"`           // 主键ID
    CreatedAt time.Time      `json:"createdAt"`                       // 创建时间
    UpdatedAt time.Time      `json:"updatedAt"`                       // 更新时间
    DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`// 软删除时间
}
```

后续的 `Adminer`、`OrangePi`、`Building` 均通过匿名字段方式引入：

```go
type ExampleModel struct {
    ModelFields
    // ...业务字段...
}
```

### 1. Adminer(管理员) 模型

```go
type Adminer struct {
    ModelFields
    Username       string    `json:"username"`         // 登录用户名(唯一)
    PasswordHash   string    `json:"password_hash"`    // Bcrypt 哈希
}
```

### 2. OrangePi (设备) 模型

```go
type OrangePi struct {
    ModelFields

    Base                        string `json:"base"`          // 关联楼栋 base
    Name                        string `json:"name"`          // Orangepi 名称
    ICCTVAuthServiceRemotePort  int    `json:"icctv_auth_service_remote_port"` // 远程认证服务
    SSHRemotePort               int    `json:"ssh_remote_port"` // SSH 远程端口
    AdminPorts                  []int  `json:"admin_ports"`   // 可用管理端口列表(1~6)
    IsActive                    bool   `json:"is_active"`     // 是否在用
}
```

### 3. Building (建筑) 模型

```go
type Building struct {
    ModelFields

    Base     string `json:"base"`     // 物理园区/楼栋唯一标识
    ISmartID string `json:"ismartid"` // ismart 系统ID
    Name     string `json:"name"`     // 楼栋名称
    Remark   string `json:"remark"`   // 备注信息
    
    // 关联关系 (一对多)
    OrangePis []OrangePi `json:"orangepis,omitempty"` // 关联的OrangePi设备列表
    NVRs      []NVR      `json:"nvrs,omitempty"`      // 关联的NVR设备列表
}
```

### 4. PublicNetConfig (公网配置) 模型

```go
type PublicNetConfig struct {
    IP string `json:"external_ip"`  // 外部IP地址
}
```

### 5. NVR (网络硬盘录像机) 模型

#### 用户认证信息子模型

```go
// AdminUser 管理员账户信息
type AdminUser struct {
    Name     string `json:"name"`     // 管理员用户名
    Password string `json:"password"` // 管理员密码
}

// User 普通用户账户信息
type User struct {
    Name     string `json:"name"`     // 用户名
    Password string `json:"password"` // 密码
}

// ChannelURL RTSP频道地址
type ChannelURL struct {
    Channel int    `json:"channel"` // 通道号
    URL     string `json:"url"`     // RTSP 地址
}
```

#### NVR 完整模型

```go
type NVR struct {
    ModelFields
    
    Name       string       `json:"name"`        // NVR 名称
    URL        string       `json:"url"`         // NVR 访问地址 (IP:Port)
    BuildingID int64        `json:"building_id"` // 关联建筑ID (外键)
    AdminUser  AdminUser    `json:"admin_user"`  // 管理员账户
    Users      []User       `json:"users"`       // 普通用户列表
    RTSPUrls   []ChannelURL `json:"rtsp_urls"`   // RTSP 地址列表
    
    // 关联关系 (多对一)
    Building   Building     `json:"building,omitempty"` // 所属建筑
}
```

**字段说明:**
- `name`: NVR 设备的显示名称
- `url`: NVR 的管理访问地址，格式为 `IP:Port` (如: `192.168.1.100:80`)
- `building_id`: 关联的建筑ID
- `admin_user`: 管理员账户信息
- `users`: 普通用户列表
- `rtsp_urls`: RTSP流地址列表，每个包含通道号和对应的RTSP URL

**完整数据示例:**

```json
{
    "id": 1,
    "name": "A楼监控系统",
    "url": "192.168.1.100:8080",
    "building_id": 5,
    "admin_user": {
        "name": "admin",
        "password": "admin123"
    },
    "users": [
        {
            "name": "operator1",
            "password": "pass123"
        },
        {
            "name": "operator2",
            "password": "pass456"
        }
    ],
    "rtsp_urls": [
        {
            "channel": 1,
            "url": "rtsp://192.168.1.100:554/stream1"
        },
        {
            "channel": 2,
            "url": "rtsp://192.168.1.100:554/stream2"
        },
        {
            "channel": 3,
            "url": "rtsp://192.168.1.100:554/stream3"
        }
    ],
    "building": {
        "id": 5,
        "name": "办公楼A栋"
    },
    "createdAt": "2025-11-24T10:00:00Z",
    "updatedAt": "2025-11-24T10:00:00Z"
}
```

