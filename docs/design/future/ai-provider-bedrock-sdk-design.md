# AI Provider：接入 AWS Bedrock + 各家改官方 SDK —— 实施计划

> 状态：计划已裁定（2026-10-12：D1–D8 用户裁定 + work 审阅补充 4 点已并入），PR-A 代码完成（分支 `feat/bedrock-provider`），待推 dev 与冒烟
> 用户已裁定：① 先 A 后 B，两个 PR　② ~~新增 `region` 字段~~ → **被 D1 推翻：region 不进数据库，写进 Master 配置**　③ 第 1 轮零 tool call 时追加提示要求先取证、最多重试一次（只作用于 analyze，D3）　④ 由 atlhyper 排进 tracker
> 凭证约束：Bedrock API key 由用户在 Web 设置页填写；⛔ 会话不经手明文，⛔ 不从环境变量 / 凭证文件取

## 一、清单核实结果（对照 work 预读清单）

| # | work 原条目 | 核实结论 |
|---|---|---|
| 1 | 验证 SDK 是否原生支持 Bedrock API key | ✅ **原生支持**，见第二节 |
| 2 | `ai/llm/bedrock/` 用 ConverseStream 实现 `ChatStream` | ✅ 可行。接口只有 `ChatStream` + `Close`（`ai/llm/interfaces.go`）；另需在 `ai/factory.go:8-11` 加空白导入（provider 靠 `init()` 注册） |
| 3 | 鉴权走 Web 填写的 key | ✅ key 已经 AES 加密存库（`database/repo/ai_provider.go:27,51`），经 `ai/role.go:providerToRoleConfig`（101-115）进入 `llm.Config.APIKey` |
| 4 | region：llm.Config + DB + API + 前端 | ⚠️ 现有库加不了列（`migrate()` 只有 `CREATE … IF NOT EXISTS`，全仓无 `ALTER TABLE`，prod 是 PVC 真数据）→ **D1 裁定：region 改走 Master 配置**，DB / DTO / 前端 / AISeed 均不涉及（见第三节） |
| 5 | `ai_provider.go:22` supportedProviders 加 bedrock | ✅ 另需 `providerNames`（25-30）、创建校验（216-273：bedrock 要 key；region 不在 provider 上，见 D1）；`updateProvider`（294-340）不复验类型与 BaseURL，顺带补 |
| 6 | `migrations.go:535` 模型目录加三个模型 | ⚠️ `context_window` 必须按模型卡逐个填（`ai/context.go` 据此裁剪消息）。⚠️ **种子只在空表时写入**（`initDefaultAIModels` 515-573 遇 `COUNT(*)>0` 直接返回）→ 已有库拿不到新模型；而前端下拉**只由后端 `models` 字段生成**（`ProviderModal.tsx:81`，`loadModelsGrouped` 只输出有目录行的 provider）→ **已有部署上 Bedrock 在界面上根本不出现**。需改为逐行 `INSERT OR IGNORE`（D2） |
| 7 | 前端 ProviderCard / ProviderModal | ✅ 另需：`useProviderForm.ts:12,52-54`、`api/ai-provider.ts:10-64` 类型、i18n（zh/ja + `types/i18n.ts`）、mock。现有缺陷：`useAISettings.ts:89` 对 ollama 也强制 key（与后端矛盾）；更新时 `baseUrl \|\| undefined`（101）导致 BaseURL 无法清空 |
| 8 | `settings.go:63 / 200–247` 写死 gemini | ✅ 与新列表**不冲突**：那是独立的旧接口 `/api/v2/settings/ai`，存 settings 表的 `ai.*` 键（明文）、AI 服务从不读取；前端 `api/settings.ts` 的三个函数无人引用；且 snake_case / camelCase 不一致，本就不可用 → **死代码**，建议删除（D5） |
| 9–12 | 四家改官方 SDK | ✅ 路径与版本见第二节。⚠️ openai 应为 **`github.com/openai/openai-go/v3`**（无后缀版停在 v1.12.0，2025-07）；ollama 的 `api` 包引入整个服务端仓依赖且为 v0.x（D4） |
| 13 | `analyze.go:111` / `chat.go:166` 零 tool call 即退出 | ✅ 确认。另发现**加剧该问题的两个缺陷**：`analyze.go:collectStreamResponse`（186-208）**吞掉 ChunkError**，流出错被当成「正常结束且无 tool call」；`openai/client.go` 只在 `finish_reason=="tool_calls"` 时发出 tool call，兼容服务端以 `stop` 收尾时 tool call 全丢。③ 是否也作用于 chat 需裁定（D3） |
| 14 | 置信度格式校验 | ⚠️ **当前无可校验对象**：`Confidence` 在 `enricher.go:371` 解析后即丢弃，不入 `SummarizeResponse` / `AIReport` / DB / UI → 要做需另起设计（D6） |
| 15–16 | 单测 + 真 key 端到端 | ✅ 但 `ai/`、`ai/llm/` **零测试**。B 的「行为不变」需先补**特征测试**再换实现（第四节）。真 key 测试只能经 Web 配置（会话不经手 key） |

**清单外补充**（核实中发现，计划内顺带修）：

| 问题 | 位置 | 处置 |
|---|---|---|
| gemini tool call ID = `call_<工具名>`，同轮两次调用同一工具 ID 重复 | `gemini/client.go:~133` | B：SDK 迁移时改为唯一 ID |
| anthropic 只读顶层 usage，`message_start` 的 `message.usage` 丢失 → 输入 token 记 0，预算统计偏低 | `anthropic/client.go` | B：SDK 自带累加器 |
| 创建 / 更新响应对**密文**做掩码（repo 原地加密后再转响应） | `ai_provider.go` + `repo/ai_provider.go` | A：顺带修 |
| HTTP 客户端无超时（只靠 ctx） | 各 `client.go` | B：SDK 迁移时统一 |
| 文档仍列四家与旧接口 | `master-api-reference.md:354-373,409`、`web-api-reference.md:476-494`、README ×3 | A / B 各自同步 |

## 二、SDK 查证结论（第 1 项）

**aws-sdk-go-v2 原生支持 Bedrock API key（Bearer）。**

| 版本 | 内容 | 出处 |
|---|---|---|
| `service/bedrockruntime` v1.31.0（2025-06-30） | 「Add API Key … support for Bedrock Runtime APIs」：`Options.BearerAuthTokenProvider`，服务声明 `smithy.api#httpBearerAuth` | 该模块 `CHANGELOG.md` |
| v1.33.0（2025-08-04）+ core v1.37.2 | `Options.AuthSchemePreference` | 同上；core `aws/config.go`（v1.37.1 无此字段，v1.37.2 有） |
| v1.35.0（2025-08-07） | 环境变量 `AWS_BEARER_TOKEN_BEDROCK` | 同上 |
| 当前最新 | bedrockruntime **v1.63.4** / core **v1.47.3** / smithy-go **v1.28.5**（均 `go 1.24`） | `go list -m @latest` |

**静态 key 的构造方式**（本机 `httptest` TLS 服务实测：请求头为 `Authorization: Bearer <key>`，Converse 与 ConverseStream 均正确解码 text / tool-use delta / stop / usage）：

```go
client := bedrockruntime.New(bedrockruntime.Options{Region: region},
    func(o *bedrockruntime.Options) { // ⚠️ 必须写在 optFn 里，见下
        o.BearerAuthTokenProvider = bearer.StaticTokenProvider{Token: bearer.Token{Value: apiKey}}
        o.AuthSchemePreference = []string{"httpBearerAuth"}
    })
```

- **不调用 `config.LoadDefaultConfig`** → `Options.Credentials == nil` → SigV4 解析器为 nil（`options.go:163-209`）→ 不触发凭证链查找，自然落到 Bearer（`auth.go:129-143,180-199`）。
- ⚠️ **实测踩坑**：`New()` 在复制 Options 之后、执行 optFn 之前调用 `resolveEnvBearerToken`（`api_client.go:200-231`、`1073-1092`）—— 若环境变量 `AWS_BEARER_TOKEN_BEDROCK` 存在，会**静默覆盖** Options 字面量里的 token。写在 optFn 里则以我方为准，代码对该环境变量零依赖。须有单测守住（设该环境变量后断言请求头仍是我方 key）。
- Bearer 签名器拒绝非 HTTPS（`smithy-go auth/bearer/middleware.go:88-96`）。
- ConverseStream 只有响应是事件流，输入不签名（`api_op_ConverseStream.go:367-373`）→ Bearer 无障碍。
- ⚠️ v1.63.0（2026-09-15）该服务切换为 schema-based (de)serialization；其他服务在同一改动中出过 union / document 反序列化缺陷。**固定版本 + 回归测试**。

**ConverseStream 映射要点**（`types` 包）：工具定义 `ToolMemberToolSpec{InputSchema: ToolInputSchemaMemberJson{document.NewLazyDocument(schema)}}`；
流事件 `ContentBlockStart(ToolUse{ToolUseId, Name})` → `ContentBlockDelta(ToolUse.Input 为 JSON 片段，按 ContentBlockIndex 拼接，ContentBlockStop 时解析)` → `MessageStop(StopReason)` → `Metadata(Usage)`；
另有 `ReasoningContent` delta（Qwen / gpt-oss 会出，需忽略或单独处理，⛔ 不得混进正文）；回填历史时 tool use 须重建为 `document.NewLazyDocument(map)`，tool result 以 `user` 角色发回。

**三个模型（东京，均 In-Region on-demand，直接用模型 ID，无需推理配置文件）**：

| 模型 | 上下文 / 最大输出 | bedrock-runtime 工具调用 |
|---|---|---|
| `qwen.qwen3-235b-a22b-2507-v1:0` | 256K / 8K | ✅ 模型卡列出 |
| `openai.gpt-oss-120b-1:0` | 128K / 16K | ⚠️ 模型卡只在 bedrock-mantle 列出 |
| `minimax.minimax-m2.5`（无 `:0`） | 196K / 8K | ⚠️ bedrock-runtime 两列均未列出 |

出处：AWS 模型卡 `model-card-qwen-qwen3-235b-a22b-2507.html` / `model-card-openai-gpt-oss-120b.html` / `model-card-minimax-minimax-m2-5.html`。
已知先例：部分模型在 ConverseStream 带 `toolConfig` 时报「doesn't support tool use in streaming mode」（re:Post，Mistral Large）。

**IAM（work 核对）**：现策略放行 `InvokeModel` / `InvokeModelWithResponseStream` / `CallWithBearerToken`。主方案 ConverseStream 所需的 `InvokeModelWithResponseStream` 已在策略内。
备选「OpenAI 兼容端点 `/openai/v1`」所需权限**未核实** —— ⛔ 不得假设现有 key 能调；若冒烟后真要走备选，先告知 work 改 Terraform（config 仓 `clusters/platform/ai/bedrock/`）。
⇒ **gpt-oss 与 minimax 的「流式 + 工具调用」未经证实**，须逐模型冒烟后才进模型目录（见第四节）。

**四家官方 SDK**（均可在 Go 1.26.6 下构建）：

| 家 | 导入路径 / 最新 | 状态 | 流式 + 工具 |
|---|---|---|---|
| Google | `google.golang.org/genai` v1.73.0 | GA。旧 `generative-ai-go` 已于 **2025-08-31 终止支持** | `Models.GenerateContentStream` → `iter.Seq2`；`FunctionDeclaration.ParametersJsonSchema` 收原始 JSON Schema |
| Anthropic | `github.com/anthropics/anthropic-sdk-go` v1.80.0 | 稳定 v1 | `Messages.NewStreaming` + `Message.Accumulate`（usage 自动累加） |
| OpenAI | `github.com/openai/openai-go/v3` v3.77.0（`go 1.26.0`） | 当前主版本 | `Chat.Completions.NewStreaming` + `ChatCompletionAccumulator` |
| Ollama | `github.com/ollama/ollama/api` v0.40.3（`go 1.26.0`） | 公开包、官方 CLI 自用；⚠️ v0.x 无 semver 承诺，模块是整个服务端仓（带入 `x/crypto/ssh`、`yaml.v3` 等） | `Client.Chat(ctx, req, fn)` 回调流；用 `api.NewClient(baseURL, httpClient)`，⛔ 不用读 `OLLAMA_HOST` 的 `ClientFromEnvironment` |

## 三、PR 拆分与改动清单

### PR-A：Bedrock provider + 配置 region + 循环修复（③）+ 删旧接口

```
go.mod                                    aws-sdk-go-v2 core ≥ v1.37.2、service/bedrockruntime（固定版本）、smithy-go
atlhyper_master_v2/
├── config/types.go                       [修改] AIConfig 增 BedrockRegion
├── config/defaults.go                    [修改] "MASTER_AI_BEDROCK_REGION": "ap-northeast-1"
├── config/loader.go                      [修改] getString("MASTER_AI_BEDROCK_REGION")
├── ai/llm/interfaces.go                  [修改] Config 增 Region（由 role.go 从全局配置填入，不来自 DB）
├── ai/llm/bedrock/                       [新增]
│   ├── client.go                         New(optFn 注入 Bearer)、ChatStream、Close；Region 为空时报配置错误
│   ├── convert.go                        Request/Message/ToolDefinition ↔ Converse 类型
│   ├── stream.go                         事件流 → Chunk（tool-use 片段按 index 拼接；忽略 reasoning；usage）
│   └── *_test.go                         转换单测 + httptest TLS 假服务（事件流编码）+ 环境变量覆盖防线
├── ai/factory.go                         [修改] 空白导入 bedrock
├── ai/role.go                            [修改] providerToRoleConfig：provider=bedrock 时填 Region = 配置值
├── ai/analyze.go                         [修改] ③：第 1 轮零 tool call → 追加取证提示、重试一次；collectStreamResponse 不再吞错
├── ai/prompts/                           [修改] 取证提示文案
├── database/sqlite/migrations.go         [修改] 模型目录逐行 INSERT OR IGNORE（D2）+ 三个 Bedrock 模型（context_window 按模型卡）
├── gateway/handler/admin/ai_provider.go  [修改] supportedProviders / 显示名「AWS Bedrock」/ bedrock 要求 key / 掩码密文缺陷
├── gateway/handler/admin/settings.go     [删除] 旧 /settings/ai（D5）
├── gateway/routes.go                     [修改] 只删 290、306 两条路由；⛔ 保留 pAISettingsRead / pAISettingsWrite（291-292 起新 provider 接口在用）
└── gateway/routes_test.go                [修改] 删 84 行（该路由不复存在，否则断言 401 会得 404）
atlhyper_web/src/
├── api/ai-provider.ts                    [修改] mock 增 bedrock
├── app/settings/ai/components/
│   ├── ProviderCard.tsx                  [修改] 颜色 / 显示名
│   └── useAISettings.ts                  [修改] ollama 不强制 key
├── i18n/ + types/i18n.ts                 [修改]
└── api/settings.ts                       [删除] 旧接口的三个无人引用的函数（D5）
docs/static/reference/*.md、README ×3     [修改] 增 bedrock；删旧 /settings/ai
```

**region 的取值链**：`MASTER_AI_BEDROCK_REGION`（缺省 `ap-northeast-1`）→ `config.AI.BedrockRegion` →
`role.go` 构造 `llm.Config.Region` → bedrock 工厂。所有 bedrock provider 共用同一 region。
**不涉及**：`ai_providers` 表结构、迁移助手、§1.9 修订、DTO、前端 region 输入框、`AISeed.Region` —— 均因 D1 取消。

**与「禁止写死环境值」的关系**：region 走环境变量、代码只提供缺省值，与现有 `MASTER_*` 配置同一机制，不违反该约束。
部署清单**无需改动**：用缺省值即可；将来要换区时，在 `clusters/requiem/apps/atlhyper/{dev,prod}/atlhyper-config.yaml` 的 ConfigMap 加该键。
本地开发同理（`config/local/ubuntu/env/atlhyper.env` 不加也可运行）。

**单独一步（⛔ 不并入 PR、执行前须用户单独批准）**：删掉旧接口只删代码；prod settings 表里旧接口写下的 `ai.*` 明文行仍在（含已失效的 Gemini key）。
清理是对 prod 库的 `DELETE FROM settings WHERE key LIKE 'ai.%'`（执行前先 `SELECT key` 列出行数给用户看）。dev 为 emptyDir，重启即无，不需处理。

### PR-B：四家改官方 SDK（行为不变）

```
ai/llm/gemini/     generative-ai-go → google.golang.org/genai（唯一 tool call ID）
ai/llm/anthropic/  手写 net/http → anthropic-sdk-go（usage 累加修复；默认模型更新，D7）
ai/llm/openai/     手写 → openai-go/v3（按增量累加 tool call，不再依赖 finish_reason）
ai/llm/ollama/     手写 → ollama/api（D4）
go.mod             移除 generative-ai-go；核查 google.golang.org/api 是否仍被引用
```

## 四、测试方案

| 层 | 内容 | 何时 |
|---|---|---|
| 特征测试（B 的前置） | 用录制的 SSE / NDJSON 夹具 + `httptest` 驱动**现有**四家客户端，断言 Chunk 序列（文本 / tool call / done+usage）。先在旧实现上全绿，再换 SDK 保持全绿 —— 「行为不变」才有判据 | PR-B 第一步 |
| 单测 | Bedrock 转换（含 document 往返、tool result 角色）、流组装（片段拼接、多 tool call、reasoning 不入正文）、Bearer 头、`AWS_BEARER_TOKEN_BEDROCK` 覆盖防线 | PR-A |
| 循环 | 假 LLMClient：第 1 轮零 tool call → 断言追加提示并重试一次、第二次仍零则收尾；流出错 → 断言报错而非「完成」 | PR-A |
| 配置 | 未设环境变量 → region = `ap-northeast-1`；设置后取设置值；bedrock 工厂拿到空 region 时报配置错误 | PR-A |
| 模型目录 | 已有目录行的库 → 启动后补出三个 Bedrock 模型（context_window 正确）、已有行不变、重复执行幂等 | PR-A |
| 逐模型冒烟 | dev 上经 Web 配置 bedrock（用户填 key），每个模型各跑：普通对话 / 流式 + ≥ 2 轮 tool call。**通过者才留在模型目录**（gpt-oss、minimax 可能落选） | PR-A 上 dev 后 |
| 端到端 | dev 上用 bedrock 跑一次真实事件分析（AIOps 深度分析） | PR-A 上 dev 后 |

真 key 只经 Web 设置页进入（加密存库），会话只观察结果，不经手明文。
**D8**：dev 的 Master 库是 emptyDir，**每次重启后由用户经 Web 重填 key**；⛔ 不放进 K8s Secret、⛔ 不走 `MASTER_AI_SEED_*`。冒烟前先确认 dev 上 bedrock provider 仍在。

## 五、风险与回滚

| 风险 | 缓解 / 回滚 |
|---|---|
| 模型目录新增行 | 只 INSERT 不改不删；回退镜像后多出的目录行无害（旧代码只按 provider+model 查） |
| bedrockruntime 新 serde 回归 | 固定版本 + 事件流回归测试；出问题退到 schema serde 之前的版本（< v1.63.0） |
| 某模型流式不支持 tool use | 冒烟不过就从目录移除；备选 OpenAI 兼容端点**权限未核实**，需先由 work 改 Terraform，⛔ 不假设现有 key 可用 |
| ③ 的追加提示导致多一轮 token | 只在第 1 轮零 tool call 时触发、至多一次；账户月预算 $10 兜底 |
| PR-B 行为漂移 | 特征测试守护；四家各自独立提交，可单独回退 |
| ollama 模块随服务端仓频繁变动 | 固定版本；D4 |

## 六、裁定记录（用户 2026-10-12，经 work 转达）

| # | 事项 | 裁定 | 理由 / 连带 |
|---|---|---|---|
| D1 | region 放哪 | **写进 Master 配置**（`MASTER_AI_BEDROCK_REGION`，缺省 `ap-northeast-1`），⛔ 不进数据库 —— 推翻原裁定② | 用户原话「代码写死在配置文件中」。连带取消：加列、迁移助手、§1.9 修订、DTO / 前端 region 输入、`AISeed.Region` |
| D2 | 模型目录种子 | 逐行 `INSERT OR IGNORE` | 已有库需能拿到新模型；是 INSERT 不是加列，不受 D1 影响 |
| D3 | ③ 的作用范围 | 只作用于 analyze | 对话里打招呼 / 问概念本不需要工具 |
| D4 | Ollama | 官方 `api` 包 + 固定版本 | 与「改官方 SDK」一致 |
| D5 | 旧 `/api/v2/settings/ai` | PR-A 删除代码；prod 残留行清理单独批准 | 见第三节「单独一步」 |
| D6 | 置信度 | 本次不做，另立设计 | 当前解析后即丢弃，无存储 / 展示 |
| D7 | Anthropic 默认模型 | PR-B 改为当前主力 Sonnet，ID 以官方列表为准 | —— |
| D8 | dev 冒烟时 key 怎么进 | **每次重启后经 Web 重填** | ⛔ 不进 K8s Secret、⛔ 不走 `MASTER_AI_SEED_*` |

**work 审阅补充（已并入）**：① prod `ai.*` 残留行清理单列、单独批准　② 删路由时保留 `pAISettingsRead/Write`，只删 routes.go:290/306 + routes_test.go:84　③ 模型目录 context_window 按模型卡逐个填　④ OpenAI 兼容端点权限未核实，走备选前先由 work 改 Terraform。
**更正 work 的一处假设**：Master 配置不用 `caarlos0/env`，而是自有 loader（`config/defaults.go` 缺省值 map + `config/loader.go` 的 `getString`），命名沿用 `MASTER_AI_*` 前缀。

**已由 Claude 定的工程项**（不需拍板，列出供知情）：Bedrock 走原生 ConverseStream；bedrockruntime 固定版本；`reasoning` 增量不进正文；tool call ID 唯一化；HTTP 超时统一；特征测试先行；region 经 `llm.Config.Region` 透传（不让 bedrock 包直接读全局配置，保持 llm 层与 config 解耦）。

## 七、与 atlhyper 当前工作的排期关系

- 当前 atlhyper 无进行中的编码任务；待办均为等用户验证 / 决定的事项（OIDC 浏览器验证、dev 密钥轮换、prod 上线）。
- 本计划与上述事项**无代码冲突**（权限模型 / OIDC 不碰 `ai/`）。唯一交叉：`ai/chat` 的权限 `ai.chat` 已在权限模型中拆出，不受影响。
- PR-A 预计：后端 + 前端 + 迁移 + 测试一期完成；上 dev 后需用户在 Web 填 key 做逐模型冒烟。PR-B 在 PR-A 冒烟通过后开始。
- 开工时机：用户在 atlhyper 侧决定。
