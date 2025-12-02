# icctv-http-service 部署总结

## 项目信息

- **项目名称**：icctv-http-service（Go 后端服务）
- **服务器**：Ubuntu 22.04.5 LTS (39.108.49.167)
- **部署目录**：/root/icctv/icctv-http-service
- **服务端口**：32001
- **MySQL 端口**：3307
- **数据库名称**：icctv_http_service
- **数据库用户**：icctv
- **数据库密码**：1090119

---

## 一、本地构建和准备

### 1. 构建 Docker 镜像

```powershell
# 在 Windows PowerShell 中执行
cd C:\Users\20216\Documents\GitHub\icctv-http-service

# 构建镜像（使用 --network=host 解决代理问题）
docker build -t icctv-http-service:latest --platform linux/amd64 --network=host .
```

**构建说明**：
- 使用 `docker.1ms.run/golang:1.24` 作为构建镜像
- 最终镜像基于 `scratch`（最小镜像）
- 编译为静态二进制文件（CGO_ENABLED=0）

### 2. 拉取 MySQL 镜像

```powershell
docker pull docker.1ms.run/mysql:8.0
```

### 3. 保存镜像为 tar 文件

```powershell
# 保存应用镜像（约 8.24 MB）
docker save icctv-http-service:latest -o icctv-http-service-latest.tar

# 保存 MySQL 镜像（约 224 MB）
docker save docker.1ms.run/mysql:8.0 -o mysql-8.0.tar
```

---

## 二、上传文件到服务器

### 1. 上传镜像文件

```powershell
# 上传应用镜像
scp icctv-http-service-latest.tar root@39.108.49.167:/root/icctv/icctv-http-service/

# 上传 MySQL 镜像
scp mysql-8.0.tar root@39.108.49.167:/root/icctv/icctv-http-service/
```

### 2. 上传配置文件

```powershell
# 上传 docker-compose.yml
scp docker-compose.yml root@39.108.49.167:/root/icctv/icctv-http-service/

# 上传 SQL 初始化脚本（如果存在）
scp sql/init_mysql_user.sql root@39.108.49.167:/root/icctv/icctv-http-service/sql/

# 上传权限修复脚本（如果存在）
scp sql/fix_user_permissions.sh root@39.108.49.167:/root/icctv/icctv-http-service/sql/
```

---

## 三、服务器部署命令

### 1. 加载镜像

```bash
cd /root/icctv/icctv-http-service

# 加载应用镜像
docker load -i icctv-http-service-latest.tar

# 加载 MySQL 镜像
docker load -i mysql-8.0.tar
```

### 2. 启动服务

```bash
# 启动所有服务
docker-compose up -d

# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f app
docker-compose logs mysql-init

# 查看特定容器日志
docker logs icctv-app
docker logs icctv-mysql
```

### 3. 停止服务

```bash
# 停止所有服务
docker-compose down

# 停止并删除数据卷（谨慎使用，会删除数据）
docker-compose down -v
```

### 4. 重启服务

```bash
# 重启所有服务
docker-compose restart

# 重启特定服务
docker-compose restart app
docker-compose restart mysql
```

---

## 四、遇到的问题和解决方案

### 问题 1：Docker 构建失败 - 网络代理问题

**错误信息**：
```
go: filippo.io/edwards25519@v1.1.0: Get "https://goproxy.cn/...": proxyconnect tcp: dial tcp 127.0.0.1:7890: connect: connection refused
```

**原因**：
- 宿主机代理设置（127.0.0.1:7890）被继承到容器
- 容器内无法访问宿主机的代理服务

**解决方案**：
使用 `--network=host` 参数构建，让容器使用宿主机网络

```powershell
docker build -t icctv-http-service:latest --platform linux/amd64 --network=host .
```

**Dockerfile 关键配置**：
- 禁用系统代理环境变量
- 使用阿里云 Go 代理：`GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`

---

### 问题 2：MySQL 用户权限错误

**错误信息**：
```
Error 1130: Host '172.19.0.3' is not allowed to connect to this MySQL server
```

**原因**：
- MySQL 容器通过 `MYSQL_USER` 环境变量创建的 `icctv` 用户默认只允许从 `localhost` 连接
- Docker 网络中，app 容器通过服务名 `mysql` 连接，实际 IP 是动态分配的（如 172.19.0.3）
- 需要允许用户从任何主机（`%`）连接

**状态**：
- app 容器一直重启，无法连接 MySQL
- 查看日志：`docker logs icctv-app` 显示连接被拒绝

**解决方案 A：手动修复（临时）**

```bash
# 使用环境变量方式执行 SQL（避免密码警告）
docker exec -e MYSQL_PWD=1090119 icctv-mysql mysql -uroot <<EOF
DROP USER IF EXISTS 'icctv'@'localhost';
DROP USER IF EXISTS 'icctv'@'%';
CREATE USER 'icctv'@'%' IDENTIFIED BY '1090119';
GRANT ALL PRIVILEGES ON icctv_http_service.* TO 'icctv'@'%';
FLUSH PRIVILEGES;
SELECT 'User permissions fixed successfully!' AS Status;
EOF

# 重启 app 容器
docker restart icctv-app
```

**解决方案 B：自动化方案（推荐）**

创建 `mysql-init` 服务，在 MySQL 健康检查通过后自动执行权限修复：

1. **创建权限修复脚本** `sql/fix_user_permissions.sh`：
   - 等待 MySQL 就绪
   - 自动修复用户权限，允许从 `%` 连接

2. **修改 docker-compose.yml**：
   - 添加 `mysql-init` 服务
   - `app` 服务依赖 `mysql-init` 完成

3. **工作流程**：
   - MySQL 启动 → 健康检查通过 → mysql-init 执行权限修复 → app 启动

**注意**：如果使用自动化方案，需要确保已上传 `sql/fix_user_permissions.sh` 文件

---

## 五、当前配置文件

### docker-compose.yml 关键配置

**MySQL 服务**：
- 镜像：`docker.1ms.run/mysql:8.0`
- 容器名：`icctv-mysql`
- 端口映射：`3307:3306`
- 数据卷：`icctv_mysql_data`
- 初始化脚本：`./sql/init_mysql_user.sql`（首次初始化）

**mysql-init 服务**（如果已配置）：
- 镜像：`docker.1ms.run/mysql:8.0`
- 容器名：`icctv-mysql-init`
- 重启策略：`no`（执行一次后退出）
- 依赖：等待 MySQL 健康检查通过
- 功能：自动执行权限修复脚本

**app 服务**：
- 镜像：`icctv-http-service:latest`
- 容器名：`icctv-app`
- 端口映射：`32001:8080`
- 依赖：MySQL 健康检查 + mysql-init 完成
- 环境变量：
  - `DB_DRIVER=mysql`
  - `DB_HOST=mysql`
  - `DB_PORT=3306`
  - `DB_USER=icctv`
  - `DB_PASS=1090119`
  - `DB_NAME=icctv_http_service`
  - `HTTP_ADDR=:8080`

### 关键文件

1. **Dockerfile**：
   - 多阶段构建（golang:1.24 → scratch）
   - 静态编译，无需运行时依赖

2. **sql/init_mysql_user.sql**（如果存在）：
   - 首次初始化脚本
   - 在 MySQL 首次启动时执行

3. **sql/fix_user_permissions.sh**（如果存在）：
   - 权限修复脚本
   - 每次启动时通过 mysql-init 服务执行

---

## 六、当前状态

**问题**：
- app 容器一直重启，无法连接 MySQL
- 错误：`Error 1130: Host '172.19.0.3' is not allowed to connect to this MySQL server`

**原因**：
- MySQL 用户权限未正确配置
- `icctv` 用户只允许从 `localhost` 连接，不允许从 Docker 网络连接

**需要**：
- 执行手动修复命令（方案 A），或
- 部署自动化方案（方案 B）

---

## 七、下一步操作建议

### 选项 1：立即修复（手动）

```bash
# 在服务器上执行
docker exec -e MYSQL_PWD=1090119 icctv-mysql mysql -uroot <<EOF
DROP USER IF EXISTS 'icctv'@'localhost';
DROP USER IF EXISTS 'icctv'@'%';
CREATE USER 'icctv'@'%' IDENTIFIED BY '1090119';
GRANT ALL PRIVILEGES ON icctv_http_service.* TO 'icctv'@'%';
FLUSH PRIVILEGES;
EOF

# 重启 app 容器
docker restart icctv-app

# 验证服务状态
docker-compose ps
docker logs icctv-app
```

### 选项 2：部署自动化方案（长期）

1. **确保已上传文件**：
   ```powershell
   scp docker-compose.yml root@39.108.49.167:/root/icctv/icctv-http-service/
   scp sql/fix_user_permissions.sh root@39.108.49.167:/root/icctv/icctv-http-service/sql/
   ```

2. **在服务器上执行**：
   ```bash
   cd /root/icctv/icctv-http-service
   
   # 停止服务
   docker-compose down
   
   # 重新启动（会自动执行权限修复）
   docker-compose up -d
   
   # 查看 mysql-init 日志确认
   docker-compose logs mysql-init
   
   # 查看 app 日志确认
   docker-compose logs app
   ```

---

## 八、验证服务

### 健康检查

```bash
# 检查服务状态
curl http://localhost:32001/health

# 或从外部访问
curl http://39.108.49.167:32001/health
```

**预期响应**：`ok`

### 测试 API

```bash
# 测试登录接口（默认管理员：admin/123456）
curl -X POST http://39.108.49.167:32001/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"123456"}'
```

---

## 九、常用维护命令

### 查看容器状态

```bash
docker-compose ps
docker ps | grep icctv
```

### 查看日志

```bash
# 实时查看所有日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f app
docker-compose logs -f mysql

# 查看最近 100 行日志
docker-compose logs --tail=100 app
```

### 进入容器

```bash
# 进入 MySQL 容器
docker exec -it icctv-mysql bash

# 进入 app 容器（scratch 镜像无 shell，无法进入）
```

### 数据库操作

```bash
# 连接 MySQL
docker exec -it icctv-mysql mysql -uicctv -p1090119 icctv_http_service

# 执行 SQL 文件
docker exec -i icctv-mysql mysql -uicctv -p1090119 icctv_http_service < /path/to/file.sql
```

### 备份和恢复

```bash
# 备份数据库
docker exec icctv-mysql mysqldump -uicctv -p1090119 icctv_http_service > backup.sql

# 恢复数据库
docker exec -i icctv-mysql mysql -uicctv -p1090119 icctv_http_service < backup.sql
```

---

## 十、注意事项

1. **数据持久化**：
   - MySQL 数据存储在 Docker 卷 `icctv_mysql_data` 中
   - 删除容器不会删除数据，但 `docker-compose down -v` 会删除数据卷

2. **端口占用**：
   - 确保 3307 和 32001 端口未被占用
   - 检查：`netstat -tuln | grep -E '3307|32001'`

3. **密码安全**：
   - 当前所有密码都是 `1090119`，生产环境请修改
   - 修改位置：`docker-compose.yml` 中的环境变量

4. **Go 代码自动初始化**：
   - Go 代码会自动创建表结构（AutoMigrate）
   - Go 代码会自动创建默认管理员账户（admin/123456）
   - 无需手动执行 SQL 脚本（除非需要特殊初始化）

5. **镜像代理**：
   - 使用 `docker.1ms.run` 作为镜像代理加速
   - 如果代理不可用，可改为官方镜像

---

## 十一、故障排查

### app 容器一直重启

1. 查看日志：`docker logs icctv-app`
2. 检查 MySQL 连接：确认 MySQL 容器正常运行
3. 检查用户权限：执行权限修复命令
4. 检查网络：确认容器在同一 Docker 网络中

### MySQL 连接失败

1. 检查 MySQL 容器状态：`docker-compose ps mysql`
2. 检查健康检查：`docker inspect icctv-mysql | grep Health`
3. 测试连接：`docker exec -it icctv-mysql mysql -uicctv -p1090119`
4. 检查用户权限：`SELECT user, host FROM mysql.user WHERE user='icctv';`

### 端口无法访问

1. 检查防火墙：`ufw status`
2. 检查端口监听：`netstat -tuln | grep 32001`
3. 检查容器端口映射：`docker port icctv-app`

---

## 十二、更新部署流程

当代码或配置更新后：

1. **本地构建新镜像**：
   ```powershell
   docker build -t icctv-http-service:latest --platform linux/amd64 --network=host .
   docker save icctv-http-service:latest -o icctv-http-service-latest.tar
   ```

2. **上传文件**：
   ```powershell
   scp icctv-http-service-latest.tar root@39.108.49.167:/root/icctv/icctv-http-service/
   scp docker-compose.yml root@39.108.49.167:/root/icctv/icctv-http-service/
   ```

3. **服务器更新**：
   ```bash
   cd /root/icctv/icctv-http-service
   docker load -i icctv-http-service-latest.tar
   docker-compose up -d --force-recreate app
   ```

---

## 附录：完整命令清单

### 本地操作（Windows PowerShell）

```powershell
# 构建镜像
docker build -t icctv-http-service:latest --platform linux/amd64 --network=host .

# 拉取 MySQL 镜像
docker pull docker.1ms.run/mysql:8.0

# 保存镜像
docker save icctv-http-service:latest -o icctv-http-service-latest.tar
docker save docker.1ms.run/mysql:8.0 -o mysql-8.0.tar

# 上传文件
scp icctv-http-service-latest.tar root@39.108.49.167:/root/icctv/icctv-http-service/
scp mysql-8.0.tar root@39.108.49.167:/root/icctv/icctv-http-service/
scp docker-compose.yml root@39.108.49.167:/root/icctv/icctv-http-service/
```

### 服务器操作（Linux Bash）

```bash
# 加载镜像
cd /root/icctv/icctv-http-service
docker load -i icctv-http-service-latest.tar
docker load -i mysql-8.0.tar

# 启动服务
docker-compose up -d

# 查看状态
docker-compose ps
docker-compose logs -f

# 停止服务
docker-compose down

# 重启服务
docker-compose restart
```
