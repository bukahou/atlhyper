-- SLO 目标不再按 entrypoint 级别记录：清掉旧数据（此前每次启动都执行一遍，现在只执行一次）。

-- +goose Up
DELETE FROM slo_targets WHERE host LIKE '%@entrypoint%';

-- +goose Down
-- 删除的旧数据不再恢复。
