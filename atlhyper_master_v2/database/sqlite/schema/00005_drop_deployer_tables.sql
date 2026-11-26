-- +goose Up
-- 部署器已删除，这两张表不再有任何读写方（索引随表一起删除）
DROP TABLE IF EXISTS deploy_history;
DROP TABLE IF EXISTS deploy_config;

-- +goose Down
-- 删除的部署配置与部署历史不再恢复。
