<p align="right">
🌐 <strong>Languages:</strong>
<a href="./README.md">日本語</a> |
<a href="./README.en.md">English</a> |
<a href="./README.zh-CN.md">简体中文</a>
</p>

## 🧠 プロジェクト名：AtlHyper

### 📌 プロジェクトの位置付け

AtlHyper は、Kubernetes クラスタの軽量系オブザービリティ・制御プラットフォームです。Node、Pod、Deployment などのリソースをリアルタイムに監視し、異常検知、アラート通知、AI 診断、さらにはクラスタ操作までを統合的に提供します。前後端分離構成を採用しており、中小規模クラスタやエッジ環境、開発・検証環境での導入を容易にします。

本プロジェクトは **Master-Agent モデル** を採用しています。Kubernetes クラスタ内部で稼働する Agent がデータを収集し、外部環境（Docker Compose 推奨）で動作する Master が集中管理を行います。Master と Agent は HTTP 通信で接続され、AI 診断サービス（AIServer）や第三者監視連携モジュール（Adapter）と連携します。

---

🫭 デモ環境：
👉 [https://atlhyper.com](https://atlhyper.com)
ID：admin / PW：123456
(一部機能は動作中)

---

### 🚀 機能概要

| モジュール     | 機能概要                                                                            |
| -------------- | ----------------------------------------------------------------------------------- |
| クラスタ概要   | Node、Pod、Service、Deployment の統合ビュー表示。統計カードとテーブルリストで構成。 |
| 異常アラート   | イベントベースの診断、重複排除、Slack・メール通知（レート制御機構付き）。           |
| 詳細ビュー     | Pod、Deployment、Namespace などの詳細構成、状態、過去イベントを可視化。             |
| 操作支援       | Pod 再起動、Node cordon/drain、リソース削除を UI 上から実行可能。                   |
| フィルター検索 | Namespace、状態、Node、原因などのフィルタリング、時間・キーワード検索対応。         |
| 操作ログ       | 全操作を構造化ログとして記録し、履歴画面に表示。                                    |
| 設定管理       | Slack・メール・Webhook 通知やアクセス設定を Web UI で統一管理。                     |

---

### 🏗️ システム構成

```plaintext
AtlHyper/
├── atlhyper_master       # 主制御プロセス（外部環境）
├── atlhyper_agent        # クラスタ内部常駐エージェント
├── atlhyper_metrics      # 指標収集デーモン（Node 指標収集）
├── atlhyper_aiservice    # AI 診断・要因分析モジュール
├── atlhyper_adapter      # 第三者監視システムとの連携アダプタ
├── model/                # 共通リソースモデル（Pod/Node/Event/Metrics...）
├── utils/                # 共通ユーティリティ（gzip, frame, config）
└── web/                  # Vue3 + ElementPlus ベースの管理フロントエンド
```

---

### 🧩 各モジュール概要

#### 🧠 atlhyper_master（主制御）

- Agent からのデータ収集・統合・保存
- /ingest/ エンドポイントによるメトリクス・イベント受付
- Slack・Mail・Webhook への通知統合
- AIService との通信・診断依頼連携
- コントロール機能（Pod 再起動、Node 隔離など）

#### 🛰️ atlhyper_agent（クラスタエージェント）

- Pod、Node、Service、Deployment、Event の監視と収集
- Metrics モジュールとの統合（Node 情報・使用率取得）
- Master への圧縮転送（gzip HTTP）
- 外部からの操作指令を受けて実行

#### 📊 atlhyper_metrics（メトリクス収集）

- Node ごとの温度、ネットワーク速度、ディスク使用率を定期取得
- 軽量構成で Raspberry Pi 環境にも適応
- 収集結果を Agent 経由で Master に転送

#### 🤖 atlhyper_aiservice（AI 診断サービス）

- Master から診断リクエストを受け、LLM による多段階解析を実行
- Stage1（初判）→ Stage2（情報取得）→ Stage3（結論）
- Gemini などの LLM API を利用（将来は RAG 拡張予定）
- 異常原因、推定 Runbook、概要レポートを生成

#### 🔌 atlhyper_adapter（第三者監視アダプタ）

- Prometheus、Zabbix、Datadog、Grafana など外部監視データを受信
- 標準化構造 `ThirdPartyRecord` に変換
- Agent 経由または直接 Master にデータ提供
- 例：/adapter/prometheus/push, /adapter/zabbix/alert

#### 💻 web（フロントエンド）

- Vue3 + ElementPlus による SPA 管理 UI
- クラスタ概要、Pod 詳細、イベントログ表示、告知設定画面
- Axios 統一 API 管理（code=20000 成功判定）
- CountUp.js, ECharts によるリアルタイム統計描画

---

### 🧠 AI パイプライン構成

| ステージ | 概要                                                                  |
| -------- | --------------------------------------------------------------------- |
| Stage1   | イベント群から初期診断を生成。必要なリソースを抽出（needResources）。 |
| Stage2   | Master から該当リソース情報を収集・整理。                             |
| Stage3   | 最終解析と RootCause・Runbook の生成。                                |

---

### 🧭 将来拡張計画

| フェーズ | 内容                                                               |
| -------- | ------------------------------------------------------------------ |
| Phase 1  | atlhyper_adapter による外部監視データ統合（Prometheus / Zabbix）。 |
| Phase 2  | AIService に RAG / Embedding 検索を導入し、知識強化型診断を実装。  |
| Phase 3  | Master にマルチクラスタ・マルチテナント対応を追加。                |
| Phase 4  | Node / Pod の自己修復戦略（Self-Healing Engine）を開発。           |

---

### 🧾 一言まとめ

> **AtlHyper は、軽量かつ拡張性の高い AI 対応 Kubernetes オブザービリティ基盤です。Master-Agent-Adapter-AI の 4 層構成により、異常検知から自動診断、制御までを一元的に統合します。**
