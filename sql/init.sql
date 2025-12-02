-- =====================================================
-- OrangePi HTTP Service 数据库初始化脚本
-- =====================================================

-- 使用指定数据库
USE icctv_http_service;

-- =====================================================
-- 禁用外键检查（允许删除有外键关系的表）
-- =====================================================
SET FOREIGN_KEY_CHECKS = 0;

-- =====================================================
-- 删除表（顺序不重要，因为已禁用外键检查）
-- =====================================================
DROP TABLE IF EXISTS `public_net_configs`;
DROP TABLE IF EXISTS `orangepis`;
DROP TABLE IF EXISTS `buildings`;
DROP TABLE IF EXISTS `adminers`;

-- =====================================================
-- 启用外键检查
-- =====================================================
SET FOREIGN_KEY_CHECKS = 1;

-- =====================================================
-- 1. 管理员表 (adminers)
-- =====================================================
CREATE TABLE `adminers` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `username` VARCHAR(100) UNIQUE NOT NULL COMMENT '登录用户名',
  `password_hash` VARCHAR(255) NOT NULL COMMENT 'Bcrypt密码哈希',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员账户表';

-- =====================================================
-- 2. 建筑信息表 (buildings)
-- =====================================================
CREATE TABLE `buildings` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `base` VARCHAR(100) UNIQUE NOT NULL COMMENT '物理园区/楼栋唯一标识',
  `ismartid` VARCHAR(100) COMMENT 'ismart系统ID',
  `name` VARCHAR(255) NOT NULL COMMENT '楼栋名称',
  `remark` TEXT COMMENT '备注信息',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_base` (`base`),
  KEY `idx_ismartid` (`ismartid`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='建筑信息表';

-- =====================================================
-- 3. OrangePi设备表 (orangepis)
-- =====================================================
CREATE TABLE `orangepis` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `base` VARCHAR(100) NOT NULL COMMENT '关联楼栋base',
  `name` VARCHAR(255) NOT NULL COMMENT 'OrangePi设备名称',
  `icctv_auth_service_remote_port` INT NOT NULL COMMENT '远程认证服务端口',
  `ssh_remote_port` INT NOT NULL COMMENT 'SSH远程端口',
  `admin_ports` JSON COMMENT '可用管理端口列表,JSON格式',
  `mediamtx_paths` JSON COMMENT 'MediaMTX路径列表',
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_base` (`base`),
  KEY `idx_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_orangepis_buildings` FOREIGN KEY (`base`) REFERENCES `buildings` (`base`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='OrangePi设备表';

-- =====================================================
-- 4. 公网配置表 (public_net_configs)
-- =====================================================
CREATE TABLE `public_net_configs` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `external_ip` VARCHAR(50) NOT NULL COMMENT '外部IP地址',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='公网配置表';

-- =====================================================
-- 测试数据插入
-- =====================================================

-- 插入管理员账户
-- 密码 123456 的 bcrypt 哈希值
INSERT INTO `adminers` (`username`, `password_hash`, `created_at`, `updated_at`) VALUES
('admin', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36DvMvFm', NOW(), NOW()),
('test_user', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcg7b3XeKeUxWdeS86E36DvMvFm', NOW(), NOW());

-- 插入建筑信息
INSERT INTO `buildings` (`base`, `ismartid`, `name`, `remark`, `created_at`, `updated_at`) VALUES
('building_a', 'ismart_001', 'A栋', '主楼办公', NOW(), NOW()),
('building_b', 'ismart_002', 'B栋', '员工宿舍', NOW(), NOW()),
('building_c', 'ismart_003', 'C栋', '设备机房', NOW(), NOW());

-- 插入OrangePi设备
INSERT INTO `orangepis` (`base`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `admin_ports`, `mediamtx_paths`, `is_active`, `created_at`, `updated_at`) VALUES
('building_a', 'OrangePi-A-001', 30001, 20001, '[1, 2, 3]', JSON_ARRAY(JSON_OBJECT('channel', 1, 'url', '/cam/1')), TRUE, NOW(), NOW()),
('building_a', 'OrangePi-A-002', 30002, 20002, '[1, 2, 3, 4]', JSON_ARRAY(JSON_OBJECT('channel', 1, 'url', '/cam/1'), JSON_OBJECT('channel', 2, 'url', '/cam/2')), TRUE, NOW(), NOW()),
('building_b', 'OrangePi-B-001', 30003, 20003, '[1, 2]', JSON_ARRAY(), TRUE, NOW(), NOW()),
('building_c', 'OrangePi-C-001', 30004, 20004, '[1, 2, 3, 4, 5, 6]', JSON_ARRAY(JSON_OBJECT('channel', 8, 'url', '/cam/8')), FALSE, NOW(), NOW());

-- 插入公网配置
INSERT INTO `public_net_configs` (`external_ip`, `created_at`, `updated_at`) VALUES
('203.0.113.0', NOW(), NOW());

-- =====================================================
-- 数据验证查询
-- =====================================================

SELECT '=== 数据插入完成 ===' as '';

SELECT '=== 管理员列表 ===' as '';
SELECT `id`, `username`, `created_at`, `updated_at` FROM `adminers` WHERE `deleted_at` IS NULL;

SELECT '=== 建筑列表 ===' as '';
SELECT `id`, `base`, `ismartid`, `name`, `remark`, `created_at`, `updated_at` FROM `buildings` WHERE `deleted_at` IS NULL;

SELECT '=== 设备列表 ===' as '';
SELECT `id`, `base`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `admin_ports`, `is_active`, `created_at` FROM `orangepis` WHERE `deleted_at` IS NULL;

SELECT '=== 公网配置 ===' as '';
SELECT `id`, `external_ip`, `created_at`, `updated_at` FROM `public_net_configs` WHERE `deleted_at` IS NULL;
-- =====================================================
-- 5. 旧库结构清理脚本（可选）
--    作用：在老版本数据库中，删除 base / 旧 ismart 列，只保留 ismart_id
--    使用方式：确认数据已备份后，单独执行本段 SQL
-- =====================================================

-- 关闭外键检查，允许删除受外键约束的列
SET FOREIGN_KEY_CHECKS = 0;

-- 1) 删除 orangepis 表上所有与 base 相关的外键约束
-- 若不存在会报 “doesn't exist”，可忽略该错误
ALTER TABLE `orangepis` DROP FOREIGN KEY `fk_buildings_orange_pis`;
ALTER TABLE `orangepis` DROP FOREIGN KEY `fk_orangepis_buildings`;
ALTER TABLE `orangepis` DROP FOREIGN KEY `fk_orangepis_building`;

-- 2) 删除 orangepis / buildings 表中的 base 列
ALTER TABLE `orangepis` DROP COLUMN `base`;
ALTER TABLE `buildings` DROP COLUMN `base`;

-- 3) 删除 buildings 中历史遗留的 ismart 列，只保留 ismart_id
ALTER TABLE `buildings` DROP COLUMN `i_smart_id`;
ALTER TABLE `buildings` DROP COLUMN `ismartid`;

-- 恢复外键检查
SET FOREIGN_KEY_CHECKS = 1;

