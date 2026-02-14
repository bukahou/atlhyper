# 节点指标迁移 Phase 1：基础设施部署与验证

> 状态：设计中
> 创建：2026-02-11
> 前置：无
> 后续：`node-metrics-phase2-agent.md`, `node-metrics-phase3-master.md`

## 1. 概要

部署 **node_exporter DaemonSet** 并修改 **OTel Collector ConfigMap**，使 OTel Collector `:8889/metrics` 端点输出节点硬件指标。

完成后 Agent 即可从单一端点拉取 SLO + 节点指标。

## 2. 背景

| 项目 | 当前 | 目标 |
|------|------|------|
| 节点指标采集 | atlhyper_metrics_v2（自建 DaemonSet，读 /proc） | node_exporter（业界标准） |
| 数据传输 | HTTP POST → Agent ReceiverClient | OTel Collector scrape → Agent OTelClient pull |
| 部署位置 | atlhyper namespace（6 Pod） | otel namespace（6 Pod） |

## 3. 集群环境

| 节点 | IP | 角色 | 架构 |
|------|-----|------|------|
| desk-zero | 192.168.0.130 | control-plane | amd64 |
| desk-one | 192.168.0.7 | worker | amd64 |
| desk-two | 192.168.0.46 | worker | amd64 |
| raspi-zero | 192.168.0.182 | worker | arm64 |
| raspi-one | 192.168.0.33 | worker | arm64 |
| raspi-nfs | 192.168.0.153 | worker | arm64 |

node_exporter 官方镜像 `prom/node-exporter` 已支持 multi-arch（amd64 + arm64）。

## 4. 配置文件位置

所有文件放在 OTel 目录下统一管理：

```
~/AtlHyper/GitHub/Config/zgmf-x10a/k8s-configs/otel/
├── configmap.yaml          ← 修改: 增加 node-exporter scrape config
├── node-exporter.yaml      ← NEW:  DaemonSet + Service
├── otel-collector.yaml     (现有)
├── kustomization.yaml      (现有) ← 修改: 加入 node-exporter.yaml
└── README.md               (现有)
```

## 5. node_exporter DaemonSet

### 5.1 资源清单 (`node-exporter.yaml`)

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: node-exporter
  namespace: otel
  labels:
    app: node-exporter
spec:
  selector:
    matchLabels:
      app: node-exporter
  template:
    metadata:
      labels:
        app: node-exporter
    spec:
      hostNetwork: true
      hostPID: true
      containers:
        - name: node-exporter
          image: prom/node-exporter:v1.9.0
          args:
            - '--path.procfs=/host/proc'
            - '--path.sysfs=/host/sys'
            - '--path.rootfs=/host/root'
            - '--collector.filesystem.mount-points-exclude=^/(dev|proc|sys|var/lib/docker/.+|var/lib/kubelet/.+)($$|/)'
            - '--collector.netclass.ignored-devices=^(veth.*|cali.*|flannel.*|cni.*)$$'
          ports:
            - containerPort: 9100
              hostPort: 9100
              name: metrics
          resources:
            requests:
              cpu: 50m
              memory: 30Mi
            limits:
              cpu: 200m
              memory: 100Mi
          volumeMounts:
            - name: proc
              mountPath: /host/proc
              readOnly: true
            - name: sys
              mountPath: /host/sys
              readOnly: true
            - name: root
              mountPath: /host/root
              mountPropagation: HostToContainer
              readOnly: true
      volumes:
        - name: proc
          hostPath:
            path: /proc
        - name: sys
          hostPath:
            path: /sys
        - name: root
          hostPath:
            path: /
      tolerations:
        - effect: NoSchedule
          operator: Exists
---
apiVersion: v1
kind: Service
metadata:
  name: node-exporter
  namespace: otel
  labels:
    app: node-exporter
spec:
  clusterIP: None
  ports:
    - port: 9100
      name: metrics
  selector:
    app: node-exporter
```

### 5.2 关键配置说明

| 配置 | 说明 |
|------|------|
| `hostNetwork: true` | 直接监听宿主机 IP:9100，OTel 用节点 IP scrape |
| `hostPID: true` | 需要访问宿主机进程信息 |
| `/host/proc`, `/host/sys`, `/host/root` | 挂载宿主机文件系统 |
| `filesystem.mount-points-exclude` | 过滤虚拟/容器文件系统 |
| `netclass.ignored-devices` | 过滤 veth/cni 等虚拟网卡 |
| `tolerations: NoSchedule` | 确保 control-plane 节点也部署 |

## 6. OTel Collector ConfigMap 修改

### 6.1 新增 scrape_config

在 `prometheus.config.scrape_configs` 末尾新增 `node-exporter` job：

```yaml
# 节点硬件指标 (node_exporter)
- job_name: 'node-exporter'
  scrape_interval: 15s
  static_configs:
    - targets:
      - '192.168.0.130:9100'  # desk-zero
      - '192.168.0.7:9100'    # desk-one
      - '192.168.0.46:9100'   # desk-two
      - '192.168.0.182:9100'  # raspi-zero
      - '192.168.0.33:9100'   # raspi-one
      - '192.168.0.153:9100'  # raspi-nfs
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: '(node_cpu_seconds_total|node_cpu_info|node_cpu_scaling_frequency_hertz|node_load1|node_load5|node_load15|node_memory_MemTotal_bytes|node_memory_MemAvailable_bytes|node_memory_MemFree_bytes|node_memory_Cached_bytes|node_memory_Buffers_bytes|node_memory_SwapTotal_bytes|node_memory_SwapFree_bytes|node_filesystem_size_bytes|node_filesystem_avail_bytes|node_disk_read_bytes_total|node_disk_written_bytes_total|node_disk_reads_completed_total|node_disk_writes_completed_total|node_disk_io_time_seconds_total|node_network_receive_bytes_total|node_network_transmit_bytes_total|node_network_receive_packets_total|node_network_transmit_packets_total|node_network_receive_errs_total|node_network_transmit_errs_total|node_network_receive_drop_total|node_network_transmit_drop_total|node_network_mtu_bytes|node_network_speed_bytes|node_network_up|node_hwmon_temp_celsius|node_hwmon_temp_max_celsius|node_uname_info|node_boot_time_seconds)'
      action: keep
```

### 6.2 设计选择

**使用 `static_configs` 而非 `kubernetes_sd_configs`**：
- 与现有 `traefik` job 风格一致
- 不需要额外 RBAC 权限
- 节点固定（6 个），维护成本低

**`scrape_interval: 15s`**：
- SLO 指标用 10s（高频）
- 节点指标用 15s（硬件变化慢，减少开销）

**`metric_relabel_configs` 白名单过滤**：
- node_exporter 默认输出数千个指标
- 只保留 `NodeMetricsSnapshot` 需要的 ~35 个指标系列
- 大幅减少 OTel Collector 内存和 Agent 解析开销

### 6.3 保留指标清单

| 类别 | 指标 | 类型 | 用途 |
|------|------|------|------|
| **CPU** | `node_cpu_seconds_total` | counter | 使用率（按 mode/cpu 分组） |
| | `node_cpu_info` | gauge | CPU 型号（label） |
| | `node_cpu_scaling_frequency_hertz` | gauge | 频率 |
| | `node_load1/5/15` | gauge | 负载 |
| **内存** | `node_memory_MemTotal_bytes` | gauge | 总内存 |
| | `node_memory_MemAvailable_bytes` | gauge | 可用内存 |
| | `node_memory_MemFree_bytes` | gauge | 空闲内存 |
| | `node_memory_Cached_bytes` | gauge | 页面缓存 |
| | `node_memory_Buffers_bytes` | gauge | 缓冲区 |
| | `node_memory_SwapTotal_bytes` | gauge | Swap 总量 |
| | `node_memory_SwapFree_bytes` | gauge | Swap 空闲 |
| **磁盘** | `node_filesystem_size_bytes` | gauge | 分区总量 |
| | `node_filesystem_avail_bytes` | gauge | 分区可用 |
| | `node_disk_read_bytes_total` | counter | 读取字节 |
| | `node_disk_written_bytes_total` | counter | 写入字节 |
| | `node_disk_reads_completed_total` | counter | 读 IOPS |
| | `node_disk_writes_completed_total` | counter | 写 IOPS |
| | `node_disk_io_time_seconds_total` | counter | I/O 利用率 |
| **网络** | `node_network_receive_bytes_total` | counter | 接收字节 |
| | `node_network_transmit_bytes_total` | counter | 发送字节 |
| | `node_network_receive_packets_total` | counter | 接收包 |
| | `node_network_transmit_packets_total` | counter | 发送包 |
| | `node_network_receive_errs_total` | counter | 接收错误 |
| | `node_network_transmit_errs_total` | counter | 发送错误 |
| | `node_network_receive_drop_total` | counter | 接收丢包 |
| | `node_network_transmit_drop_total` | counter | 发送丢包 |
| | `node_network_mtu_bytes` | gauge | MTU |
| | `node_network_speed_bytes` | gauge | 链路速度 |
| | `node_network_up` | gauge | 接口状态 |
| **温度** | `node_hwmon_temp_celsius` | gauge | 传感器温度 |
| | `node_hwmon_temp_max_celsius` | gauge | 温度上限 |
| **系统** | `node_uname_info` | gauge | OS/Kernel（label） |
| | `node_boot_time_seconds` | gauge | 启动时间 |

### 6.4 完整 configmap.yaml（修改后）

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: otel-collector-config
  namespace: otel
data:
  config.yaml: |
    receivers:
      otlp:
        protocols:
          grpc:
            endpoint: 0.0.0.0:4317
          http:
            endpoint: 0.0.0.0:4318

      prometheus:
        config:
          scrape_configs:
            # Linkerd 服务网格指标
            - job_name: 'linkerd-prometheus'
              scrape_interval: 10s
              static_configs:
                - targets: ['prometheus.linkerd-viz.svc.cluster.local:9090']
              metrics_path: /federate
              params:
                'match[]':
                  - '{job=~"linkerd.*"}'
                  - '{job="kubernetes-nodes-cadvisor"}'
              metric_relabel_configs:
                - source_labels: [__name__]
                  regex: '(request_total|response_total|response_latency_ms_bucket|response_latency_ms_count|response_latency_ms_sum|tcp_open_connections|tcp_read_bytes_total|tcp_write_bytes_total|container_cpu_usage_seconds_total|container_memory_working_set_bytes|container_network_receive_bytes_total|container_network_transmit_bytes_total)'
                  action: keep

            # Traefik Ingress 指标
            - job_name: 'traefik'
              scrape_interval: 10s
              static_configs:
                - targets: ['traefik-metrics.kube-system.svc.cluster.local:9100']
              metric_relabel_configs:
                - source_labels: [__name__]
                  regex: 'traefik_(entrypoint|service|router)_.*'
                  action: keep

            # 节点硬件指标 (node_exporter)
            - job_name: 'node-exporter'
              scrape_interval: 15s
              static_configs:
                - targets:
                  - '192.168.0.130:9100'
                  - '192.168.0.7:9100'
                  - '192.168.0.46:9100'
                  - '192.168.0.182:9100'
                  - '192.168.0.33:9100'
                  - '192.168.0.153:9100'
              metric_relabel_configs:
                - source_labels: [__name__]
                  regex: '(node_cpu_seconds_total|node_cpu_info|node_cpu_scaling_frequency_hertz|node_load1|node_load5|node_load15|node_memory_MemTotal_bytes|node_memory_MemAvailable_bytes|node_memory_MemFree_bytes|node_memory_Cached_bytes|node_memory_Buffers_bytes|node_memory_SwapTotal_bytes|node_memory_SwapFree_bytes|node_filesystem_size_bytes|node_filesystem_avail_bytes|node_disk_read_bytes_total|node_disk_written_bytes_total|node_disk_reads_completed_total|node_disk_writes_completed_total|node_disk_io_time_seconds_total|node_network_receive_bytes_total|node_network_transmit_bytes_total|node_network_receive_packets_total|node_network_transmit_packets_total|node_network_receive_errs_total|node_network_transmit_errs_total|node_network_receive_drop_total|node_network_transmit_drop_total|node_network_mtu_bytes|node_network_speed_bytes|node_network_up|node_hwmon_temp_celsius|node_hwmon_temp_max_celsius|node_uname_info|node_boot_time_seconds)'
                  action: keep

    processors:
      memory_limiter:
        check_interval: 1s
        limit_mib: 900
        spike_limit_mib: 200

      resource:
        attributes:
          - key: cluster.name
            value: zgmf-x10a
            action: upsert

    exporters:
      debug:
        verbosity: basic

      prometheus:
        endpoint: 0.0.0.0:8889
        namespace: otel

    extensions:
      health_check:
        endpoint: 0.0.0.0:13133

    service:
      extensions: [health_check]

      pipelines:
        traces:
          receivers: [otlp]
          processors: [memory_limiter, resource]
          exporters: [debug]

        metrics:
          receivers: [otlp, prometheus]
          processors: [memory_limiter, resource]
          exporters: [debug, prometheus]

      telemetry:
        logs:
          level: info
        metrics:
          address: 0.0.0.0:8888
```

## 7. 部署步骤

```bash
# 1. 部署 node_exporter
kubectl apply -f k8s-configs/otel/node-exporter.yaml

# 2. 确认所有节点运行
kubectl -n otel get pods -l app=node-exporter -o wide

# 3. 直接验证 node_exporter 端点
curl -s 192.168.0.130:9100/metrics | grep node_cpu_seconds_total | head -5

# 4. 更新 OTel Collector ConfigMap
kubectl apply -f k8s-configs/otel/configmap.yaml

# 5. 重启 OTel Collector 使配置生效
kubectl -n otel rollout restart deployment otel-collector

# 6. 等待就绪
kubectl -n otel rollout status deployment otel-collector
```

## 8. 验证

### 8.1 node_exporter 端点验证

```bash
# 每个节点都能访问
for ip in 192.168.0.130 192.168.0.7 192.168.0.46 192.168.0.182 192.168.0.33 192.168.0.153; do
  echo "=== $ip ==="
  curl -s --connect-timeout 3 $ip:9100/metrics | grep -c "^node_"
done
```

### 8.2 OTel Collector 端点验证

```bash
# 使用临时 Pod 验证 OTel 输出中包含 node 指标
kubectl run curl-test --rm -it --image=curlimages/curl -- \
  sh -c 'curl -s otel-collector.otel.svc:8889/metrics | grep "otel_node_" | head -20'
```

### 8.3 关键指标检查清单

| 指标 | 验证命令片段 | 预期 |
|------|-------------|------|
| CPU | `grep otel_node_cpu_seconds_total` | 每节点 × 每核 × 8 mode |
| 内存 | `grep otel_node_memory_MemTotal` | 6 个节点 |
| 磁盘 | `grep otel_node_filesystem_size` | 每节点至少 1 个分区 |
| 网络 | `grep otel_node_network_receive_bytes` | 每节点至少 1 个接口 |
| 温度 | `grep otel_node_hwmon_temp` | amd64 节点有，raspi 视硬件 |
| 系统信息 | `grep otel_node_uname_info` | 6 条，含 nodename label |

### 8.4 instance → 节点名映射验证

```bash
# 确认 instance label 包含节点 IP，uname_info 包含 nodename
kubectl run curl-test --rm -it --image=curlimages/curl -- \
  sh -c 'curl -s otel-collector.otel.svc:8889/metrics | grep otel_node_uname_info'
```

预期输出格式：
```
otel_node_uname_info{instance="192.168.0.130:9100",nodename="desk-zero",release="6.8.0-85-generic",...} 1
```

Agent 将使用 `nodename` label 作为节点标识。

## 9. 注意事项

- **OTel namespace 前缀**：OTel Collector 的 prometheus exporter 配置了 `namespace: otel`，输出的指标名会加 `otel_` 前缀（如 `otel_node_cpu_seconds_total`）。Agent 解析时需去除此前缀。
- **Traefik 端口冲突**：Traefik metrics 使用 9100 端口（Service 端口），但 node_exporter 用 hostPort 9100。两者不冲突：Traefik 的 9100 是 ClusterIP Service 端口，node_exporter 的 9100 是宿主机端口。
- **内存影响**：node_exporter 每节点约 30-100Mi，OTel Collector 因新增指标可能增加 50-100Mi，当前限制 900Mi 足够。
