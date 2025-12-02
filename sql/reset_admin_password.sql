-- 重置管理员密码为 admin123
UPDATE adminers 
SET password_hash = '$2a$10$ZLxrRyRj2hFdFL1HhSE4oetoK5D.KdTnZ0UHCfqK3nFhcKhHIeLTu' 
WHERE username = 'admin';

-- 验证更新
SELECT id, username, LEFT(password_hash, 20) as hash_preview 
FROM adminers 
WHERE username = 'admin';

