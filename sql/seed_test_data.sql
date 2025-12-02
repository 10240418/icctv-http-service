-- =====================================================
-- 测试数据种子脚本 - 为 Building, OrangePi, NVR 插入30条数据
-- =====================================================

USE icctv_http_service;

-- =====================================================
-- 1. 插入30条建筑数据
-- =====================================================
INSERT INTO `buildings` (`ismart_id`, `name`, `remark`, `created_at`, `updated_at`) VALUES
('ismart_test_001', '测试楼栋-01', '测试用建筑01', NOW(), NOW()),
('ismart_test_002', '测试楼栋-02', '测试用建筑02', NOW(), NOW()),
('ismart_test_003', '测试楼栋-03', '测试用建筑03', NOW(), NOW()),
('ismart_test_004', '测试楼栋-04', '测试用建筑04', NOW(), NOW()),
('ismart_test_005', '测试楼栋-05', '测试用建筑05', NOW(), NOW()),
('ismart_test_006', '测试楼栋-06', '测试用建筑06', NOW(), NOW()),
('ismart_test_007', '测试楼栋-07', '测试用建筑07', NOW(), NOW()),
('ismart_test_008', '测试楼栋-08', '测试用建筑08', NOW(), NOW()),
('ismart_test_009', '测试楼栋-09', '测试用建筑09', NOW(), NOW()),
('ismart_test_010', '测试楼栋-10', '测试用建筑10', NOW(), NOW()),
('ismart_test_011', '测试楼栋-11', '测试用建筑11', NOW(), NOW()),
('ismart_test_012', '测试楼栋-12', '测试用建筑12', NOW(), NOW()),
('ismart_test_013', '测试楼栋-13', '测试用建筑13', NOW(), NOW()),
('ismart_test_014', '测试楼栋-14', '测试用建筑14', NOW(), NOW()),
('ismart_test_015', '测试楼栋-15', '测试用建筑15', NOW(), NOW()),
('ismart_test_016', '测试楼栋-16', '测试用建筑16', NOW(), NOW()),
('ismart_test_017', '测试楼栋-17', '测试用建筑17', NOW(), NOW()),
('ismart_test_018', '测试楼栋-18', '测试用建筑18', NOW(), NOW()),
('ismart_test_019', '测试楼栋-19', '测试用建筑19', NOW(), NOW()),
('ismart_test_020', '测试楼栋-20', '测试用建筑20', NOW(), NOW()),
('ismart_test_021', '测试楼栋-21', '测试用建筑21', NOW(), NOW()),
('ismart_test_022', '测试楼栋-22', '测试用建筑22', NOW(), NOW()),
('ismart_test_023', '测试楼栋-23', '测试用建筑23', NOW(), NOW()),
('ismart_test_024', '测试楼栋-24', '测试用建筑24', NOW(), NOW()),
('ismart_test_025', '测试楼栋-25', '测试用建筑25', NOW(), NOW()),
('ismart_test_026', '测试楼栋-26', '测试用建筑26', NOW(), NOW()),
('ismart_test_027', '测试楼栋-27', '测试用建筑27', NOW(), NOW()),
('ismart_test_028', '测试楼栋-28', '测试用建筑28', NOW(), NOW()),
('ismart_test_029', '测试楼栋-29', '测试用建筑29', NOW(), NOW()),
('ismart_test_030', '测试楼栋-30', '测试用建筑30', NOW(), NOW());

-- 创建"未绑定"占位楼栋（用于OrangePi未绑定状态的测试）
INSERT INTO `buildings` (`ismart_id`, `name`, `remark`, `created_at`, `updated_at`) VALUES
('unbound_temp', '临时未绑定占位楼栋', '用于测试unbind功能的临时楼栋', NOW(), NOW());

-- =====================================================
-- 2. 插入30条OrangePi数据 (前5个绑定到building 1-5, 其余25个绑定到临时楼栋)
-- =====================================================
-- 前5个OrangePi绑定到building 1-5 (使用ismart_id关联)
INSERT INTO `orangepis` (`ismart_id`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `is_active`, `created_at`, `updated_at`) VALUES
('ismart_test_001', 'OrangePi-Test-001', 40001, 22001, TRUE, NOW(), NOW()),
('ismart_test_002', 'OrangePi-Test-002', 40002, 22002, TRUE, NOW(), NOW()),
('ismart_test_003', 'OrangePi-Test-003', 40003, 22003, TRUE, NOW(), NOW()),
('ismart_test_004', 'OrangePi-Test-004', 40004, 22004, TRUE, NOW(), NOW()),
('ismart_test_005', 'OrangePi-Test-005', 40005, 22005, TRUE, NOW(), NOW());

-- 其余25个OrangePi绑定到临时楼栋 (可用于测试bind功能)
INSERT INTO `orangepis` (`ismart_id`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `is_active`, `created_at`, `updated_at`) VALUES
('unbound_temp', 'OrangePi-Test-006', 40006, 22006, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-007', 40007, 22007, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-008', 40008, 22008, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-009', 40009, 22009, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-010', 40010, 22010, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-011', 40011, 22011, FALSE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-012', 40012, 22012, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-013', 40013, 22013, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-014', 40014, 22014, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-015', 40015, 22015, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-016', 40016, 22016, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-017', 40017, 22017, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-018', 40018, 22018, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-019', 40019, 22019, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-020', 40020, 22020, FALSE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-021', 40021, 22021, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-022', 40022, 22022, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-023', 40023, 22023, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-024', 40024, 22024, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-025', 40025, 22025, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-026', 40026, 22026, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-027', 40027, 22027, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-028', 40028, 22028, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-029', 40029, 22029, TRUE, NOW(), NOW()),
('unbound_temp', 'OrangePi-Test-030', 40030, 22030, TRUE, NOW(), NOW());

-- =====================================================
-- 3. 插入30条NVR数据 (每个关联一个建筑)
-- =====================================================
-- 注意: 需要先获取building_id，所以使用子查询
INSERT INTO `nvrs` (`name`, `url`, `building_id`, `admin_user`, `users`, `rtsp_urls`, `created_at`, `updated_at`)
SELECT 
    CONCAT('NVR-Test-', LPAD(ROW_NUMBER() OVER (ORDER BY id), 3, '0')) AS name,
    CONCAT('192.168.1.', 100 + ROW_NUMBER() OVER (ORDER BY id), ':8000') AS url,
    id AS building_id,
    JSON_OBJECT('name', 'admin', 'password', 'admin123') AS admin_user,
    JSON_ARRAY(
        JSON_OBJECT('name', 'user1', 'password', 'user123'),
        JSON_OBJECT('name', 'operator', 'password', 'oper456')
    ) AS users,
    JSON_ARRAY(
        JSON_OBJECT('channel', 1, 'url', CONCAT('rtsp://192.168.1.', 100 + ROW_NUMBER() OVER (ORDER BY id), ':554/stream1')),
        JSON_OBJECT('channel', 2, 'url', CONCAT('rtsp://192.168.1.', 100 + ROW_NUMBER() OVER (ORDER BY id), ':554/stream2'))
    ) AS rtsp_urls,
    NOW() AS created_at,
    NOW() AS updated_at
FROM `buildings`
WHERE `ismart_id` LIKE 'ismart_test_%'
LIMIT 30;

-- =====================================================
-- 数据验证查询
-- =====================================================

SELECT '========================================' AS '';
SELECT '=== 测试数据插入完成 ===' AS '';
SELECT '========================================' AS '';

SELECT '' AS '';
SELECT '=== 建筑数据统计 ===' AS '';
SELECT COUNT(*) AS '总建筑数', 
       COUNT(CASE WHEN ismart_id LIKE 'ismart_test_%' THEN 1 END) AS '测试建筑数'
FROM `buildings` 
WHERE `deleted_at` IS NULL;

SELECT '' AS '';
SELECT '=== OrangePi数据统计 ===' AS '';
SELECT COUNT(*) AS '总OrangePi数', 
       COUNT(CASE WHEN name LIKE 'OrangePi-Test-%' THEN 1 END) AS '测试OrangePi数',
       COUNT(CASE WHEN is_active = TRUE THEN 1 END) AS '激活状态数'
FROM `orangepis` 
WHERE `deleted_at` IS NULL;

SELECT '' AS '';
SELECT '=== NVR数据统计 ===' AS '';
SELECT COUNT(*) AS '总NVR数', 
       COUNT(CASE WHEN name LIKE 'NVR-Test-%' THEN 1 END) AS '测试NVR数'
FROM `nvrs` 
WHERE `deleted_at` IS NULL;

SELECT '' AS '';
SELECT '=== 前5条建筑数据 ===' AS '';
SELECT `id`, `ismart_id`, `name`, `remark` 
FROM `buildings` 
WHERE `ismart_id` LIKE 'ismart_test_%' AND `deleted_at` IS NULL
LIMIT 5;

SELECT '' AS '';
SELECT '=== 前5条OrangePi数据 ===' AS '';
SELECT `id`, `ismart_id`, `name`, `icctv_auth_service_remote_port`, `ssh_remote_port`, `is_active`
FROM `orangepis` 
WHERE `name` LIKE 'OrangePi-Test-%' AND `deleted_at` IS NULL
LIMIT 5;

SELECT '' AS '';
SELECT '=== 前5条NVR数据 ===' AS '';
SELECT `id`, `name`, `url`, `building_id`
FROM `nvrs` 
WHERE `name` LIKE 'NVR-Test-%' AND `deleted_at` IS NULL
LIMIT 5;

SELECT '' AS '';
SELECT '========================================' AS '';
SELECT '=== 数据验证完成 ===' AS '';
SELECT '========================================' AS '';
