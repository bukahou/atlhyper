# 8 个 K8s 资源 API + 详情扩展 — 已完成

> 设计文档: [cluster-resources-api-design.md](../../design/archive/cluster-resources-api-design.md)

## 资源列表

Job, CronJob, PV, PVC, NetworkPolicy, ResourceQuota, LimitRange, ServiceAccount

## 完成内容

### Phase 1: 后端 API 实装 + 前端对接（commit da6bf9b）

- model_v2 共享模型（Agent 已采集）
- Master model/convert/handler/routes — 8 个资源 List API
- 前端 mock 替换为真实 API 调用

### Phase 2: 详情弹窗基础版（commit 626714d）

- 8 个资源新增 Detail API（Master handler Get 方法）
- 8 个 DetailModal 前端组件（简单概览+标签）

### Phase 3: 详情扩展（commit 51dbdc1）

- model_v2 扩展: Job/CronJob +PodTemplate/Conditions/Spec, PV +VolumeSource/ClaimRef, NP +完整规则, SA +Secret 名称, 全部 +Labels/Annotations
- Agent converter 扩展: 提取新字段（convertPodTemplate 复用）
- Master model/convert 扩展: 8 个 Detail 类型全部丰富
- 前端: Job/CronJob 3 Tab（概览+容器+标签），NP 3 Tab（概览+规则+标签），其余 2 Tab（概览+标签）
- i18n 完成
