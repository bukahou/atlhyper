## 🧠 プロジェクト名：AtlHyper

### 📌 プロジェクトの位置付け

AtlHyper は、Kubernetes クラスタの軽量系オブザービリティ・制御プラットフォームです。Node、Pod、Deployment などの資源の実日監視、異常アラート、問題解析やクラスタ操作を実現します。前後端分離構成を採用し、中小規模のローカル部署やエッジククラスタ、開発環境に有効です。

本プロジェクトは **MarstAgent モデル** を採用します。Kubernetes クラスタなどに Agent を常駐させ、データを取得します。主控プロセスである Marst は、Docker Compose などの外部環境で動作することが推奨され、HTTP で各 Agent と通信します。集中管理や複数クラスタの支援を実現します。

---

🫭 デモ表示：
👉 [https://atlhyper.com](https://atlhyper.com)
(デモ環境、一部機能はすでに動作中)
ID：admin
PW：123456

---

### 🚀 機能

| モジュール   | 概要                                                                                          |
| ------------ | --------------------------------------------------------------------------------------------- |
| クラスタ概览 | Node、Pod、Service、Deployment の実日カードとリスト表示                                       |
| 異常アラート | イベントに基づく诊断、重複排除、Slack/メール通知（レート制御機構付き）                        |
| 資料詳細項目 | Pod、Deployment、Namespace などの詳細情報、状態、設定、過去イベント                           |
| 操作支援     | Pod 再起動、Node cordon/drain、資料削除などを UI で実行                                       |
| フィルター   | 各テーブルは Namespace、状態、Node、原因などの項目フィルターを持ち、時間/キーワード検索に対応 |
| 操作ログ編成 | 全ての操作はログに記録され、操作审評画面に表示                                                |
| 設定管理     | Slack/メール/Webhook 通知や許可設定を Web UI 上で管理                                         |

---

### 🛠️ 技術構成

#### 🔧 バックエンド (Golang)

- Gin フレームによる REST API 構築
- controller-runtime / client-go を通じて Kubernetes API に接続
- 異常告知エンジンはモジュール化（阈値判断、レート制御、軽量ログ格式化）
- SQLite を内藏、ログや告知などの統計に使用
- Kubernetes 内部、Docker Compose 外部どちらでも動作可

#### 📺 フロントエンド (Vue2 + Element UI)

- HTML ベースの UI を SPA 構成で再構築
- 組み込み型コンポーネント化 (InfoCard、DataTable、EventTable など)
- ページング、フィルター、時間範囲検索、キーワード検索に対応
- CountUp や ECharts により、数値アニメーションや図表を描画

---
