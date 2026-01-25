"use client";

import { useEffect, useState, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { PageHeader, LoadingSpinner } from "@/components/common";
import { toast } from "@/components/common/Toast";
import { useAuthStore } from "@/store/authStore";
import {
  AlertTriangle,
  Bot,
  Loader2,
  AlertCircle,
  Eye,
  EyeOff,
  Check,
  RefreshCw,
} from "lucide-react";

import {
  getAIConfig,
  updateAIConfig,
  testAIConnection,
  mockAIConfig,
  type AIConfigResponse,
  type AIConfigUpdateRequest,
} from "@/api/settings";

export default function AISettingsPage() {
  const { t } = useI18n();
  const { user, isAuthenticated } = useAuthStore();

  // 权限判断
  const isGuest = !isAuthenticated;
  const isAdmin = user?.role === 3;

  // 状态
  const [loading, setLoading] = useState(true);
  const [config, setConfig] = useState<AIConfigResponse | null>(null);

  // 表单状态
  const [localEnabled, setLocalEnabled] = useState(false);
  const [provider, setProvider] = useState("gemini");
  const [apiKey, setApiKey] = useState("");
  const [model, setModel] = useState("");
  const [toolTimeout, setToolTimeout] = useState(30);

  // UI 状态
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [hasChanges, setHasChanges] = useState(false);
  const [showApiKey, setShowApiKey] = useState(false);

  // 加载数据
  useEffect(() => {
    if (isGuest) {
      setConfig(mockAIConfig);
      setLocalEnabled(mockAIConfig.enabled);
      setProvider(mockAIConfig.provider);
      setModel(mockAIConfig.model);
      setToolTimeout(mockAIConfig.tool_timeout);
      setLoading(false);
      return;
    }

    getAIConfig()
      .then((res) => {
        setConfig(res.data);
        setLocalEnabled(res.data.enabled);
        setProvider(res.data.provider);
        setModel(res.data.model);
        setToolTimeout(res.data.tool_timeout);
      })
      .catch((err) => {
        console.error("Failed to load AI config:", err);
        toast.error("加载 AI 配置失败");
      })
      .finally(() => {
        setLoading(false);
      });
  }, [isGuest]);

  // 检测变化
  useEffect(() => {
    if (!config) return;
    const changed =
      localEnabled !== config.enabled ||
      provider !== config.provider ||
      apiKey !== "" || // 密码有输入即视为变化
      model !== config.model ||
      toolTimeout !== config.tool_timeout;
    setHasChanges(changed);
  }, [localEnabled, provider, apiKey, model, toolTimeout, config]);

  // 提供商切换时更新模型
  useEffect(() => {
    if (!config) return;
    const providerInfo = config.available_providers.find((p) => p.id === provider);
    if (providerInfo && providerInfo.models.length > 0) {
      // 如果当前模型不在新提供商的模型列表中，切换到第一个模型
      if (!providerInfo.models.includes(model)) {
        setModel(providerInfo.models[0]);
      }
    }
  }, [provider, config, model]);

  // 保存配置
  const handleSave = useCallback(async () => {
    if (!isAdmin || !hasChanges) return;
    setSaving(true);
    try {
      const data: AIConfigUpdateRequest = {
        enabled: localEnabled,
        provider,
        model,
        tool_timeout: toolTimeout,
      };
      // 只有输入了 API Key 才更新
      if (apiKey) {
        data.api_key = apiKey;
      }
      const res = await updateAIConfig(data);
      setConfig(res.data);
      setApiKey(""); // 保存后清空 API Key 输入
      if (res.data.requires_restart) {
        toast.success("AI 配置已保存，需要重启服务才能生效");
      } else {
        toast.success("AI 配置已保存");
      }
    } catch (err) {
      console.error("Failed to save AI config:", err);
      toast.error("保存失败");
    } finally {
      setSaving(false);
    }
  }, [isAdmin, hasChanges, localEnabled, provider, apiKey, model, toolTimeout]);

  // 测试连接
  const handleTest = useCallback(async () => {
    setTesting(true);
    try {
      const res = await testAIConnection();
      if (res.data.success) {
        toast.success(res.data.message);
      } else {
        toast.error(res.data.message);
      }
    } catch (err) {
      console.error("Failed to test AI connection:", err);
      toast.error("测试失败");
    } finally {
      setTesting(false);
    }
  }, []);

  // 切换启用状态
  const handleToggle = async () => {
    if (!isAdmin) return;
    const newEnabled = !localEnabled;
    setLocalEnabled(newEnabled);
    // 立即保存开关状态
    setSaving(true);
    try {
      const res = await updateAIConfig({ enabled: newEnabled });
      setConfig(res.data);
      toast.success(newEnabled ? "AI 功能已启用" : "AI 功能已禁用");
    } finally {
      setSaving(false);
    }
  };

  // 获取当前提供商的模型列表
  const getModelsForProvider = () => {
    if (!config) return [];
    const providerInfo = config.available_providers.find((p) => p.id === provider);
    return providerInfo?.models || [];
  };

  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center py-12">
          <LoadingSpinner />
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-6">
        <PageHeader
          title="AI 配置"
          description="配置 AI 功能，包括提供商、API Key 和模型选择"
        />

        {/* Guest 提示 */}
        {isGuest && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
            <AlertTriangle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0" />
            <p className="text-sm text-yellow-800 dark:text-yellow-300">
              演示模式 - 显示的是示例数据。请登录后查看真实配置。
            </p>
          </div>
        )}

        {/* 非 Admin 提示 */}
        {!isGuest && !isAdmin && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
            <AlertTriangle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
            <p className="text-sm text-blue-800 dark:text-blue-300">
              您只有查看权限。如需修改配置，请联系管理员。
            </p>
          </div>
        )}

        {/* AI 配置卡片 */}
        <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
          {/* 头部 */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-[var(--border-color)]">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-violet-100 dark:bg-violet-900/40 flex items-center justify-center">
                <Bot className="w-5 h-5 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <h3 className="font-medium text-default">AI 功能</h3>
                <p className="text-sm text-muted">
                  {config?.effective_enabled ? (
                    <span className="text-green-600 flex items-center gap-1">
                      <Check className="w-3 h-3" /> 已启用
                    </span>
                  ) : localEnabled ? (
                    <span className="text-yellow-600">配置不完整</span>
                  ) : (
                    "已禁用"
                  )}
                </p>
              </div>
            </div>

            {/* 开关 */}
            <button
              onClick={handleToggle}
              disabled={!isAdmin || saving}
              className={`relative w-12 h-6 rounded-full transition-colors ${
                localEnabled ? "bg-green-500" : "bg-gray-300 dark:bg-gray-600"
              } ${!isAdmin ? "opacity-50 cursor-not-allowed" : "cursor-pointer"}`}
            >
              <span
                className={`absolute top-1 left-1 w-4 h-4 rounded-full bg-white transition-transform ${
                  localEnabled ? "translate-x-6" : "translate-x-0"
                }`}
              />
            </button>
          </div>

          {/* 校验错误提示 */}
          {config?.validation_errors && config.validation_errors.length > 0 && (
            <div className="mx-6 mt-4 p-3 rounded-lg bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800">
              <div className="flex items-start gap-2">
                <AlertCircle className="w-4 h-4 text-yellow-600 mt-0.5 flex-shrink-0" />
                <div className="text-sm text-yellow-700 dark:text-yellow-400">
                  <p className="font-medium mb-1">配置不完整</p>
                  <ul className="list-disc list-inside space-y-0.5">
                    {config.validation_errors.map((err, i) => (
                      <li key={i}>{err}</li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>
          )}

          {/* 配置表单 */}
          <div className="px-6 py-4 space-y-4">
            {/* Provider 选择 */}
            <div>
              <label className="block text-sm font-medium text-default mb-2">
                AI 提供商 <span className="text-red-500">*</span>
              </label>
              <select
                value={provider}
                onChange={(e) => setProvider(e.target.value)}
                disabled={!isAdmin}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
                  bg-[var(--bg-primary)] text-default
                  focus:outline-none focus:ring-2 focus:ring-violet-500/50
                  disabled:opacity-60 disabled:cursor-not-allowed
                  [&>option]:bg-white [&>option]:text-gray-900
                  dark:[&>option]:bg-gray-800 dark:[&>option]:text-gray-100"
              >
                {config?.available_providers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>

            {/* API Key */}
            <div>
              <label className="block text-sm font-medium text-default mb-2">
                API Key <span className="text-red-500">*</span>
              </label>
              <div className="relative">
                <input
                  type={showApiKey ? "text" : "password"}
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder={config?.api_key_set ? "留空则不修改" : "请输入 API Key"}
                  disabled={!isAdmin}
                  className="w-full px-3 py-2 pr-10 rounded-lg border border-[var(--border-color)]
                    bg-[var(--bg-primary)] text-default font-mono
                    placeholder:text-muted
                    focus:outline-none focus:ring-2 focus:ring-violet-500/50
                    disabled:opacity-60 disabled:cursor-not-allowed"
                />
                <button
                  type="button"
                  onClick={() => setShowApiKey(!showApiKey)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted hover:text-default"
                >
                  {showApiKey ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                </button>
              </div>
              {config?.api_key_set && (
                <p className="mt-1 text-xs text-muted font-mono">
                  当前: {config.api_key_masked}
                </p>
              )}
            </div>

            {/* Model 选择 */}
            <div>
              <label className="block text-sm font-medium text-default mb-2">
                模型 <span className="text-red-500">*</span>
              </label>
              <select
                value={model}
                onChange={(e) => setModel(e.target.value)}
                disabled={!isAdmin}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
                  bg-[var(--bg-primary)] text-default
                  focus:outline-none focus:ring-2 focus:ring-violet-500/50
                  disabled:opacity-60 disabled:cursor-not-allowed
                  [&>option]:bg-white [&>option]:text-gray-900
                  dark:[&>option]:bg-gray-800 dark:[&>option]:text-gray-100"
              >
                {getModelsForProvider().map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            </div>

            {/* Tool Timeout */}
            <div>
              <label className="block text-sm font-medium text-default mb-2">
                Tool 调用超时（秒）
              </label>
              <input
                type="number"
                value={toolTimeout}
                onChange={(e) => setToolTimeout(parseInt(e.target.value) || 30)}
                min={5}
                max={300}
                disabled={!isAdmin}
                className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)]
                  bg-[var(--bg-primary)] text-default
                  focus:outline-none focus:ring-2 focus:ring-violet-500/50
                  disabled:opacity-60 disabled:cursor-not-allowed"
              />
              <p className="mt-1 text-xs text-muted">
                AI 调用 Kubernetes 查询工具的超时时间，建议 30-60 秒
              </p>
            </div>
          </div>

          {/* 操作按钮 */}
          {isAdmin && (
            <div className="flex justify-end gap-3 px-6 py-4 border-t border-[var(--border-color)] bg-[var(--bg-secondary)]">
              <button
                onClick={handleTest}
                disabled={testing || !config?.effective_enabled}
                className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)]
                  bg-[var(--bg-primary)] text-default
                  hover:bg-[var(--bg-secondary)]
                  disabled:opacity-50 disabled:cursor-not-allowed
                  transition-colors flex items-center gap-2"
              >
                {testing ? (
                  <Loader2 className="w-4 h-4 animate-spin" />
                ) : (
                  <RefreshCw className="w-4 h-4" />
                )}
                测试连接
              </button>
              <button
                onClick={handleSave}
                disabled={saving || !hasChanges}
                className="px-4 py-2 text-sm rounded-lg
                  bg-violet-600 text-white
                  hover:bg-violet-700
                  disabled:opacity-50 disabled:cursor-not-allowed
                  transition-colors flex items-center gap-2"
              >
                {saving && <Loader2 className="w-4 h-4 animate-spin" />}
                保存
              </button>
            </div>
          )}
        </div>

        {/* 重启提示 */}
        {config?.requires_restart && (
          <div className="flex items-center gap-3 p-4 rounded-lg bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800">
            <RefreshCw className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
            <p className="text-sm text-blue-800 dark:text-blue-300">
              配置已修改，需要重启 Master 服务才能生效。
            </p>
          </div>
        )}
      </div>
    </Layout>
  );
}
