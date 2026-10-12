# 硬件健康检测 设计文档

> 状态：第 1 期代码完成（分支 `fix/hardware-health-judgement`，未推送）；第 2 期起待逐项批准
> 来源：两会话研讨定稿（2026-10-12），带交互原型的设计稿在仓库外，不入库

## 背景 + 目标

硬件健康页（`GET /api/v2/observe/metrics/hardware` → 指标页的 7 张摘要卡 + 10 列矩阵 + 节点展开详情）
目前有三类问题：

1. **判色取「最热」而不是「最危险」**：同一类传感器的健康线可以相差 30°C 以上（机械盘 50 / NVMe 84），
   超线的机械盘会藏在更热但正常的 NVMe 后面，格子、overall、速览 tile 一起显示绿色。
2. **没有盘的身份与 SMART**：只看温度，看不到坏扇区、寿命、CRC；也分不清哪块是机械盘
   （`drivetemp` 同时覆盖机械盘与 SATA SSD）。
3. **只有瞬时值**：Agent 只取最近 2 分钟最新值，没有峰值、持续时长、计数器增量。

**约束（用户裁定）**：⛔ 不改现有布局 —— 外层页面、7 张摘要卡、10 列矩阵 + 综合一律不动；
新增内容只进节点展开详情。矩阵「磁盘温度」格只显示状态最差的一块。

## 核心架构

```
node-exporter ─┐
smartctl_exporter (第 2 期) ─┼─> OTel Collector (白名单) ─> ClickHouse ─> Agent 查询 ─> 快照 ─> Master 判定 ─> Web 渲染
                              │
         判定全部在 Master：Agent 只报原始读数，前端只按 status 着色（大后端小前端）
```

阈值来源优先级（`metrics_thresholds.go`）：传感器自报 max/crit > 硬件画像表 > 通用保守值。
⚠️ 第 2 期起对机械盘例外（见功能三）：自报值是厂商工作上限，不是健康线。

## 功能一：温度判定「先最差、同档最热」 — [✅ 第 1 期]

**用户故事**：机械盘超过自己的健康线时，矩阵格、综合列、速览 tile 必须变色，不能被更热但正常的盘盖住。

**规则**：候选集合里先按 status 取最差（crit > warn > good），同档再取温度最高者。

| 位置 | 范围 | 改动 |
|---|---|---|
| `tempCell`（metrics_hardware.go） | 单节点、同一类传感器 → 矩阵三列温度格 | 逐个判定后按规则挑选 |
| `pickMaxTemp`（同上） | 全集群、跨类别 → 速览「最高温度」「最高盘温」tile | 同规则 |
| `moreConcerning` / `statusRank` | 新增 | 两处共用的比较函数 |

**不改的一处**：Agent 侧 `fillTemperature` 的 `hottestCPU`（agent metrics.go）。
它产出的是读数事实 `CPUTempC`（CPU 类最热核心），不是判定；改成「先最差」需要 Agent 知道阈值，
违反「判定全部在 Master」。CPU 类内各核阈值通常一致，取最热不会掩盖告警。

**测试**（`metrics_hardware_worst_test.go`）：

| 用例 | 断言 |
|---|---|
| 机械盘 55°C（健康线 50）+ NVMe 58°C（84） | 格子 = 机械盘 55 / warn |
| 两块 NVMe 同为 good | 取更热的那块（回归守护） |
| 同上节点走 `buildHardwareRow` | diskTemp 与 overall 均为 warn |
| CPU 74°C good + 另一节点机械盘 55°C warn | tile 指向机械盘 |

**待定（用户）**：速览 tile 文案「最高温度 / 最高盘温」在新规则下语义变为「最需关注的温度点」。
数值仍是真实读数，但不一定是全集群最高的那个。候选：A 保留文案（颜色说明它为何被选中）；
B 改为「最需关注温度」。属文案修改，不动布局。

## 功能二：只自报 max 时补对称 crit — [✅ 第 1 期]

**问题**：传感器只报 max 不报 crit 时，旧逻辑直接套画像 crit。自报 max 高于画像 crit 时
（实测某网卡 PHY 自报 max=120，其他档 crit=95）得到 120/95，温度越过 95 就跳过 warn 直接 crit。

**规则**：`crit = max(画像 crit, max ÷ 0.9)`，与既有的「只报 crit → max = crit × 0.9」对称。

| 输入（自报 max / crit，画像 max / crit） | 旧 | 新 |
|---|---|---|
| 120 / 0，85 / 95 | 120 / 95 ❌ | 120 / 133.33 |
| 70 / 0，85 / 95 | 70 / 95 | 70 / 95（不变） |
| 84 / 89，— | 84 / 89 | 不变 |
| 0 / 100，— | 90 / 100 | 不变 |

测试：`TestResolveTempLimits_MaxOnlyKeepsWarnBand`（表驱动，另断言 crit > max 恒成立）。

## 功能三：机械盘健康线与盘身份 — [待实现，依赖第 2 期 smartctl_exporter]

- 机械盘显式阈值档 50 / 60；自报 max/crit 只作「规格上限」展示，不参与判色。
  NVMe、CPU 仍是自报优先（自报 = 降频点）。
- 前提是知道哪块是机械盘：现部署的 node-exporter 未挂 `/run/udev`，`node_disk_info` 的 model/serial 全空，
  也没有转速。⛔ 不能拿「chip_name = drivetemp」或「有自报阈值」当判据（SATA SSD 也是 drivetemp；
  有没有自报阈值取决于固件）。
- smartctl_exporter 作为盘的唯一事实来源：温度、型号、转速、SMART 同出一份，按 device 天然对齐；
  `drivetemp` 的 chip 是 SCSI H:C:T:L，与 sdX 无可关联的标签，降为兜底。
- 矩阵格标签「sda · 型号」；休眠盘为独立状态「休眠」（中性色，排序同 good），⛔ 不显示无数据或绿色。

## 功能四：SMART 判定 — [待实现，依赖第 2 期]

借鉴 Backblaze 故障率研究的判定形状：

| 属性 | 黄 | 红 |
|---|---|---|
| 197 待处理扇区 / 198 离线不可修复 | — | > 0 |
| 5 重映射 / 187 报告不可修复 / 188 命令超时 | > 0 | 24h 增量 > 0 |
| 199 UDMA CRC（多为数据线） | — | 24h 增量 > 0（⛔ 不看绝对值） |
| NVMe critical_warning / media_errors | — | ≠ 0 / > 0 |
| NVMe 寿命已用 / 备用块 | ≥ 80% | 备用块 < 阈值 |
| SMART 整体自评 | — | FAILED |

存量稳定的重映射盘很常见，Backblaze 的结论是「相关」而非「将坏」⇒ 存量判黄、增量判红，避免告警疲劳。

## 功能五：趋势与持续时长 — [待实现]

- 「持续 ≥ 15 分钟才升档」放 **Master 内存状态机**（快照 10 秒一到，per-sensor 记首次越线时间）。
  ⛔ 不每周期查 ClickHouse —— 周期性重复查询已造成过一次 ClickHouse 负载问题。
- 24h 峰值与计数器增量用一条低频聚合（≥ 5 分钟 + 缓存）。⛔ 不采 `node_hwmon_temp_highest_celsius`。

## 功能六：节点展开详情 v2 — [待实现，依赖第 2 期采集面]

信息架构「结论优先 + USE 资源 + 物理盘卡 + 存储 / 网卡 + 折叠的系统区」。待定阈值（经验值，待 7 天数据校准）：

| 信号 | 黄 / 红 | 出处 |
|---|---|---|
| 负载 / 核 | 2 / 4，持续 15m | node-mixin `systemSaturationPerCoreThreshold=2` |
| 主缺页 | 500/s | node-mixin `memoryMajorPagesFaultsThreshold=500` |
| PSI CPU some | 25 / 50 % | 经验值 |
| PSI 内存 some / full | 10 / 25 %，5 / 10 % | 经验值 |
| PSI IO some / full | 30 / 60 %，10 / 25 % | 经验值 |
| TCP 重传率 | 1 / 5 %，OutSegs < 100/s 不判色 | 经验值；需扩白名单 |

## 实施阶段

| 期 | 内容 | 部署 | 状态 |
|---|---|---|---|
| 0 | 设计文档 + tracker；7 天温度 / PSI 分布（需批准只读通道） | 无 | 文档 ✅ |
| 1 | 功能一、二 | 仅 Master | ✅ 代码 + 测试，待推送 |
| 2-A | Collector 白名单扩约 20 项（与 `metrics_catalog.go` 同步） | Collector | 待批准 |
| 2-B | node-exporter：systemd（只看 failed）、textfile、可选 `/run/udev` | DaemonSet | 待批准 |
| 2-C | smartctl_exporter DaemonSet（间隔 ≥ 10m、standby 不唤醒） | 新组件 | 待批准 |
| 3 | 功能三、四、五、六 | Agent + Master + Web | 待 2 期 |

每期发布：push（批准）→ CI → dev 验证（批准）→ prod（批准）。
⚠️ prod 的 ArgoCD 应用尚未完成首次接管，首次 SYNC 需单独决策（见 tracker「CI/CD 接入」）。

## 文件变更清单

```
atlhyper_master_v2/service/query/
├── metrics_hardware.go              [✅ 修改] tempCell / pickMaxTemp 先最差；resolveTempLimits 对称 crit；新增 moreConcerning / statusRank
└── metrics_hardware_worst_test.go   [✅ 新增] 功能一、二的测试
docs/design/active/hardware-health-design.md  [✅ 新增] 本文件
```

## 附录：待用户裁定项

D1 drivetemp 去留（机械盘可能因每 15 秒读温度而不再休眠）· D2 引入 smartctl_exporter ·
D3 转速来源 · D5 树莓派降频位 · D6 内核 I/O 报错（journald）· D7 hw-check 与 atlhyper 阈值差异 ·
D8 两边阈值同步机制 · 速览 tile 文案（功能一）· prod 首次 SYNC。
