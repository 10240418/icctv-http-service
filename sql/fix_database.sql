-- =====================================================
-- 数据库修复脚本
-- 解决 init.sql 与 Go 模型不匹配的问题
-- =====================================================

USE icctv_http_service;

-- =====================================================
-- 禁用外键检查
-- =====================================================
SET FOREIGN_KEY_CHECKS = 0;

-- =====================================================
-- 清理所有旧表
-- =====================================================
DROP TABLE IF EXISTS `nvrs`;
DROP TABLE IF EXISTS `public_net_configs`;
DROP TABLE IF EXISTS `orangepis`;
DROP TABLE IF EXISTS `buildings`;
DROP TABLE IF EXISTS `adminers`;

-- =====================================================
-- 启用外键检查
-- =====================================================
SET FOREIGN_KEY_CHECKS = 1;

-- =====================================================
-- 重新创建表（与 Go 模型完全一致）
-- =====================================================

-- 1. 管理员表
CREATE TABLE `adminers` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `username` VARCHAR(100) UNIQUE NOT NULL COMMENT '登录用户名',
  `password_hash` VARCHAR(255) NOT NULL COMMENT 'Bcrypt密码哈希',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_adminers_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='管理员账户表';

-- 2. 建筑信息表（使用 ismart_id 作为关联键）
CREATE TABLE `buildings` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `ismart_id` VARCHAR(100) NOT NULL UNIQUE COMMENT 'ismart系统ID，作为外键关联',
  `name` VARCHAR(255) NOT NULL COMMENT '楼栋名称',
  `remark` TEXT COMMENT '备注信息',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_buildings_ismart_id` (`ismart_id`),
  KEY `idx_buildings_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='建筑信息表';

-- 3. OrangePi设备表（通过 ismart_id 关联建筑）
CREATE TABLE `orangepis` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `ismart_id` VARCHAR(100) NOT NULL COMMENT '关联楼栋的ismart_id',
  `name` VARCHAR(255) NOT NULL COMMENT 'OrangePi设备名称',
  `icctv_auth_service_remote_port` INT NOT NULL COMMENT '远程认证服务端口',
  `ssh_remote_port` INT NOT NULL COMMENT 'SSH远程端口',
  `mediamtx_paths` JSON NULL COMMENT 'MediaMTX路径配置(JSON)',
  `is_active` BOOLEAN DEFAULT TRUE COMMENT '是否激活',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_orangepis_ismart_id` (`ismart_id`),
  KEY `idx_orangepis_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_orangepis_buildings` 
    FOREIGN KEY (`ismart_id`) 
    REFERENCES `buildings` (`ismart_id`)
    ON UPDATE CASCADE 
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='OrangePi设备表';

-- 4. NVR设备表（通过 building_id 关联建筑）
CREATE TABLE `nvrs` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `name` VARCHAR(255) NOT NULL COMMENT 'NVR名称',
  `url` VARCHAR(255) NOT NULL COMMENT 'NVR访问地址',
  `building_id` BIGINT NOT NULL COMMENT '关联建筑ID',
  `admin_user` JSON COMMENT '管理员账户信息',
  `users` JSON COMMENT '普通用户列表',
  `rtsp_urls` JSON COMMENT 'RTSP地址列表',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_nvrs_building_id` (`building_id`),
  KEY `idx_nvrs_deleted_at` (`deleted_at`),
  CONSTRAINT `fk_nvrs_buildings` 
    FOREIGN KEY (`building_id`) 
    REFERENCES `buildings` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='NVR设备表';

-- 5. 公网配置表
CREATE TABLE `public_net_configs` (
  `id` BIGINT PRIMARY KEY AUTO_INCREMENT,
  `external_ip` VARCHAR(50) NOT NULL COMMENT '外部IP地址',
  `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  `deleted_at` TIMESTAMP NULL COMMENT '软删除时间',
  KEY `idx_public_net_configs_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='公网配置表';

-- =====================================================
-- 插入测试数据
-- =====================================================

-- 插入管理员账户（密码: 123456）
-- 使用 bcrypt 在线生成器生成的哈希值
INSERT INTO `adminers` (`username`, `password_hash`, `created_at`, `updated_at`) VALUES
('admin', '$2b$12$c69LQAVpVP0YCFNVHHuitOySbNQNNqn2Ku9rrRYxxfH2zUhIjMnv2', NOW(), NOW()),
('test_user', '$2b$12$c69LQAVpVP0YCFNVHHuitOySbNQNNqn2Ku9rrRYxxfH2zUhIjMnv2', NOW(), NOW());

-- 插入建筑信息
INSERT INTO `buildings` (`ismart_id`, `name`, `remark`, `created_at`, `updated_at`) VALUES
('ismart_001', 'A栋主楼', '主要办公区域', NOW(), NOW()),
('ismart_002', 'B栋副楼', '研发中心', NOW(), NOW()),
('ismart_003', 'C栋宿舍', '员工宿舍楼', NOW(), NOW());

-- 插入OrangePi设备
INSERT INTO `orangepis` (`ismart_id`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `mediamtx_paths`, `is_active`, `created_at`, `updated_at`) VALUES
('ismart_001', 'OrangePi-A-001', 30001, 20001, JSON_ARRAY(JSON_OBJECT('channel', 1, 'url', '/cam/1')), TRUE, NOW(), NOW()),
('ismart_001', 'OrangePi-A-002', 30002, 20002, JSON_ARRAY(JSON_OBJECT('channel', 1, 'url', '/cam/1'), JSON_OBJECT('channel', 2, 'url', '/cam/2')), TRUE, NOW(), NOW()),
('ismart_002', 'OrangePi-B-001', 30003, 20003, JSON_ARRAY(), TRUE, NOW(), NOW()),
('ismart_003', 'OrangePi-C-001', 30004, 20004, JSON_ARRAY(JSON_OBJECT('channel', 8, 'url', '/cam/8')), FALSE, NOW(), NOW());

-- 插入公网配置
INSERT INTO `public_net_configs` (`external_ip`, `created_at`, `updated_at`) VALUES
('203.0.113.100', NOW(), NOW());

-- =====================================================
-- 验证数据
-- =====================================================

SELECT '=== 管理员列表 ===' as '';
SELECT `id`, `username`, LEFT(`password_hash`, 20) as `password_hash`, `created_at` FROM `adminers` WHERE `deleted_at` IS NULL;

SELECT '=== 建筑列表 ===' as '';
SELECT `id`, `ismart_id`, `name`, `remark` FROM `buildings` WHERE `deleted_at` IS NULL;

SELECT '=== 设备列表 ===' as '';
SELECT `id`, `ismart_id`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `is_active` FROM `orangepis` WHERE `deleted_at` IS NULL;

SELECT '=== 公网配置 ===' as '';
SELECT `id`, `external_ip` FROM `public_net_configs` WHERE `deleted_at` IS NULL;

SELECT '=== 数据库修复完成 ===' as '';
