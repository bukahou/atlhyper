# AtlHyper

**軽量 Kubernetes マルチクラスター監視・運用プラットフォーム**

[English](../README.md) | [中文](README_zh.md) | 日本語

---

AtlHyper は軽量 Kubernetes 環境向けの監視・管理プラットフォームです。Master-Agent アーキテクチャを採用し、マルチクラスター統合管理、リアルタイムリソース監視、異常検知、SLO 追跡、リモート運用をサポートしています。

---

## 機能

- **マルチクラスター管理** — 単一のダッシュボードで複数の Kubernetes クラスターを管理
- **リアルタイム監視** — Pod、Node、Deployment のステータスをリアルタイム表示、メトリクス可視化
- **異常検知** — CrashLoopBackOff、OOMKilled、ImagePullBackOff などを自動検知
- **SLO 監視** — Ingress メトリクスに基づくサービス可用性、レイテンシ、エラー率の追跡
- **アラート通知** — メール (SMTP) と Slack (Webhook) に対応
- **リモート運用** — kubectl コマンドのリモート実行、Pod 再起動、レプリカ数調整
- **AI アシスタント** — 自然言語でクラスター運用（オプション）
- **監査ログ** — 完全な操作履歴とユーザー追跡
- **多言語対応** — 英語、中国語、日本語

---

## 技術スタック

| コンポーネント | 技術 | 説明 |
|---------------|------|------|
| **Master** | Go + Gin + SQLite/MySQL | 中央制御、データ集約、API サーバー |
| **Agent** | Go + controller-runtime | クラスターデータ収集、コマンド実行 |
| **Metrics** | Go (DaemonSet) | ノードレベルのメトリクス収集 (CPU/メモリ/ディスク/ネットワーク) |
| **Web** | Next.js 15 + TypeScript + Tailwind CSS | モダンなレスポンシブダッシュボード |

---

## アーキテクチャ

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           AtlHyper プラットフォーム                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────┐     ┌─────────────────────────────────────────────────┐   │
│   │   Web UI    │────▶│                    Master                       │   │
│   │  (Next.js)  │◀────│                                                 │   │
│   └─────────────┘     │  ┌─────────┐  ┌──────────┐  ┌───────────────┐   │   │
│                       │  │ Gateway │  │ DataHub  │  │   Services    │   │   │
│                       │  │  (API)  │  │ (メモリ) │  │ (SLO/アラート)│   │   │
│                       │  └─────────┘  └──────────┘  └───────────────┘   │   │
│                       │                     │                           │   │
│                       │              ┌──────┴──────┐                    │   │
│                       │              │ データベース │                    │   │
│                       │              │(SQLite/MySQL)│                   │   │
│                       └──────────────┴──────────────┴───────────────────┘   │
│                                          │                                  │
│            ┌─────────────────────────────┼─────────────────────────────┐    │
│            │                             │                             │    │
│            ▼                             ▼                             ▼    │
│   ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   │Agent (クラスタA)│         │Agent (クラスタB)│         │Agent (クラスタN)│
│   │                 │         │                 │         │                 │
│   │  ┌───────────┐  │         │  ┌───────────┐  │         │  ┌───────────┐  │
│   │  │  Source   │  │         │  │  Source   │  │         │  │  Source   │  │
│   │  │ ├─ イベント│  │         │  │ ├─ イベント│  │         │  │ ├─ イベント│  │
│   │  │ ├─ スナップ│ │         │  │ ├─ スナップ│ │         │  │ ├─ スナップ│ │
│   │  │ └─ メトリクス│ │        │  │ └─ メトリクス│ │        │  │ └─ メトリクス│ │
│   │  ├───────────┤  │         │  ├───────────┤  │         │  ├───────────┤  │
│   │  │  Executor │  │         │  │  Executor │  │         │  │  Executor │  │
│   │  └───────────┘  │         │  └───────────┘  │         │  └───────────┘  │
│   └────────┬────────┘         └────────┬────────┘         └────────┬────────┘
│            │                           │                           │        │
│            ▼                           ▼                           ▼        │
│   ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
│   │   Kubernetes    │         │   Kubernetes    │         │   Kubernetes    │
│   │   クラスター A   │         │   クラスター B   │         │   クラスター N   │
│   │ ┌─────────────┐ │         │ ┌─────────────┐ │         │ ┌─────────────┐ │
│   │ │   Metrics   │ │         │ │   Metrics   │ │         │ │   Metrics   │ │
│   │ │ (DaemonSet) │ │         │ │ (DaemonSet) │ │         │ │ (DaemonSet) │ │
│   │ └─────────────┘ │         │ └─────────────┘ │         │ └─────────────┘ │
│   └─────────────────┘         └─────────────────┘         └─────────────────┘
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## データフロー

AtlHyper は4つのモジュールで構成され、それぞれ独立したデータフローを持ちます：

### 1. Agent データフロー（4つのストリーム）

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Agent → Master データフロー                        │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [イベントストリーム]                                                      │
│  K8s Watch ──▶ 異常フィルタ ──▶ DataHub ──▶ Pusher ──▶ Master            │
│  • 検知: CrashLoop、OOM、ImagePull、NodeNotReady など                    │
│                                                                          │
│  [スナップショットストリーム]                                               │
│  SDK.List() ──▶ Snapshot ──▶ Pusher ──▶ Master                          │
│  • リソース: Pod、Node、Deployment、Service、Ingress など                 │
│                                                                          │
│  [メトリクスストリーム]                                                     │
│  Metrics DaemonSet ──▶ Agent Gateway ──▶ Receiver ──▶ Pusher ──▶ Master │
│  • ノードメトリクス: 各ノードの CPU、メモリ、ディスク、ネットワーク          │
│                                                                          │
│  [コマンドストリーム]                                                      │
│  Master ──▶ Agent Gateway ──▶ Executor ──▶ K8s SDK ──▶ 結果 ──▶ Master  │
│  • 操作: Pod 再起動、レプリカ調整、ノード隔離など                           │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 2. Metrics DaemonSet データフロー

```
┌──────────────────────────────────────────────────────────────────────────┐
│                     メトリクスコレクター（各ノード）                         │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                   │
│  │ /proc/stat  │    │/proc/meminfo│    │/proc/diskstats│                 │
│  │ /proc/net   │    │   syscall   │    │/proc/mounts │                   │
│  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                   │
│         │                  │                  │                          │
│         ▼                  ▼                  ▼                          │
│  ┌─────────────────────────────────────────────────────┐                 │
│  │           メトリクスコレクター (Go)                   │                 │
│  │  • CPU: 使用率、コアごと、ロードアベレージ              │                 │
│  │  • メモリ: 使用中、利用可能、キャッシュ、バッファ        │                 │
│  │  • ディスク: 容量、IO レート、IOPS、使用率             │                 │
│  │  • ネットワーク: インターフェースごとのバイト/パケット   │                 │
│  └──────────────────────────┬──────────────────────────┘                 │
│                             │                                            │
│                             ▼                                            │
│                    POST /metrics/push                                    │
│                             │                                            │
│                             ▼                                            │
│                    ┌─────────────────┐                                   │
│                    │  Agent (同ノード) │                                  │
│                    └─────────────────┘                                   │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 3. Master データフロー（3つのストリーム）

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         Master データストリーム                            │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  [1. クラスタースナップショット — メモリストレージ (DataHub)]               │
│  ─────────────────────────────────────────────────────────────────────── │
│  Agent ──▶ AgentSDK ──▶ Processor ──▶ DataHub (メモリ)                   │
│                                            │                             │
│  用途: Pod/Node のリアルタイムクエリ        ◀── Web API クエリ            │
│  保持: 最新のスナップショットのみ                                          │
│                                                                          │
│  [2. コマンドディスパッチ — メッセージキュー]                               │
│  ─────────────────────────────────────────────────────────────────────── │
│  ユーザー/AI ──▶ API ──▶ CommandBus ──▶ Agent 実行                       │
│                          │                                               │
│  用途: リモート操作      Agent ──▶ 結果 ──▶ CommandBus ──▶ API           │
│  保持: 一時的                                                             │
│                                                                          │
│  [3. 永続化データ — データベース]                                          │
│  ─────────────────────────────────────────────────────────────────────── │
│  Agent ──▶ Processor ──┬──▶ イベント ──▶ DB (event_history)              │
│                        ├──▶ SLO メトリクス ──▶ DB (slo_* テーブル)        │
│                        └──▶ ノードメトリクス ──▶ DB (node_metrics_history)│
│                                    │                                     │
│  用途: 履歴分析                    ◀── トレンド/SLO API クエリ            │
│  保持: 30-180日（設定可能）                                               │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

### 4. Web フロントエンドデータフロー

```
┌──────────────────────────────────────────────────────────────────────────┐
│                       Web フロントエンドフロー                             │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                   │
│  │  ブラウザ   │    │  Next.js    │    │   Master    │                   │
│  │             │───▶│ ミドルウェア │───▶│   Gateway   │                   │
│  │             │◀───│  (プロキシ)  │◀───│   (API)     │                   │
│  └─────────────┘    └─────────────┘    └─────────────┘                   │
│                                                                          │
│  • 認証: JWT トークンを localStorage に保存                               │
│  • API プロキシ: /api/v2/* → Master:8080 (ランタイム設定)                 │
│  • 状態: Zustand でグローバル状態管理                                     │
│  • リアルタイム: 設定可能な間隔でポーリング                                 │
│                                                                          │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## スクリーンショット

### クラスター概要
クラスターの健全性、リソース使用状況、最近のアラートをリアルタイムで表示。

![クラスター概要](img/overview.png)

### Pod 管理
ネームスペース横断で Pod を一覧、フィルタ、管理。詳細なステータスを表示。

![Pod 管理](img/cluster_pod.png)

### アラートダッシュボード
クラスターアラートの表示と分析。フィルタリングと AI 分析をサポート。

![アラートダッシュボード](img/cluster_alert.png)

### ノードメトリクス
ノードレベルの詳細なメトリクスと履歴トレンドグラフ。

![ノードメトリクス](img/system_metrics.png)

### SLO 監視
Ingress メトリクスに基づくサービスレベル目標の追跡。

![SLO 概要](img/workbench_slo_overview.png)

![SLO 詳細](img/workbench_slo.png)

---

## デプロイ

### 前提条件

- Go 1.21+
- Node.js 18+
- Kubernetes クラスター（Agent デプロイ用）
- Docker（コンテナ化デプロイ）

### クイックスタート（開発環境）

**1. Master 起動**
```bash
export MASTER_ADMIN_USERNAME=admin
export MASTER_ADMIN_PASSWORD=$(openssl rand -base64 16)
export MASTER_JWT_SECRET=$(openssl rand -base64 32)

cd cmd/atlhyper_master_v2
go run main.go
# Gateway: :8080, AgentSDK: :8081
```

**2. Agent 起動（K8s クラスター内）**
```bash
cd cmd/atlhyper_agent_v2
go run main.go \
  --cluster-id=my-cluster \
  --master=http://<MASTER_IP>:8081
```

**3. Web 起動**
```bash
cd atlhyper_web
npm install && npm run dev
# アクセス: http://localhost:3000
```

### Kubernetes デプロイ（Helm）

```bash
# Helm リポジトリ追加（公開済みの場合）
helm repo add atlhyper https://charts.atlhyper.io

# Master インストール
helm install atlhyper-master atlhyper/atlhyper \
  --set master.admin.username=admin \
  --set master.admin.password=<YOUR_PASSWORD> \
  --set master.jwt.secret=<YOUR_SECRET>

# Agent インストール（各クラスター）
helm install atlhyper-agent atlhyper/atlhyper-agent \
  --set agent.clusterId=production \
  --set agent.masterUrl=http://atlhyper-master:8081
```

### Kubernetes デプロイ（YAML）

デプロイ順序: **Master → Agent → Metrics → Web**

```bash
cd deploy/k8s

# 1. ネームスペースと設定を作成
kubectl apply -f atlhyper-config.yaml

# 2. Master デプロイ
kubectl apply -f atlhyper-Master.yaml

# 3. Agent デプロイ
kubectl apply -f atlhyper-agent.yaml

# 4. Metrics デプロイ (DaemonSet)
kubectl apply -f atlhyper-metrics.yaml

# 5. Web デプロイ
kubectl apply -f atlhyper-web.yaml

# 6. (オプション) Traefik IngressRoute
kubectl apply -f atlhyper-traefik.yaml
```

### 設定リファレンス

#### Master 環境変数

| 変数 | 必須 | デフォルト | 説明 |
|------|------|-----------|------|
| `MASTER_ADMIN_USERNAME` | はい | - | 管理者ユーザー名 |
| `MASTER_ADMIN_PASSWORD` | はい | - | 管理者パスワード |
| `MASTER_JWT_SECRET` | はい | - | JWT 署名キー |
| `MASTER_GATEWAY_PORT` | いいえ | `8080` | Web/API ポート |
| `MASTER_AGENTSDK_PORT` | いいえ | `8081` | Agent データポート |
| `MASTER_DB_TYPE` | いいえ | `sqlite` | データベースタイプ |
| `MASTER_DB_DSN` | いいえ | - | MySQL/PostgreSQL DSN |
| `MASTER_LOG_LEVEL` | いいえ | `info` | ログレベル |

#### Agent 設定

| フラグ | 必須 | 説明 |
|--------|------|------|
| `--cluster-id` | はい | クラスター固有識別子 |
| `--master` | はい | Master AgentSDK URL |

#### Metrics DaemonSet

Metrics コレクターは自動的に DaemonSet としてデプロイされ、ローカル Agent に報告します。ConfigMap で設定：

| 変数 | デフォルト | 説明 |
|------|-----------|------|
| `METRICS_AGENT_URL` | `http://atlhyper-agent:8082` | Agent メトリクスエンドポイント |
| `METRICS_PUSH_INTERVAL` | `15s` | プッシュ間隔 |

---

## プロジェクト構造

```
atlhyper/
├── atlhyper_master_v2/       # Master（中央制御）
│   ├── gateway/              # HTTP API (Web + AgentSDK)
│   ├── datahub/              # メモリデータストア
│   ├── database/             # 永続化ストレージ (SQLite/MySQL)
│   ├── service/              # ビジネスロジック (SLO, アラート)
│   ├── ai/                   # AI アシスタント統合
│   └── config/               # 設定管理
│
├── atlhyper_agent_v2/        # Agent（クラスタープロキシ）
│   ├── source/               # データソース
│   │   ├── event/            # K8s イベントウォッチャー
│   │   ├── snapshot/         # リソーススナップショット
│   │   └── metrics/          # メトリクスレシーバー
│   ├── executor/             # コマンド実行
│   ├── sdk/                  # K8s 操作
│   └── pusher/               # データプッシュスケジューラー
│
├── atlhyper_metrics_v2/      # メトリクスコレクター (DaemonSet)
│   ├── collector/            # CPU、メモリ、ディスク、ネットワーク
│   └── pusher/               # Agent へプッシュ
│
├── atlhyper_web/             # Web フロントエンド
│   ├── src/app/              # Next.js ページ
│   ├── src/components/       # React コンポーネント
│   ├── src/api/              # API クライアント
│   └── src/i18n/             # 国際化
│
├── model_v2/                 # 共有データモデル
├── cmd/                      # エントリーポイント
└── deploy/                   # デプロイ設定
    ├── helm/                 # Helm チャート
    └── k8s/                  # K8s マニフェスト
```

---

## セキュリティ

### 機密情報

- API キー、パスワード、シークレットをコードに**ハードコードしない**
- すべての認証情報は環境変数を使用
- AI API キーはデータベースに暗号化して保存（Web UI で設定）

### コミット前チェック

```bash
# 潜在的な API キー漏洩をスキャン
grep -rE "sk-[a-zA-Z0-9]{20,}|AIza[a-zA-Z0-9]{30,}" \
  --include="*.go" --include="*.ts" --include="*.tsx" .
```

### .gitignore で除外されるファイル

- `atlhyper_master_v2/database/sqlite/data/` — データベースファイル
- `atlhyper_web/.env.local` — ローカル環境
- `*.db` — すべての SQLite データベース

---

## ライセンス

MIT

---

## リンク

- [GitHub リポジトリ](https://github.com/bukahou/atlhyper)
