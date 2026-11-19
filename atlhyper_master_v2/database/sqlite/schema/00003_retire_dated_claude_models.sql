-- 内置模型目录里带日期后缀的旧 Claude 模型 ID 换成不带日期的新目录（新目录由启动时的种子补齐）。
-- 只删目录条目：已经选了旧模型的提供商记录不受影响。

-- +goose Up
DELETE FROM ai_provider_models
WHERE provider = 'anthropic'
  AND model IN (
    'claude-sonnet-4-20250514',
    'claude-opus-4-5-20251101',
    'claude-3-5-sonnet-20241022',
    'claude-3-5-haiku-20241022',
    'claude-3-opus-20240229',
    'claude-3-haiku-20240307'
  );

-- +goose Down
-- 旧目录条目不再恢复。
