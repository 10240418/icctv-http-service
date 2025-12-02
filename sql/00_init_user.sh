#!/bin/bash
# =====================================================
# MySQL 用户初始化脚本
# 在 Docker 首次初始化时自动执行
# 支持通过环境变量配置用户名和密码
# =====================================================

set -e

echo "=== Initializing MySQL user for Docker network access ==="

# 使用环境变量，如果未设置则使用默认值
DB_USER="${MYSQL_USER:-icctv}"
DB_PASS="${MYSQL_PASSWORD:-1090119}"
DB_NAME="${MYSQL_DATABASE:-icctv_http_service}"

echo "Creating user: ${DB_USER}@%"

# 执行 SQL 命令创建用户
mysql -u root -p"${MYSQL_ROOT_PASSWORD}" <<-EOSQL
    -- 删除可能存在的旧用户（避免冲突）
    DROP USER IF EXISTS '${DB_USER}'@'localhost';
    DROP USER IF EXISTS '${DB_USER}'@'127.0.0.1';
    DROP USER IF EXISTS '${DB_USER}'@'%';

    -- 创建用户，允许从任意主机连接（Docker 网络需要）
    CREATE USER '${DB_USER}'@'%' IDENTIFIED BY '${DB_PASS}';

    -- 授予数据库完全权限
    GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'%';

    -- 刷新权限使其立即生效
    FLUSH PRIVILEGES;

    -- 验证用户创建成功
    SELECT user, host FROM mysql.user WHERE user='${DB_USER}';
EOSQL

echo "=== MySQL user ${DB_USER}@% created successfully ==="
