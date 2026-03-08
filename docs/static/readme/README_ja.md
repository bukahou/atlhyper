# AtlHyper

**AI 時代の次世代 Kubernetes SRE プラットフォーム**

[English](../../README.md) | [中文](README_zh.md) | 日本語

---

AtlHyper は AI 時代の次世代 SRE プラットフォームです。Master-Agent アーキテクチャを採用し、マルチクラスター Kubernetes 環境を管理します。四信号ドメインのフルスタックオブザーバビリティ（Metrics / APM / Logs / SLO）、アルゴリズム駆動の AIOps エンジン、AI アシスタント運用を提供し、「システム動作認知モデル」を構築してシステムが自らを理解できるようにすることを目指しています。

---

## 機能

- **マルチクラスター管理** — 単一のダッシュボードで複数の Kubernetes クラスターを管理、Agent 自動登録
- **リアルタイム監視** — Pod、Node、Deployment など 21 種類の K8s リソースのリアルタイムステータスとメトリクス可視化
- **四信号ドメインオブザーバビリティ** — ClickHouse + OTel Collector ベースの Metrics / APM / Logs / SLO フルスタックオブザーバビリティ
- **分散トレーシング (APM)** — Trace ウォーターフォール図、Span 詳細、サービストポロジー、レイテンシ分布、データベース呼出分析
- **ログクエリ** — 多次元フィルタ（サービス/レベル/ソースクラス）、ヒストグラムタイムライン、構造化ログ詳細、Trace 関連付け
- **SLO 監視** — Ingress（Traefik）+ サービスメッシュ（Linkerd）二層 SLO トラッキング、レイテンシ分布、エラーバジェット、ステータスコード分布
- **AIOps エンジン** — 依存グラフ構築、EMA 動的ベースライン、三段階リスクスコアリング、ステートマシン、インシデントライフサイクル管理
- **因果トポロジーグラフ** — 四層有向非巡回グラフ（Ingress→Service→Pod→Node）、リスク伝播可視化
- **AI アシスタント** — マルチモデル駆動の自然言語運用（Chat + Tool Use）、Gemini / OpenAI / Claude / Ollama（ローカル）対応、インシデント要約と根本原因分析
- **AI マルチロールルーティング** — 3つの AI ロール（background / chat / analysis）、ロール別プロバイダールーティング、日次トークン/コール予算制御
- **AI インシデント分析** — インシデント作成時に自動バックグラウンド分析、マルチラウンド Tool Calling による深層調査（最大8ラウンド）、信頼度スコア付き構造化レポート
- **アラート通知** — メール (SMTP) と Slack (Webhook) に対応
- **リモート運用** — kubectl コマンドのリモート実行、Pod 再起動、レプリカ数調整、ノード隔離
- **監査ログ** — 完全な操作履歴とユーザー追跡
- **多言語対応** — 中国語、日本語

---

## 技術スタック

| コンポーネント | 技術 | 説明 |
|---------------|------|------|
| **Master** | Go 1.24 + net/http + SQLite | 中央制御、データ集約、API サーバー、AIOps エンジン |
| **Agent** | Go 1.24 + client-go + ClickHouse | クラスターデータ収集、OTel データクエリ、コマンド実行 |
| **Web** | Next.js 16 + React 19 + Tailwind CSS 4 + ECharts + G6 | 可視化管理インターフェース |
| **オブザーバビリティ** | ClickHouse + OTel Collector + Linkerd | 時系列ストレージ、テレメトリ収集、サービスメッシュ |
| **AI** | Gemini / OpenAI / Claude / Ollama (Chat + Tool Use) | AI チャット運用、マルチロールルーティング、インシデント分析 |

---

## システムアーキテクチャ

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                         AtlHyper プラットフォーム                                  │
├──────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌──────────┐    ┌──────────────────────────────────────────────────────────┐    │
│  │  Web UI  │───▶│                        Master                           │    │
│  │(Next.js) │◀───│                                                          │    │
│  └──────────┘    │  ┌─────────┐ ┌────────┐ ┌─────────┐ ┌────────────────┐  │    │
│                  │  │ Gateway │ │DataHub │ │ Service │ │   Database     │  │    │
│                  │  │  (API)  │ │(メモリ) │ │(業務層) │ │   (SQLite)     │  │    │
│                  │  └─────────┘ └────────┘ └─────────┘ └────────────────┘  │    │
│                  │  ┌──────────────────┐   ┌──────────────────────────┐     │    │
│                  │  │  AIOps Engine    │   │ AI (Multi-LLM+ロール)   │     │    │
│                  │  │依存グラフ│基線│ﾘｽｸ│   │ Gemini│OpenAI│Claude   │     │    │
│                  │  │ｽﾃｰﾄﾏｼﾝ│ｲﾝｼﾃﾞﾝﾄ   │   │ Ollama│ロールルーティング│     │    │
│                  │  │       │          │   └──────────────────────────┘     │    │
│                  │  └──────────────────┘                                    │    │
│                  └──────────────────────────────────────────────────────────┘    │
│                                          │                                       │
│         ┌────────────────────────────────┼────────────────────────────────┐      │
│         │                                │                                │      │
│         ▼                                ▼                                ▼      │
│  ┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐  │
│  │Agent (クラスタA) │         │Agent (クラスタB) │         │Agent (クラスタN) │  │
│  │                  │         │                  │         │                  │  │
│  │ SDK (K8s+CH)     │         │ SDK (K8s+CH)     │         │ SDK (K8s+CH)     │  │
│  │ Repository       │         │ Repository       │         │ Repository       │  │
│  │ Concentrator     │         │ Concentrator     │         │ Concentrator     │  │
│  │ Service          │         │ Service          │         │ Service          │  │
│  │ Scheduler        │         │ Scheduler        │         │ Scheduler        │  │
│  └────────┬─────────┘         └────────┬─────────┘         └────────┬─────────┘  │
│           │                            │                            │            │
│           ▼                            ▼                            ▼            │
│  ┌──────────────────┐         ┌──────────────────┐         ┌──────────────────┐  │
│  │ Kubernetes クラスタ│         │ Kubernetes クラスタ│         │ Kubernetes クラスタ│  │
│  │                  │         │                  │         │                  │  │
│  │ ┌──────────────┐ │         │ ┌──────────────┐ │         │ ┌──────────────┐ │  │
│  │ │OTel Collector│ │         │ │OTel Collector│ │         │ │OTel Collector│ │  │
│  │ │node_exporter │ │         │ │node_exporter │ │         │ │node_exporter │ │  │
│  │ │   Linkerd    │ │         │ │   Linkerd    │ │         │ │   Linkerd    │ │  │
│  │ │  ClickHouse  │ │         │ │  ClickHouse  │ │         │ │  ClickHouse  │ │  │
│  │ └──────────────┘ │         │ └──────────────┘ │         │ └──────────────┘ │  │
│  └──────────────────┘         └──────────────────┘         └──────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## スクリーンショット

### クラスター概要
クラスター健全性、ワークロード概要、SLO 概要、ノードリソース使用量、最近のアラート。

![クラスター概要](../img/overview.png)

### Pod 管理
ネームスペース横断の Pod リスト、フィルタ対応。右側ドロワーで Pod 詳細（基本情報、コンテナ、ボリュームマウント、ネットワーク、スケジューリング）を表示。

![Pod 管理](../img/cluster-pod.png)

### Node 管理
ノードリストと詳細ドロワー。システム情報、ロール、Pod CIDR、コンテナランタイムバージョンを表示。ノードの隔離/隔離解除操作に対応。

![Node 管理](../img/cluster-node.png)

### ノードメトリクス
クラスター全体のノードレベルハードウェアメトリクス：CPU、メモリ、ディスク、温度。複数の時間範囲（1h/6h/1d/7d）と粒度（1m/5m/15m）に対応。

![ノードメトリクス](../img/observe-metrics.png)

### APM 分散トレーシング
分散トレーシング分析：レイテンシ分布ヒストグラム、Trace ウォーターフォール図、Span 詳細（データベース属性と K8s コンテキストを含む）、Trace - Log 関連付け対応。

![APM 分散トレーシング](../img/observe-apm.png)

### ログクエリ
多次元ログフィルタ（サービス/レベル/ソースクラス）、タイムラインヒストグラム、構造化ログ詳細（Trace ID、K8s リソース情報を含む）、全文検索対応。

![ログクエリ](../img/observe-logs.png)

### SLO 監視
ドメインレベル SLO 概要（可用性、P95 レイテンシ、エラー率、エラーバジェット）、レイテンシ分布ヒストグラム、リクエストメソッド分布、ステータスコード分布。

![SLO 監視](../img/observe-slo.png)

### AIOps リスクダッシュボード
クラスターリスクスコア（0-100）、高リスクエンティティリスト。ローカルリスク/最終リスク/リスクレベルと初回異常検知時刻を表示。

![AIOps リスク](../img/aiops-risk.png)

### AIOps 因果トポロジー
四層依存グラフ（Node→Pod→Service→Ingress）、リスク伝播可視化。ノード詳細パネルでベースラインメトリクスと因果チェーンを表示。

![AIOps トポロジー](../img/aiops-topology.png)

### AI アシスタント
マルチモデル駆動の自然言語運用対話（Gemini / OpenAI / Claude / Ollama 対応）、Tool Use（インシデントクエリ、分析）に対応。3つの AI ロール（background / chat / analysis）によるロール別プロバイダールーティングと予算制御。構造化されたインシデント要約と根本原因分析を自動出力。

![AI アシスタント](../img/aiops-chat.png)

---

## データフロー

### Agent → Master（スナップショット送信 + コマンド実行）

```
[スナップショットストリーム]
K8s SDK ──▶ Repository ──▶ SnapshotService ──▶ Scheduler ──▶ Master
• K8s リソース: Pod、Node、Deployment、Service、Ingress など 21 種類
• OTel データ: ClickHouse クエリで Metrics / APM / Logs / SLO 四信号ドメインを取得
• 時系列集約: Concentrator リングバッファ（1 時間 × 1 分粒度）

[コマンドストリーム]
Master ──▶ Agent Poll ──▶ CommandService ──▶ K8s SDK ──▶ 結果 → Master

[ハートビートストリーム]
Agent ──▶ 定期ハートビート ──▶ Master（接続状態維持）
```

### オブザーバビリティパイプライン（OTel → ClickHouse → Agent）

```
[ノードメトリクス]  node_exporter ──▶ OTel Collector ──▶ ClickHouse
[Ingress]          Traefik ──▶ OTel Collector ──▶ ClickHouse
[Mesh]             Linkerd Proxy ──▶ OTel Collector ──▶ ClickHouse
[Traces]           アプリ SDK ──▶ OTel Collector ──▶ ClickHouse
[Logs]             アプリログ ──▶ Filebeat ──▶ OTel Collector ──▶ ClickHouse

                    ClickHouse ◀── Agent 定期クエリ
```

---

## デプロイ

### 前提条件

- Go 1.24+
- Node.js 20+
- Kubernetes クラスター（Agent デプロイ用）
- ClickHouse（オブザーバビリティデータストレージ）

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
# クラスター ID は自動検出（kube-system UID）、環境変数で指定も可能
export AGENT_MASTER_URL=http://<MASTER_IP>:8081
# export AGENT_CLUSTER_ID=my-cluster  # オプション、デフォルトは自動検出

cd cmd/atlhyper_agent_v2
go run main.go
```

**3. Web 起動**
```bash
cd atlhyper_web
npm install && npm run dev
# アクセス: http://localhost:3000
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
| `MASTER_LOG_LEVEL` | いいえ | `info` | ログレベル |

#### Agent 環境変数

| 変数 | 必須 | デフォルト | 説明 |
|------|------|-----------|------|
| `AGENT_MASTER_URL` | はい | - | Master AgentSDK アドレス |
| `AGENT_CLUSTER_ID` | いいえ | 自動検出 | クラスター固有識別子（デフォルトは kube-system UID を使用） |
| `AGENT_CLICKHOUSE_DSN` | いいえ | - | ClickHouse 接続アドレス（OTel クエリを有効化） |

---

## プロジェクト構造

```
atlhyper/
├── atlhyper_master_v2/     # Master（中央制御）— 41k 行
│   ├── gateway/            #   HTTP API ゲートウェイ
│   │   └── handler/        #     Handler（k8s/observe/aiops/admin/slo サブディレクトリ）
│   ├── service/            #   ビジネスロジック（query + operations）
│   ├── datahub/            #   メモリデータストア
│   ├── database/           #   永続化 (SQLite)
│   ├── processor/          #   データ処理
│   ├── agentsdk/           #   Agent 通信層
│   ├── mq/                 #   メッセージキュー
│   ├── aiops/              #   AIOps エンジン
│   ├── ai/                 #   AI アシスタント (Gemini/OpenAI/Claude/Ollama)
│   ├── slo/                #   SLO ルート更新
│   ├── notifier/           #   アラート通知
│   └── config/             #   設定
│
├── atlhyper_agent_v2/      # Agent（クラスタープロキシ）— 20k 行
│   ├── sdk/                #   K8s + ClickHouse SDK
│   ├── repository/         #   データリポジトリ (K8s + CH クエリ)
│   ├── service/            #   スナップショット/コマンドサービス
│   ├── concentrator/       #   OTel 時系列集約（リングバッファ）
│   ├── scheduler/          #   スケジューラー
│   └── gateway/            #   Agent↔Master 通信
│
├── atlhyper_web/           # Web フロントエンド — 55k 行
│   ├── src/app/            #   Next.js ページ
│   ├── src/components/     #   React コンポーネント
│   ├── src/api/            #   API クライアント
│   ├── src/datasource/     #   データソース層（API + mock フォールバック）
│   └── src/i18n/           #   国際化 (中国語/日本語)
│
├── model_v3/               # 共有モデル (cluster/agent/metrics/slo/command/apm/log)
├── common/                 # ユーティリティ (logger/crypto/gzip)
├── cmd/                    # エントリーポイント
└── docs/                   # ドキュメント
```

---

## AIOps エンジン

アルゴリズム駆動の AIOps エンジンで、自動化された異常検知、根本原因特定、インシデントライフサイクル管理を実現。コア設計原則：**説明可能なアルゴリズム** — すべてのリスクスコアは具体的な計算式と入力メトリクスに遡及可能、ML ブラックボックスではない。

### M1 — 依存グラフ（Correlator）

`ClusterSnapshot` から四層の有向非巡回グラフ（DAG）を自動構築：

```
Ingress ──routes_to──▶ Service ──selects──▶ Pod ──runs_on──▶ Node
                         │
                         └──calls──▶ Service（Linkerd サービス間トラフィック）
```

- **データソース**: K8s API（リソース関係）+ Linkerd outbound（サービス間呼出）+ OTel Traces（トレースチェーン）
- **グラフ構造**: 正方向/逆方向隣接リスト、BFS によるチェーントレーシング対応
- **永続化**: 各スナップショット後に非同期で SQLite へ書き込み

### M2 — ベースラインエンジン（Baseline）

**EMA（指数移動平均）+ 3σ 動的ベースライン**、二チャネル異常検知：

**チャネル A — 統計型検知：**

```
EMA_t = α × x_t + (1-α) × EMA_{t-1}     (α = 0.033, 60 サンプルポイント相当)
異常スコア = sigmoid(|x - EMA| / σ - 3)    (偏差 > 3σ で異常)
```

| エンティティ | 監視メトリクス |
|-------------|--------------|
| Node | cpu_usage, memory_usage, disk_usage, psi_cpu/memory/io |
| Pod | restart_count, is_running, not_ready_containers |
| Service (Linkerd) | error_rate, avg_latency, request_rate |
| Ingress (Traefik) | error_rate, avg_latency |

**チャネル B — 確定的検知（コールドスタートバイパス）：**

| 検知項目 | スコア |
|---------|-------|
| OOMKilled | 0.95 |
| CrashLoopBackOff | 0.90 |
| 設定エラー | 0.80 |
| K8s Critical Event（5 分以内） | 0.85 |
| Deployment 利用不可 ≥75% | 0.95 |

### M3 — リスクスコアリング（Risk Scorer）

三段階パイプライン、ローカルメトリクスからグローバルトポロジーへ：

```
Stage 1 — ローカルリスク:  R_local = max(R_stat, R_det)
Stage 2 — 時系列減衰:     W_time = 0.7 + 0.3 × (1 - exp(-Δt / τ))
Stage 3 — グラフ伝播:     R_final = f(R_weighted, avg(R_final(deps)), SLO_context)
```

| R_final | レベル |
|---------|--------|
| ≥ 0.8 | Critical |
| ≥ 0.6 | High |
| ≥ 0.4 | Medium |
| ≥ 0.2 | Low |
| < 0.2 | Healthy |

### M4 — ステートマシン（State Machine）

```
                    R>0.2 持続≥2min           R>0.5 持続≥5min
  Healthy ──────────────────▶ Warning ──────────────────▶ Incident
     ▲  R<0.15 持続≥5min       │                            │
     └──────────────────────────┘          R<0.15 持続≥10min │
                                                             ▼
                               R>0.2 即時再発             Recovery
                    Warning ◀─────────────────────────────── │
                                                             │
                                         定期チェック（10min）│
                                              Stable ◀───────┘
```

### M5 — インシデントストア（Incident Store）

SQLite で永続化、構造化されたインシデント記録：

| データ | 内容 |
|--------|------|
| **Incident** | ID、クラスター、状態、重大度、根本原因エンティティ、ピークリスク、持続時間 |
| **Entity** | 影響を受けたエンティティリスト（R_local / R_final / ロールを含む） |
| **Timeline** | 状態変更タイムライン |
| **Statistics** | MTTR、再発率、重大度分布、Top 根本原因 |

### M6 — AI 強化（AI Enhancer）

LLM 駆動のインシデント分析、3つの AI ロール：

| ロール | トリガー | 説明 |
|--------|---------|------|
| **background** | 自動（インシデント作成/エスカレーション時） | 高速要約、対処提案、類似インシデント。レートリミット（60秒/インシデント）、結果24時間キャッシュ |
| **chat** | ユーザー起動 | インタラクティブな自然言語運用、SSE ストリーミング |
| **analysis** | ユーザー起動 | マルチラウンド深層調査（最大8ラウンド × 5 Tool Call）、信頼度スコア付き構造化レポート |

- **マルチプロバイダー**: Gemini / OpenAI / Claude / Ollama（ローカル）、各ロールは異なるプロバイダーにルーティング可能
- **ロール予算**: ロール毎の日次トークンリミットとコールリミット、予算枯渇時にフォールバックプロバイダーへ切り替え
- **レポート永続化**: すべての AI レポート（要約、根本原因分析、調査ステップ）を SQLite に永続化

---

## セキュリティ

- API キー、パスワード、シークレットをコードに**ハードコード禁止**
- すべての認証情報は環境変数を使用
- AI API キーはデータベースに暗号化して保存（Web UI で設定）
- K8s Secret 内容はマスク表示

---

## ライセンス

MIT

---

## リンク

- [GitHub リポジトリ](https://github.com/bukahou/atlhyper)
