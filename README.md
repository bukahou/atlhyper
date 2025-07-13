# 🧠 NeuroController ・ Plugin-Based Kubernetes Anomaly Controller

---

【🔢 Project Overview 】

NeuroController 是一个面向 Kubernetes 集群的轻量级异常检测与多通道告警系统，设计目标是补充 APM 与 Prometheus 之间的告警盲区，提供基于事件的快速响应机制。支持插件化 Watcher、无重复日志持久化、多级告警机制，适用于 Raspberry Pi 等边端环境。

【📊 System Architecture 】

组成模块：

Watcher Plugins：Pod/Deployment/Endpoint 等资源监控

Diagnosis Engine：异常采集、组合、去重

Alert Dispatcher：Slack / Email 多通道告警分类发送

Log Persistence：结构化日志本地写入

UI API Server：REST 接口提供前端查询与操作

### ・Plugin-Based Resource Watcher Framework

- Kubernetes の核心資源毎に個別の Watcher プラグインを提供
- 統一登録と生命周期管理を支援
- `controller-runtime`により従来の同期化イベント構成に対応
- Each core Kubernetes resource is managed by a separate watcher plugin
- All watchers are centrally registered with unified lifecycle management
- Built on `controller-runtime`, supports efficient concurrent event watching

### ・多段階告知モジュール

### ・Multi-Level Alerting System

- 軽量系 (Slack)、固定系、高リスク系 (メール) の 3 レベルの告知機構
- すべての告知は統一的な構造で出力 (`AlertGroupData`)
- 重複排除、レート制限機能を内蔵
- Built-in lightweight (Slack), strict, and high-risk (Email) alert levels
- Unified `AlertGroupData` format for all alerts
- Deduplication and throttling built-in

## 💬 Slack 告知例 / Slack Alert Example

以下は Slack BlockKit を用いた軽量通知の実例です：

![Slack Alert Sample](NeuroController/docs/images/slack.png)

## 📧 メール通知例 / Email Alert Template

システム異常が発生した場合に送信される HTML メール通知の実例：

![Email Alert Sample](NeuroController/docs/images/mail.png)

### ・ログ清潔・持続化メカニズム

### ・Log Cleaning and Persistence

- 異常イベントを Cleaner が調整し、重複や無視可能なログを削除
- 清潔後のログは `/var/log/neurocontroller/cleaned_events.log` にローカル出力
- インタフェース経由の外部分析も支援
- Events are deduplicated and cleaned by a dedicated `Cleaner`
- Logs are persisted at `/var/log/neurocontroller/cleaned_events.log`
- Easy to integrate with external analysis systems

### ・多通信の告知実装

### ・Multi-Channel Alert Notification

- Slack Block Kit 形式の軽量通知を支援
- Email はテンプレートと制限ロジック付き
- 両者は実行時に検知され、同時依存を避ける
- Slack support with Block Kit formatting
- Email alerts with template & rate-limit logic
- Fully independent and concurrent channels

### ・簡潔な Kubernetes 配備

### ・Lightweight Kubernetes Deployment

- `Deployment` + `ClusterRole` + `Binding` により簡単配備
- 使用リソースは極少、Raspberry Pi 環境に有効
- ConfigMap によりはあゆる設定値が管理可能
- Minimal resource usage (below 256Mi / 200m)
- Designed for Raspberry Pi and edge environments
- All thresholds and configs are managed via ConfigMap

---

## 🗋 モジュール一覧 / Module Overview

| パス                   | 機能概要                      |
| ---------------------- | ----------------------------- |
| `cmd/neurocontroller/` | プログラム入り口              |
| `internal/watcher/`    | 資源監視プラグイン            |
| `internal/diagnosis/`  | 異常収集 + 清潔               |
| `internal/alerter/`    | 告知解析・トリガー判定        |
| `external/slack/`      | Slack 通知モジュール          |
| `external/mailer/`     | Email 通知モジュール          |
| `internal/logging/`    | クリーンログ出力              |
| `interfaces/`          | JSON 形式統一インターフェース |
| `config/`              | 告知関連設定                  |

---

## 🚀 情報戦略・適用場面 / Use Cases

- Raspberry Pi / K3s など軽量 K8s の異常監視コントローラ
- Prometheus の代替となるイベント駆動型ログ型告知基盤
- APM システムと連携した統合オブザービリティー
- CI/CD と連携した異常時の自動回復、ロールバック等

---

## 📊 例：構造化ログの出力 / Example: Structured Alert Logs

NeuroController の実行中に記録された構造化告知ログの一部脱敏化サンプルです:
Below is a sample (sanitized) of structured alert logs recorded by NeuroController at runtime:

```json
{
  "category": "Condition",
  "eventTime": "2025-06-09T08:42:05Z",
  "kind": "Pod",
  "message": "Pod 未準備、可能原因未知または未報告",
  "name": "<pod-name>",
  "namespace": "default",
  "reason": "NotReady",
  "severity": "warning",
  "time": "2025-06-09T08:42:20Z"
}
{
  "category": "Warning",
  "eventTime": "2025-06-09T08:42:05Z",
  "kind": "Deployment",
  "message": "Deployment に不可用レプリカが存在、イメージプル失敗やPodクラッシュの可能性",
  "name": "<deployment-name>",
  "namespace": "default",
  "reason": "UnavailableReplica",
  "severity": "info",
  "time": "2025-06-09T08:42:20Z"
}
{
  "category": "Endpoint",
  "eventTime": "2025-06-09T08:42:06Z",
  "kind": "Endpoints",
  "message": "すべてのPodがEndpointsから除外された (利用可能なバックエンドがなし)",
  "name": "<service-name>",
  "namespace": "default",
  "reason": "NoReadyAddress",
  "severity": "critical",
  "time": "2025-06-09T08:42:20Z"
}
```

これらのログは、Pod から Deployment 、Endpoint への告知チェーンを可視化し、根本原因の解析や自動対応シナリオの起点となります。
These logs visualize the alert chain from Pod to Deployment to Endpoint, enabling downstream root cause analysis and triggering of automated response strategies.

# 🕸️ NeuroController 利用ガイド · Usage Guide

---

## ✅ 方法 ①：ローカル開発テスト · Local Development

### 📂 kubeconfig ファイルの取得 · Obtain kubeconfig File

Kubernetes（例：K3s）クラスタから kubeconfig ファイルをエクスポートします（例：`admin-k3s.yaml`）。
Export your kubeconfig from the Kubernetes cluster (e.g., K3s), e.g., `admin-k3s.yaml`.

### 🛠️ 環境変数の設定 · Set Environment Variable

環境変数 `KUBECONFIG` にパスを設定し、コントローラがクラスタへ接続できるようにします：
Set the file path to the `KUBECONFIG` environment variable so the controller can connect to the cluster:

```bash
export KUBECONFIG=/path/to/admin-k3s.yaml
```

### 🚀 コントローラの起動 · Run the Controller

以下のコマンドで NeuroController を直接起動します：
Run NeuroController directly via Go:

```bash
go run ./cmd/neurocontroller/main.go
```

---

## ✅ 方法 ②：公開イメージからのデプロイ · Deploy from Public Image

Docker Hub 上にある公開イメージ `bukahou/neurocontroller:v1.1.0` をそのまま使用してデプロイ可能です。以下は `Deployment` および `ClusterRoleBinding` の完全な例です：
You can deploy directly using the public Docker Hub image `bukahou/neurocontroller:v1.1.0`. Below is a complete example `Deployment` and `ClusterRoleBinding`:

```yaml
# ===============================
# 🔐 2. NeuroController - ClusterRole（访问权限定义）
# ===============================
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: neurocontroller-cluster-admin
subjects:
  - kind: ServiceAccount
    name: default
    namespace: neuro # 👈 确保和你的 controller 部署一致
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
# ===============================
# 🔗 3. ClusterRoleBinding（赋权给 neuro 命名空间的 default SA）
# ===============================
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: neurocontroller-binding
subjects:
  - kind: ServiceAccount
    name: default
    namespace: neuro
roleRef:
  kind: ClusterRole
  name: neurocontroller-role
  apiGroup: rbac.authorization.k8s.io
---
# ===============================
# 🚀 4. NeuroController - 主控制器 Deployment
# ===============================
apiVersion: apps/v1
kind: Deployment
metadata:
  name: neurocontroller
  namespace: neuro
  labels:
    app: neurocontroller
  annotations:
    neurocontroller.version.latest: "bukahou/neurocontroller:v1.3.0" # 📌 当前部署版本
    neurocontroller.version.previous: "bukahou/neurocontroller:v1.0.0" # 📌 上一次部署版本（用于回滚）

spec:
  replicas: 1
  selector:
    matchLabels:
      app: neurocontroller
  template:
    metadata:
      labels:
        app: neurocontroller
    spec:
      serviceAccountName: default

      nodeSelector:
        kubernetes.io/hostname: desk-eins

      tolerations:
        - key: "node-role.kubernetes.io/control-plane"
          operator: "Exists"
          effect: "NoSchedule"
        - key: "node-role.kubernetes.io/master"
          operator: "Exists"
          effect: "NoSchedule"

      containers:
        - name: neurocontroller
          image: bukahou/neurocontroller:v1.3.0
          imagePullPolicy: Always
          ports:
            - containerPort: 8081 # 📌 Gin 启动服务监听端口
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "256Mi"
              cpu: "200m"
          volumeMounts:
            - name: neuro-log
              mountPath: /var/log/neurocontroller
          envFrom:
            - configMapRef:
                name: neurocontroller-config

      volumes:
        - name: neuro-log
          hostPath:
            path: /var/log/neurocontroller
            type: DirectoryOrCreate
---
# ===============================
# 🌐 5. NeuroController - Service（供 Ingress / 内部调用）
# ===============================
apiVersion: v1
kind: Service
metadata:
  name: neurocontroller-nodeport
  namespace: neuro
spec:
  selector:
    app: neurocontroller
  type: NodePort
  ports:
    - name: static-web
      protocol: TCP
      port: 80
      targetPort: 8081
      nodePort: 30081

---
# ===============================
# 🛠 NeuroController - ConfigMap（環境設定）
# ===============================
apiVersion: v1
kind: ConfigMap
metadata:
  name: neurocontroller-config
  namespace: controller-ns
data:
  # =======================
  # 🔧 診断関連の設定
  # =======================
  DIAGNOSIS_CLEAN_INTERVAL: "5s" # クリーンアップ処理の実行間隔
  DIAGNOSIS_WRITE_INTERVAL: "6s" # ログファイル書き込み間隔
  DIAGNOSIS_RETENTION_RAW_DURATION: "10m" # 元イベントの保持期間
  DIAGNOSIS_RETENTION_CLEANED_DURATION: "5m" # クリーン済みイベントの保持期間
  DIAGNOSIS_UNREADY_THRESHOLD_DURATION: "7s" # アラート発報のしきい値時間
  DIAGNOSIS_ALERT_DISPATCH_INTERVAL: "5s" # メール送信のポーリング間隔
  DIAGNOSIS_UNREADY_REPLICA_PERCENT: "0.6" # レプリカ異常割合のアラート閾値（0〜1）

  # =======================
  # 📡 Kubernetes API ヘルスチェック
  # =======================
  KUBERNETES_API_HEALTH_CHECK_INTERVAL: "15s" # /healthz のチェック間隔

  # =======================
  # 📬 メールアラート設定
  # =======================
  MAIL_SMTP_HOST: "smtp.gmail.com" # SMTP サーバホスト名
  MAIL_SMTP_PORT: "587" # SMTP ポート番号
  MAIL_USERNAME: "<your_email_username>" # メールアカウントのユーザー名
  MAIL_PASSWORD: "<your_app_password_or_token>" # アプリパスワードやトークン
  MAIL_FROM: "neuro@example.com" # 送信元メールアドレス
  MAIL_TO: "user1@example.com,user2@example.com" # 送信先（カンマ区切り）
  ENABLE_EMAIL_ALERT: "true" # メールアラート有効化（true/false）

  # =======================
  # 💬 Slack アラート設定
  # =======================
  SLACK_WEBHOOK_URL: "https://hooks.slack.com/services/XXX/YYY/ZZZ" # Webhook URL
  SLACK_ALERT_DISPATCH_INTERVAL: "5s" # Slack 通知の送信間隔
  ENABLE_SLACK_ALERT: "true" # Slackアラート有効化（true/false）
```

## 📦 Deployment マニフェストの作成 · Write Deployment Manifest

クラスタの構成に応じて Deployment マニフェストを自作し、イメージを利用して適用してください。
Write a Deployment manifest using the pushed image and apply it to your cluster.

さらなる支援や構成例の提供が必要な場合は、いつでもメンテナにご連絡ください。
If you need more help or example manifests, feel free to reach out to the maintainer.
