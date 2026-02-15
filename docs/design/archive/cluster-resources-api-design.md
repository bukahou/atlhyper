# 8 个 K8s 资源后端 API 实装 + 前端对接

## Context

前端已完成 8 个资源的 mock 页面（Job, CronJob, PV, PVC, NetworkPolicy, ResourceQuota, LimitRange, ServiceAccount）。Agent 已采集全部 8 个资源并通过 ClusterSnapshot 推送给 Master。现在需要：在 Master 侧实装 API → 前端切换到真实数据。

遵循 TDD 流程和大后端小前端规范。

---

## 数据源确认（Agent → Master）

Agent 通过 `model_v2.ClusterSnapshot` 上报的 8 个资源：

| 资源 | model_v2 位置 | snapshot 字段 | 结构特点 |
|------|--------------|---------------|---------|
| Job | `model_v2/job.go` | `snapshot.Jobs` | CommonMeta + Active/Succeeded/Failed/Complete/StartTime/FinishTime |
| CronJob | `model_v2/job.go` | `snapshot.CronJobs` | CommonMeta + Schedule/Suspend/ActiveJobs/LastScheduleTime/LastSuccessfulTime |
| PV | `model_v2/storage.go` | `snapshot.PersistentVolumes` | CommonMeta + Capacity/Phase/StorageClass/AccessModes/ReclaimPolicy（集群级，无 namespace） |
| PVC | `model_v2/storage.go` | `snapshot.PersistentVolumeClaims` | CommonMeta + Phase/VolumeName/StorageClass/AccessModes/RequestedCapacity/ActualCapacity |
| NetworkPolicy | `model_v2/policy.go` | `snapshot.NetworkPolicies` | 扁平结构（已 camelCase）：Name/Namespace/PolicyTypes/IngressRuleCount/EgressRuleCount |
| ResourceQuota | `model_v2/policy.go` | `snapshot.ResourceQuotas` | 扁平结构（已 camelCase）：Name/Namespace/Hard/Used/Scopes |
| LimitRange | `model_v2/policy.go` | `snapshot.LimitRanges` | 扁平结构（已 camelCase）：Name/Namespace/Items |
| ServiceAccount | `model_v2/policy.go` | `snapshot.ServiceAccounts` | 扁平结构（已 camelCase）：Name/Namespace/SecretsCount/ImagePullSecretsCount/AutomountServiceAccountToken |

**注意**：前端 mock 数据包含一些 model_v2 中不存在的字段（如 Job 的 `completions`/`parallelism`/`duration`，PV 的 `claimRef`/`volumeMode`）。API 只返回 Agent 实际采集的字段，前端页面需要相应调整。

---

## 实现流程（TDD）

### Step 1: 定义 API 响应类型 — `model/`

参考模式：`atlhyper_master_v2/model/daemonset.go`

**CommonMeta 资源**（需要 snake_case → camelCase 扁平化）:

- `model/job.go` — `JobItem`：Name/Namespace/Active/Succeeded/Failed/Complete/StartTime/FinishTime/CreatedAt/Age
- `model/cronjob.go` — `CronJobItem`：Name/Namespace/Schedule/Suspend/ActiveJobs/LastScheduleTime/LastSuccessfulTime/CreatedAt/Age
- `model/pv.go` — `PVItem`：Name/Capacity/Phase/StorageClass/AccessModes/ReclaimPolicy/CreatedAt/Age（无 Namespace）
- `model/pvc.go` — `PVCItem`：Name/Namespace/Phase/VolumeName/StorageClass/AccessModes/RequestedCapacity/ActualCapacity/CreatedAt/Age

**扁平结构资源**（model_v2 已是 camelCase，创建 model 类型保持一致性和解耦）:

- `model/network_policy.go` — `NetworkPolicyItem`（字段与 model_v2 一致，直接映射）
- `model/resource_quota.go` — `ResourceQuotaItem`（字段与 model_v2 一致）
- `model/limit_range.go` — `LimitRangeItem` + `LimitRangeItemEntry`（字段与 model_v2 一致）
- `model/service_account.go` — `ServiceAccountItem`（字段与 model_v2 一致）

### Step 2: TDD — 先写 convert 测试（RED）

参考模式：`atlhyper_master_v2/model/convert/daemonset_test.go`

为 8 个资源各写一个测试文件，每个包含：
- `Test<Resource>Item_FieldMapping` — 正常路径字段映射
- `Test<Resource>Items_NilInput` — nil 输入返回空切片
- `Test<Resource>Items_EmptyInput` — 空切片输入返回空切片

运行测试确认全 RED（编译失败）。

### Step 3: 实现 convert 函数（GREEN）

参考模式：`atlhyper_master_v2/model/convert/pod.go`

为 8 个资源各写：`XxxItem(*model_v2.Xxx) model.XxxItem` + `XxxItems([]model_v2.Xxx) []model.XxxItem`

CommonMeta 资源的转换逻辑：
- `Name` ← `src.GetName()`
- `Namespace` ← `src.GetNamespace()`
- `CreatedAt` ← `src.CreatedAt.Format("2006-01-02T15:04:05Z07:00")`
- 时间指针 ← nil 检查后格式化，nil 返回 `""`

扁平结构资源：直接字段赋值。

运行测试确认全 GREEN。

### Step 4: Service 层 — 添加查询方法

**`service/interfaces.go`** — Query 接口新增 8 个方法：

```go
GetJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.Job, error)
GetCronJobs(ctx context.Context, clusterID string, namespace string) ([]model_v2.CronJob, error)
GetPersistentVolumes(ctx context.Context, clusterID string) ([]model_v2.PersistentVolume, error)
GetPersistentVolumeClaims(ctx context.Context, clusterID string, namespace string) ([]model_v2.PersistentVolumeClaim, error)
GetNetworkPolicies(ctx context.Context, clusterID string, namespace string) ([]model_v2.NetworkPolicy, error)
GetResourceQuotas(ctx context.Context, clusterID string, namespace string) ([]model_v2.ResourceQuota, error)
GetLimitRanges(ctx context.Context, clusterID string, namespace string) ([]model_v2.LimitRange, error)
GetServiceAccounts(ctx context.Context, clusterID string, namespace string) ([]model_v2.ServiceAccount, error)
```

**`service/query/impl.go`** — 实现 8 个查询方法（参考 `GetDaemonSets`：snapshot 获取 → namespace 过滤 → 返回）。PV 无 namespace 参数。

### Step 5: Handler + 路由

参考模式：`atlhyper_master_v2/gateway/handler/daemonset.go`

8 个 handler 文件（只实现 List，暂不需要 Detail）：

| Handler | 路由 | 特点 |
|---------|------|------|
| `handler/job.go` | `/api/v2/jobs` | cluster_id + namespace |
| `handler/cronjob.go` | `/api/v2/cronjobs` | cluster_id + namespace |
| `handler/pv.go` | `/api/v2/pvs` | cluster_id（无 namespace） |
| `handler/pvc.go` | `/api/v2/pvcs` | cluster_id + namespace |
| `handler/network_policy.go` | `/api/v2/network-policies` | cluster_id + namespace |
| `handler/resource_quota.go` | `/api/v2/resource-quotas` | cluster_id + namespace |
| `handler/limit_range.go` | `/api/v2/limit-ranges` | cluster_id + namespace |
| `handler/service_account.go` | `/api/v2/service-accounts` | cluster_id + namespace |

每个 Handler 流程：获取参数 → `svc.GetXxx()` → `convert.XxxItems()` → `writeJSON()`

**`gateway/routes.go`** — 在公开路由部分注册 8 条路由。

### Step 6: 后端验证

```bash
go test ./atlhyper_master_v2/model/convert/... -v
go build ./atlhyper_master_v2/...
```

### Step 7: 前端对接

**7a. `api/cluster-resources.ts`** — 切换到真实 API 调用（使用 `get<>()` 请求）

**7b. 更新前端类型和 mock 类型对齐 API 响应字段**：
- Job: 移除 mock-only 字段（`completions`/`parallelism`/`duration`/`completionTime`），添加 `complete`/`finishTime`
- PV: 移除 `claimRef`/`volumeMode`，`status` → `phase`
- 策略组: `ingressRules` → `ingressRuleCount` 等

**7c. 更新 8 个 page.tsx** — 调整表格列和统计卡片的字段名匹配 API

**7d. 删除 mock** — `api/mock/cluster-resources.ts`

### Step 8: 前端验证

```bash
cd atlhyper_web && npx next build
```

---

## 文件变更结构

```
atlhyper/
├── atlhyper_master_v2/
│   ├── model/                              # API 响应类型（camelCase JSON）
│   │   ├── job.go                          # [新建] JobItem
│   │   ├── cronjob.go                      # [新建] CronJobItem
│   │   ├── pv.go                           # [新建] PVItem
│   │   ├── pvc.go                          # [新建] PVCItem
│   │   ├── network_policy.go               # [新建] NetworkPolicyItem
│   │   ├── resource_quota.go               # [新建] ResourceQuotaItem
│   │   ├── limit_range.go                  # [新建] LimitRangeItem + LimitRangeItemEntry
│   │   ├── service_account.go              # [新建] ServiceAccountItem
│   │   └── convert/                        # model_v2 → model 转换
│   │       ├── job.go                      # [新建] JobItem() + JobItems()
│   │       ├── job_test.go                 # [新建] TDD 测试
│   │       ├── cronjob.go                  # [新建] CronJobItem() + CronJobItems()
│   │       ├── cronjob_test.go             # [新建] TDD 测试
│   │       ├── pv.go                       # [新建] PVItem() + PVItems()
│   │       ├── pv_test.go                  # [新建] TDD 测试
│   │       ├── pvc.go                      # [新建] PVCItem() + PVCItems()
│   │       ├── pvc_test.go                 # [新建] TDD 测试
│   │       ├── network_policy.go           # [新建] NetworkPolicyItem() + NetworkPolicyItems()
│   │       ├── network_policy_test.go      # [新建] TDD 测试
│   │       ├── resource_quota.go           # [新建] ResourceQuotaItem() + ResourceQuotaItems()
│   │       ├── resource_quota_test.go      # [新建] TDD 测试
│   │       ├── limit_range.go              # [新建] LimitRangeItem() + LimitRangeItems()
│   │       ├── limit_range_test.go         # [新建] TDD 测试
│   │       ├── service_account.go          # [新建] ServiceAccountItem() + ServiceAccountItems()
│   │       └── service_account_test.go     # [新建] TDD 测试
│   ├── service/
│   │   ├── interfaces.go                   # [修改] Query 接口新增 8 个方法签名
│   │   └── query/
│   │       └── impl.go                     # [修改] 实现 8 个查询方法
│   └── gateway/
│       ├── routes.go                       # [修改] 注册 8 条公开路由
│       └── handler/
│           ├── job.go                      # [新建] JobHandler.List
│           ├── cronjob.go                  # [新建] CronJobHandler.List
│           ├── pv.go                       # [新建] PVHandler.List（无 namespace）
│           ├── pvc.go                      # [新建] PVCHandler.List
│           ├── network_policy.go           # [新建] NetworkPolicyHandler.List
│           ├── resource_quota.go           # [新建] ResourceQuotaHandler.List
│           ├── limit_range.go              # [新建] LimitRangeHandler.List
│           └── service_account.go          # [新建] ServiceAccountHandler.List
│
└── atlhyper_web/src/
    ├── api/
    │   ├── cluster-resources.ts            # [修改] mock → 真实 API 调用
    │   └── mock/
    │       └── cluster-resources.ts        # [删除] mock 数据移除
    └── app/cluster/
        ├── job/page.tsx                    # [修改] 字段名匹配 API
        ├── cronjob/page.tsx                # [修改] 字段名匹配 API
        ├── pv/page.tsx                     # [修改] 字段名匹配 API
        ├── pvc/page.tsx                    # [修改] 字段名匹配 API
        ├── network-policy/page.tsx         # [修改] 字段名匹配 API
        ├── resource-quota/page.tsx         # [修改] 字段名匹配 API
        ├── limit-range/page.tsx            # [修改] 字段名匹配 API
        └── service-account/page.tsx        # [修改] 字段名匹配 API
```

**统计**: 新建 32 个文件 / 修改 11 个文件 / 删除 1 个文件
