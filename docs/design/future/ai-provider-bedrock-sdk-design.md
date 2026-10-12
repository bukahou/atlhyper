# AI Provider：接入 AWS Bedrock + 各家改官方 SDK —— 实施计划

> 状态：实施计划草案（2026-10-12），送 work 审阅；开工时机由用户在 atlhyper 侧决定
> 用户已裁定：① 先 A 后 B，两个 PR　② 新增 `region` 字段（默认 `ap-northeast-1`），不借用 BaseURL　③ 第 1 轮零 tool call 时追加提示要求先取证、最多重试一次　④ 由 atlhyper 排进 tracker
> 凭证约束：Bedrock API key 由用户在 Web 设置页填写；⛔ 会话不经手明文，⛔ 不从环境变量 / 凭证文件取

## 一、清单核实结果（对照 work 预读清单）

| # | work 原条目 | 核实结论 |
|---|---|---|
| 1 | 验证 SDK 是否原生支持 Bedrock API key | ✅ **原生支持**，见第二节 |
| 2 | `ai/llm/bedrock/` 用 ConverseStream 实现 `ChatStream` | ✅ 可行。接口只有 `ChatStream` + `Close`（`ai/llm/interfaces.go`）；另需在 `ai/factory.go:8-11` 加空白导入（provider 靠 `init()` 注册） |
| 3 | 鉴权走 Web 填写的 key | ✅ key 已经 AES 加密存库（`database/repo/ai_provider.go:27,51`），经 `ai/role.go:providerToRoleConfig`（101-115）进入 `llm.Config.APIKey` |
| 4 | region：llm.Config + DB + API + 前端 | ⚠️ **漏了关键一环：现有库加不了列**。`migrate()` 只有 `CREATE … IF NOT EXISTS`，全仓无 `ALTER TABLE`；prod 是 PVC 真数据 → 需裁定（第六节 D1）。另需改 `database/types.go:210`、`database/sqlite/ai.go:113/128/152/171`、`config/types.go:76`（AISeed）、`config/loader.go:83`、`database/sync.go:SeedAIProvider` |
| 5 | `ai_provider.go:22` supportedProviders 加 bedrock | ✅ 另需 `providerNames`（25-30）、创建校验（216-273：bedrock 要 key、region 校验 / 缺省）；`updateProvider`（294-340）不复验类型与 BaseURL，顺带补 |
| 6 | `migrations.go:535` 模型目录加三个模型 | ⚠️ **种子只在空表时写入**（`initDefaultAIModels` 515-573 遇 `COUNT(*)>0` 直接返回）→ 已有库拿不到新模型；而前端下拉**只由后端 `models` 字段生成**（`ProviderModal.tsx:81`，`loadModelsGrouped` 只输出有目录行的 provider）→ **已有部署上 Bedrock 在界面上根本不出现**。需改为逐行 `INSERT OR IGNORE`（D2） |
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
⇒ **gpt-oss 与 minimax 的「流式 + 工具调用」未经证实**，须逐模型冒烟后才进模型目录（见第四节）。

**四家官方 SDK**（均可在 Go 1.26.6 下构建）：

| 家 | 导入路径 / 最新 | 状态 | 流式 + 工具 |
|---|---|---|---|
| Google | `google.golang.org/genai` v1.73.0 | GA。旧 `generative-ai-go` 已于 **2025-08-31 终止支持** | `Models.GenerateContentStream` → `iter.Seq2`；`FunctionDeclaration.ParametersJsonSchema` 收原始 JSON Schema |
| Anthropic | `github.com/anthropics/anthropic-sdk-go` v1.80.0 | 稳定 v1 | `Messages.NewStreaming` + `Message.Accumulate`（usage 自动累加） |
| OpenAI | `github.com/openai/openai-go/v3` v3.77.0（`go 1.26.0`） | 当前主版本 | `Chat.Completions.NewStreaming` + `ChatCompletionAccumulator` |
| Ollama | `github.com/ollama/ollama/api` v0.40.3（`go 1.26.0`） | 公开包、官方 CLI 自用；⚠️ v0.x 无 semver 承诺，模块是整个服务端仓（带入 `x/crypto/ssh`、`yaml.v3` 等） | `Client.Chat(ctx, req, fn)` 回调流；用 `api.NewClient(baseURL, httpClient)`，⛔ 不用读 `OLLAMA_HOST` 的 `ClientFromEnvironment` |

## 三、PR 拆分与改动清单

### PR-A：Bedrock provider + region + 循环修复（③）

```
go.mod                                    aws-sdk-go-v2 core ≥ v1.37.2、service/bedrockruntime（固定版本）、smithy-go
atlhyper_master_v2/
├── ai/llm/interfaces.go                  [修改] Config 增 Region
├── ai/llm/bedrock/                       [新增]
│   ├── client.go                         New(optFn 注入 Bearer)、ChatStream、Close
│   ├── convert.go                        Request/Message/ToolDefinition ↔ Converse 类型
│   ├── stream.go                         事件流 → Chunk（tool-use 片段按 index 拼接；忽略 reasoning；usage）
│   └── *_test.go                         转换单测 + httptest TLS 假服务（事件流编码）+ 环境变量覆盖防线
├── ai/factory.go                         [修改] 空白导入 bedrock
├── ai/role.go                            [修改] providerToRoleConfig 透传 Region
├── ai/analyze.go                         [修改] ③：第 1 轮零 tool call → 追加取证提示、重试一次；collectStreamResponse 不再吞错
├── ai/prompts/                           [修改] 取证提示文案
├── database/sqlite/migrations.go         [修改] ai_providers.region（D1）；模型目录逐行 INSERT OR IGNORE（D2）+ Bedrock 模型
├── database/types.go / sqlite/ai.go      [修改] Region 读写
├── database/sync.go / config/*           [修改] AISeed.Region（MASTER_AI_SEED_REGION）
├── gateway/handler/admin/ai_provider.go  [修改] supportedProviders / 显示名「AWS Bedrock」/ 校验 / DTO region / 掩码密文缺陷
└── gateway/handler/admin/settings.go     [删除，若 D5 = 删] 旧 /settings/ai
atlhyper_web/src/
├── api/ai-provider.ts                    [修改] 类型 region + mock
├── app/settings/ai/components/
│   ├── ProviderModal.tsx                 [修改] region 输入（仅 bedrock，默认 ap-northeast-1）
│   ├── ProviderCard.tsx                  [修改] 颜色 / 显示名
│   ├── useProviderForm.ts                [修改] region
│   └── useAISettings.ts                  [修改] ollama 不强制 key
├── i18n/ + types/i18n.ts                 [修改]
└── api/settings.ts                       [删除，若 D5 = 删]
docs/static/reference/*.md、README ×3     [修改]
```

**表结构**：`ai_providers` 增 `region TEXT NOT NULL DEFAULT ''`（空 = 非 bedrock；bedrock 读取时空值按 `ap-northeast-1`）。到达已有库的方式见 D1。
**API**：`ProviderCreateRequest` / `ProviderUpdateRequest`（指针）/ `ProviderResponse` 增 `region`；bedrock 校验 region 形如 `^[a-z]{2}(-[a-z]+)+-\d$`。

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
| 单测 | Bedrock 转换（含 document 往返、tool result 角色）、流组装（片段拼接、多 tool call、reasoning 不入正文）、Bearer 头、`AWS_BEARER_TOKEN_BEDROCK` 覆盖防线、region 缺省 | PR-A |
| 循环 | 假 LLMClient：第 1 轮零 tool call → 断言追加提示并重试一次、第二次仍零则收尾；流出错 → 断言报错而非「完成」 | PR-A |
| 迁移 | 旧结构库文件 → 迁移后有 region 列、旧数据保留、新模型行出现、重复执行幂等 | PR-A |
| 逐模型冒烟 | dev 上经 Web 配置 bedrock（用户填 key），每个模型各跑：普通对话 / 流式 + ≥ 2 轮 tool call。**通过者才进模型目录**（gpt-oss、minimax 可能落选） | PR-A 上 dev 后 |
| 端到端 | dev 上用 bedrock 跑一次真实事件分析（AIOps 深度分析） | PR-A 上 dev 后 |

真 key 只经 Web 设置页进入（加密存库），会话只观察结果，不经手明文。

## 五、风险与回滚

| 风险 | 缓解 / 回滚 |
|---|---|
| region 加列不可逆 | 只加不删，旧镜像按显式列名 SELECT，忽略新列 → 回滚 = 回退镜像即可 |
| bedrockruntime 新 serde 回归 | 固定版本 + 事件流回归测试；出问题退到 schema serde 之前的版本（< v1.63.0） |
| 某模型流式不支持 tool use | 冒烟不过不进目录；备选走 Bedrock 的 OpenAI 兼容端点（`/openai/v1`，同一 key）经 openai-go |
| ③ 的追加提示导致多一轮 token | 只在第 1 轮零 tool call 时触发、至多一次；账户月预算 $10 兜底 |
| PR-B 行为漂移 | 特征测试守护；四家各自独立提交，可单独回退 |
| ollama 模块随服务端仓频繁变动 | 固定版本；D4 |

## 六、需用户拍板

### D1 现有库怎么加 `region` 列
- **背景**：CLAUDE.md §1.9 规定 SQLite「改建表语句 + 删库重建，不写增量迁移」。但 prod 的 Master 库在 PVC 上，存有 provider（含加密 key）、预算、会话、审计；dev 是 emptyDir（每次重启自动重建）。`CREATE TABLE IF NOT EXISTS` 不会补列。
- **影响**：
  - **A. 加一个「只增列」的幂等迁移助手**（`pragma_table_info` 查列是否存在，不存在才 `ALTER TABLE ADD COLUMN`）：已有数据保留；需修订 §1.9 —— 允许「只增不改不删」的列迁移，仍禁止靠吞「duplicate column name」错误来跳过。
  - **B. prod 删库重建**：零迁移代码；代价是 prod 的 provider、预算、会话、审计全部丢失，需手工重配。
  - **C. 不加列，region 存到别处**（如 settings 表按 provider id 存）：零 DDL；但违背裁定②的「新增字段」本意，数据分散两处。
- **推荐 A**：这是今后任何加列都会再遇到的问题，一次立规；风险只在新增。

### D2 模型目录种子改为逐行 `INSERT OR IGNORE`
- **背景**：目录只在空表时写入，已有库拿不到新模型，Bedrock 在界面上不出现。
- **影响**：改后每次启动补齐缺失的内置模型（已有行不动）。代价：若曾手工删过某个内置模型，重启后它会回来（目前没有删目录模型的界面）。
- **推荐改**。

### D3 循环修复（③）是否也作用于 AI 对话
- **背景**：③ 针对「分析」过早收尾。对话场景里，打招呼、问概念等问题本来就不需要工具，强制先取证会多一轮无意义调用。
- **影响**：只改分析 → 对话行为不变；两处都改 → 对话也多一轮。
- **推荐只改 `analyze.go`**；对话保持现状。

### D4 Ollama 用官方 `api` 包还是保留手写 REST
- **背景**：`ollama/api` 是官方公开包，但 v0.x 无兼容承诺，且模块是整个服务端仓。
- **影响**：用官方包 → 与「各家改官方 SDK」裁定一致，代价是依赖变重、升级可能破坏接口；保留手写 → 依赖轻，但违背裁定。
- **推荐用官方包并固定版本**（与裁定一致；破坏性升级由固定版本挡住）。

### D5 删除旧的 `/api/v2/settings/ai` 接口
- **背景**：独立旧接口，AI 服务从不读取，前端无人调用，字段命名本就不一致（不可用），且把 key 明文存在 settings 表。
- **影响**：删除 = 去掉一个对外 API（文档同步删除）；保留 = 继续存在一个可写明文 key 的死接口。
- **推荐在 PR-A 中删除**。

### D6 置信度（第 14 项）
- **背景**：模型报告里的置信度目前解析后即丢弃，没有存储也没有展示。
- **影响**：要做「校验」需先决定存不存、展不展示（DB 加字段 + UI），已超出本次范围。
- **推荐本次不做，另立设计**。

### D7 Anthropic 默认模型
- **背景**：手写客户端默认 `claude-sonnet-4-20250514`；Bedrock 上的 Claude 因 Marketplace 协议未开通，本次不进目录。
- **影响**：只影响直连 Anthropic 时「未填模型」的缺省值。
- **推荐 PR-B 改为当前主力 Sonnet**，具体 ID 届时以 Anthropic 官方模型列表为准。

**已由 Claude 定的工程项**（不需拍板，列出供知情）：Bedrock 走原生 ConverseStream（与裁定一致），OpenAI 兼容端点只作备选；bedrockruntime 固定版本；`reasoning` 增量不进正文；ID 唯一化；HTTP 超时统一；特征测试先行。

## 七、与 atlhyper 当前工作的排期关系

- 当前 atlhyper 无进行中的编码任务；待办均为等用户验证 / 决定的事项（OIDC 浏览器验证、dev 密钥轮换、prod 上线）。
- 本计划与上述事项**无代码冲突**（权限模型 / OIDC 不碰 `ai/`）。唯一交叉：`ai/chat` 的权限 `ai.chat` 已在权限模型中拆出，不受影响。
- PR-A 预计：后端 + 前端 + 迁移 + 测试一期完成；上 dev 后需用户在 Web 填 key 做逐模型冒烟。PR-B 在 PR-A 冒烟通过后开始。
- 开工时机：用户在 atlhyper 侧决定。
