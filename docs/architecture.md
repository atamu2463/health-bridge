# HealthBridge システム構成・API／DB設計

## 1. 文書の位置づけ

本書は、現在の実装と、これから実装するMVPのAPI・DB設計を分けて記録します。実装前に確定した設計と、引き続き判断が必要な事項を明示します。

- 参照資料：Googleドキュメント[「【Docs】体調管理・共有アプリ」](https://docs.google.com/document/d/1EKeJU0Jydtram7yDto50QVDMukCxzGgsQxHxHNcGDyk/edit?tab=t.0)の「要件定義・技術選定 v2（再レビュー用）」「画面設計 v2（再レビュー用）」「DB設計v2」「API設計v2」「2026.09.17MTG」
- 資料参照日：2026年9月18日
- 判断基準：第2回レビュー後の決定、Issue #110 / #112 / #133の反映内容、現在のコードをv2案より優先

業務API設計は実装前のものです。DBモデル、マイグレーション、マスターデータ、管理用ユーザー作成CLIに加え、セッション認証APIと認証middlewareは実装済みです。業務APIと、本人・担当範囲に基づくサーバー側認可は未実装です。

## 2. 現在と予定のシステム構成

```text
利用者のWebブラウザ
        │
        ▼
Frontend: Next.js / React / TypeScript
配置: Vercel（公開中）
        │
        │ HTTP / JSON（接続予定）
        ▼
Backend: Go / Gin / GORM
配置: Render Docker Web Service（公開中）
現在: GET /health、共通middleware、graceful shutdown、DBモデル／マイグレーション
予定: PostgreSQLセッション認証、Ginによる業務API
        │
        │ PostgreSQLプロトコル / GORM
        │ Session pooler、TLS（sslmode=require）
        ▼
PostgreSQL
開発: Docker上
本番: Supabase PostgreSQL（構築済み）
```

フロントエンドはモックUIを公開している段階であり、Render上のバックエンドとは業務APIで接続していません。GoバックエンドはGORMからSession poolerを経由してSupabase PostgreSQLへ接続し、Supabase Data API、Supabase Auth、`supabase-js`は使用しません。

### 2.1 本番環境（構築済み）

| 対象 | 配置・リージョン | 現在の設定・確認範囲 |
| --- | --- | --- |
| フロントエンド | Vercel | `https://health-bridge-management.vercel.app` でモックUIを公開 |
| バックエンド | Render Docker Web Service / Singapore / Free | `main`をデプロイ対象とし、Root Directoryは`backend`。DockerfileのCMDを使用し、Docker Commandは上書きしていない。Health Check Pathは`/health`、Auto-Deployは無効 |
| データベース | Supabase PostgreSQL / 東京 | GORMでPostgreSQLへ接続。Session poolerとTLS（`sslmode=require`）を使用 |

Render Freeプランでは非稼働時間後にサービスがスリープし、次の初回アクセスに時間がかかる場合があります。

## 3. コンポーネントと実装状態

| コンポーネント | 役割 | 現在の状態 |
| --- | --- | --- |
| フロントエンド | 利用者別画面、入力、一覧、グラフ、モック操作 | モックUI実装済み。業務API未接続 |
| モック状態管理 | 従業員・担当関係・体調記録の一時更新 | React Contextで実装。再読み込みで初期化 |
| バックエンド | 認証・認可、業務ルール、DB操作 | ローカルでは認証API、セッション認証middleware、5テーブルのモデル／マイグレーション、管理用ユーザー作成CLIまで実装。RenderはIssue #138の共通基盤まで公開済み |
| Health Check | HTTPプロセスの稼働確認 | `GET /health` 実装済み。Renderの公開環境で正常応答を確認済み。DB疎通確認ではない |
| 業務API | アカウント・体調記録・担当関係の処理 | 設計済み・未実装 |
| 開発DB | ローカル開発データの保存 | Docker ComposeでPostgreSQLを構成。5テーブルとマスターデータを明示コマンドで作成可能 |
| 本番DB | 公開環境のデータ保存 | Supabase PostgreSQLを構築し、Issue #138時点の4テーブルとマスターデータを投入済み。Issue #139の`sessions`は未反映 |
| 認証・認可 | PostgreSQLセッションとロール・担当関係に基づく制御 | セッション認証とrole確認middlewareは実装済み。業務APIの本人・担当範囲の認可は未実装。本番反映・本番E2Eは未確認 |

## 4. API

### 4.1 実装済みAPI

| メソッド | パス | 目的 | 実装 |
| --- | --- | --- | --- |
| `GET` | `/health` | HTTPプロセスが応答できることを確認し、`200 OK` とJSON `{"status":"ok"}` を返す | Ginで実装済み |
| `POST` | `/api/auth/login` | 資格情報を照合し、セッションCookieを発行する | Ginで実装済み。本番E2Eは未確認 |
| `GET` | `/api/auth/me` | セッションから現在のユーザーを返す | 認証middlewareを含めてGinで実装済み。本番E2Eは未確認 |
| `POST` | `/api/auth/logout` | セッションを削除し、Cookieを失効させる | Ginで実装済み。本番E2Eは未確認 |

サーバー起動前にDB接続を行いますが、`GET /health` のハンドラー自体はDBへ問い合わせません。

公開環境の `https://health-bridge-p3kx.onrender.com/health` では、HTTP 200、`Content-Type: application/json`、JSON `{"status":"ok"}` を確認しています。ルート `/` が404を返すのは、ルートを実装していない現在の仕様どおりです。Renderのログに接続文字列やDBパスワードが表示されていないことも確認済みです。

ルーターは起動処理から分離しています。GinのLoggerとRecoveryを使用し、未定義ルートは `404`、panic時は `500` の安全なJSONを返します。内部エラーやSQL情報はレスポンスへ含めません。APIエラーは `error.code` と利用者向けの `error.message` を持つ共通形式で返します。

起動時に `DATABASE_URL` と `ALLOWED_ORIGINS` を必須環境変数として検証し、`PORT` は未指定時に `8080` を使用します。許可Originはカンマ区切りで明示し、ワイルドカードは受け付けません。CORS middlewareはcredentialを許可し、Cookie認証に必要なHTTPメソッドとヘッダーを対象にします。状態変更リクエストではOriginを検証します。

現在の公開環境では、Issue #138時点のCORSとヘルスチェックまで確認済みです。コード上ではcredential付きCORS、Origin検証、セッション認証を実装済みですが、これらの本番反映と本番E2Eは未確認です。業務APIと、本人・担当範囲に基づくサーバー側認可は未実装です。

HTTPサーバーはOSの終了シグナルを受けると10秒を上限にgraceful shutdownし、その後にDB接続を閉じます。起動失敗は通常停止と区別してエラーにします。

### 4.2 MVPのAPI設計（認証API以外は未実装）

| API ID | メソッド | パス | 目的 | 役割・データ範囲 | 関連機能 |
| --- | --- | --- | --- | --- | --- |
| API-02 | `POST` | `/api/auth/login` | メールアドレスとパスワードでログイン | 認証不要 | F-02 |
| API-03 | `POST` | `/api/auth/logout` | ログイン状態を終了 | ログイン済みユーザー | F-02 |
| API-04 | `POST` | `/api/employees` | employeeを作成し、実行者を担当managerに設定 | managerのみ | F-03 |
| API-05 | `GET` | `/api/employees/{employeeId}` | employeeの登録情報を取得 | managerのみ・自身の担当employeeに限定 | F-09 |
| API-06 | `PATCH` | `/api/employees/{employeeId}` | employeeの氏名・メールアドレス等を更新 | managerのみ・自身の担当employeeに限定 | F-13 |
| API-07 | `PATCH` | `/api/employees/{employeeId}/deactivate` | employeeを物理削除せず無効化 | managerのみ・自身の担当employeeに限定 | F-13 |
| API-08 | `POST` | `/api/health-records` | 出勤時または退勤時の体調を登録 | employeeのみ・本人の記録 | F-04 |
| API-09 | `GET` | `/api/health-records?period=week\|month` | 本人の体調記録を期間指定で取得 | employeeのみ・本人の記録 | F-05, F-06 |
| API-10 | `GET` | `/api/managed-employees?date=YYYY-MM-DD&filter=...` | 指定日の担当employee一覧と体調を取得 | managerのみ・自身の担当employeeかつ `is_active = true` のemployeeに限定 | F-07, F-08 |
| API-11 | `GET` | `/api/managed-employees/{employeeId}/health-records` | 担当employeeの体調履歴を取得 | managerのみ・自身の担当employeeに限定 | F-09 |
| API-12 | `GET` | `/api/managers` | 担当変更先となる有効manager一覧を取得 | managerのみ | F-10 |
| API-13 | `PATCH` | `/api/managed-employees/{employeeId}/manager` | 担当employeeを別managerへ引き継ぐ | managerのみ・現在の担当managerに限定 | F-10 |

担当関係は、作成時（API-04）、担当範囲の参照・更新（API-05〜07、10、11）、担当変更（API-12、13）にまたがります。画面に対象IDが含まれていても信用せず、API側でログインユーザーのroleと `users.manager_id` を照合する必要があります。

旧API-01のmanager一般登録はMVP対象外とし、API IDは別機能へ再利用しません。managerは実装済みの管理用CLIで作成し、パスワードをbcryptでハッシュ化して保存します。既存のmanager登録画面はモックUIとして残しますが、実APIへは接続しません。組織用招待コードとメール認証は将来拡張です。

API-14「チーム体調傾向」は現行MVPから除外済みのため、業務APIへ追加しません。機能IDと同様にAPI IDも別機能へ再利用しません。

### 4.3 認証・認可とAPI共通仕様（セッション認証実装済み）

#### 認証・セッション

- JWTは使用せず、PostgreSQLで管理するサーバー側セッション方式とする
- パスワードはbcryptでハッシュ化し、平文では保存しない。bcryptはパスワードに使用し、セッショントークンには使用しない
- ログイン成功時は `crypto/rand` で32バイトのランダム値を生成し、Base64 URL形式にエンコードしたセッショントークンをCookieへ格納する
- DBにはセッショントークンそのものではなくSHA-256ダイジェストを保存する
- 認証時はCookieのトークンから同じSHA-256ダイジェストを生成し、対象セッションを検索する
- セッション有効期限は24時間とする
- ログアウト時は対象セッションをDBから削除し、Cookieを失効させる
- 未認証、トークン不正、セッション無効または期限切れの場合は `401 Unauthorized` とする
- 認証時にユーザーの有効状態を確認し、無効化済みユーザーは既存セッションが残っていても認証を許可せず `401 Unauthorized` とする

`sessions`テーブル、bcrypt処理を行う管理用ユーザー作成CLI、ログイン・ログアウト・現在ユーザー取得、セッションの発行と検証、Cookie、認証middlewareは実装済みです。公開環境への反映と本番E2Eは未確認です。

#### Cookie・CORS・CSRF

- Cookie名は `health_bridge_session` とし、`HttpOnly`、`Path=/api` を設定する
- Cookieの発行と失効には同じCookie名と `Path=/api` を使用する
- 本番環境は `Secure=true`、`SameSite=None` とする
- ローカル環境はHTTPで検証できるCookie設定へ切り替え、Cookie属性を環境ごとに設定できるようにする
- フロントエンドは認証が必要なリクエストで `credentials: "include"` を使用する
- credential付きCORSではワイルドカードOriginを使用せず、環境変数で明示した許可Originだけを `Access-Control-Allow-Origin` に設定する
- Cookieを送受信できるよう、許可Originへの応答だけに `Access-Control-Allow-Credentials: true` を設定する
- クロスオリジンの状態変更リクエストで発生する `OPTIONS` preflightへ応答し、`Access-Control-Allow-Methods` に `GET`、`POST`、`PUT`、`PATCH`、`DELETE`、`OPTIONS` を設定する
- JSONリクエストでは `Content-Type: application/json` を使用し、`Access-Control-Allow-Headers` に `Content-Type` を設定する
- 将来ほかの非単純ヘッダーを使用する場合は、`Access-Control-Allow-Headers` へ対象ヘッダーを明示的に追加する
- CSRF対策として、`POST`、`PUT`、`PATCH`、`DELETE` の状態変更リクエストでは `Origin` が許可Originと一致することを検証する
- `GET`は参照専用とし、データを変更しない

#### 認可

- 認証済みユーザーのIDとroleはセッションから特定し、クライアントが送信したrole、user ID、manager IDを認可根拠として信用しない
- employeeは本人の情報と体調記録だけを参照・操作できる
- managerはDB上で現在担当しているemployeeだけを参照・操作できる
- API-13の引き継ぎ先は、DB上で `role = manager` かつ `is_active = true` のユーザーに限定し、API側で検証する
- 担当managerを変更した後は、旧managerによる参照・操作を `403 Forbidden` とする
- 認証済みでもroleまたは担当範囲が許可されない場合は `403 Forbidden` とする
- 無効化済みemployeeは通常の担当一覧から除外し、無効化前のアカウント情報と体調記録は保持する

#### 日付・時刻

- 業務上の「今日」と日付範囲は `Asia/Tokyo` を基準に判定する
- 体調記録の `record_date` と、週・月・指定日の期間条件は同じ日付境界を使用する

#### 共通エラーレスポンス

業務APIのエラーは、原則として次の形式で返します。HTTPステータスでエラー種別を示し、`code`はフロントエンドの分岐、`message`は利用者向け表示に使用します。

```json
{
  "error": {
    "code": "invalid_request",
    "message": "リクエスト内容を確認してください"
  }
}
```

利用者向けレスポンスには内部エラーの詳細を含めません。サーバーログには調査に必要な情報を残しますが、パスワード、セッショントークン、DB接続情報などの秘密情報、リクエスト・レスポンス本文、体調情報、コメント、メールアドレス、氏名などの健康情報・個人情報は出力しません。調査には内部IDなど、業務データそのものを含まない識別情報を使用します。

#### API共通表現

- APIのリクエスト・レスポンスではIDを文字列として表現する
- パスパラメータも文字列として受け取り、バックエンドで妥当な整数IDへ変換・検証してからDB処理に使用する
- DBの主キーは整数とし、APIの文字列IDとは境界で明示的に変換する
- conditionコードは `excellent`、`good`、`normal`、`caution`、`bad` とする
- timingは `clockIn`、`clockOut` とする。DBの識別子はsnake_caseを維持するが、`health_records.timing` に保存する値も同じコードを使用する
- managerの表示が必要なレスポンスでは、少なくとも文字列の `id` と `name` を持つオブジェクトを返す

```json
{
  "id": "1",
  "name": "鈴木 花子"
}
```

モック固有の `new-...` 形式のIDや固定ユーザーID `"1"` は、実API接続後には使用しません。認証済みユーザーのIDとrole、現在の担当managerはサーバー側で特定し、クライアントから送信されたroleやmanager IDを認証・認可の根拠にしません。

### 4.4 フロントエンドとAPI・DB設計の整合方針

現在のフロントエンドで使用する型・値を維持しつつ、モック固有の表現を本番設計へ持ち込まないよう、次の方針で統一します。

| 対象 | 確定した設計 | モックからの置き換え方針 |
| --- | --- | --- | --- |
| 最上位の体調コード | API・DBとも `excellent` | フロントエンドの既存コードをそのまま使用する |
| 入力タイミング | API・DBの保存値とも `clockIn` / `clockOut` | フロントエンドの既存コードをそのまま使用する |
| ID | DBは整数、APIは文字列 | `new-...` を廃止し、APIが返すIDへ置き換える |
| 担当manager | DBは `users.manager_id`、表示APIは `{ id, name }` | 氏名文字列だけの関連付けを廃止し、IDで識別する |

次の事項は本対応では確定せず、関連実装前に別途判断します。これらを決める場合も、上記の確定方針は変更しません。

- 期限切れセッションの削除方法
- ローカル環境の `SameSite` 設定とVercel Preview URLを許可Originへ登録する運用
- 無効化済みユーザーは既存セッションでも認証を許可しない。セッション行を削除するタイミングは関連実装時に決定する
- 無効化後の履歴参照、担当変更履歴、同時更新の扱い
- 業務API実装後の本番デモデータ投入方法
- manager自身の編集・削除UIと、到達不能な従業員検索・追加ダイアログのMVP上の扱い

## 5. DB設計（5テーブル実装済み）

DB設計v2を第2回レビュー後のMVPと照合し、次の5テーブルをGORMモデルとPostgreSQLスキーマとして実装しています。チーム集計用テーブルと担当変更履歴は追加していません。

### 5.1 テーブル概要

| テーブル | 主なカラム | 主な制約・役割 |
| --- | --- | --- |
| `roles` | `id`, `name` | `name` は一意。初期値は `manager`, `employee` |
| `users` | `id`, `name`, `email`, `password_hash`, `role_id`, `manager_id`, `is_active`, `deactivated_at`, timestamps | `email` は一意。`role_id` → `roles.id`。`manager_id` → `users.id` の自己参照で現在の担当managerを表す。氏名文字列では関連付けない |
| `sessions` | `id`, `user_id`, `token_digest`, `expires_at`, `created_at` | `user_id` → `users.id`。SHA-256ダイジェストを32バイトの`bytea`で保存し、一意制約を設定。平文トークンは保存しない |
| `conditions` | `id`, `code`, `name`, `score`, `display_order` | `code`, `name`, `score`, `display_order` はそれぞれ一意。5段階の固定値 |
| `health_records` | `id`, `employee_id`, `record_date`, `timing`, `condition_id`, `comment`, `created_at` | `employee_id` → `users.id`、`condition_id` → `conditions.id`。同一employee・日付・timingを一意にする |

### 5.2 重要な制約と業務ルール

- PostgreSQLの各テーブルの主キーは整数とする。GORMモデルでは `uint`、PostgreSQLでは `bigint` として実装し、APIではIDを文字列として返してDBアクセス前に変換・検証する
- `users.manager_id` はnullableな自己参照とし、employeeの現在の担当managerを示す。manager自身は `NULL` とする
- `sessions.token_digest` は32バイトの`bytea`、NOT NULL、一意とし、一意indexをトークン検索にも使用する
- `sessions.user_id`と`sessions.expires_at`にindexを設定し、ユーザー単位のセッション操作と期限切れ検索に使用する
- sessionが参照しているuserの物理削除は拒否する。userの無効化ではsessionを保持し、認証処理で`is_active`を確認する
- `health_records` に `UNIQUE (employee_id, record_date, timing)` を設定する
- `timing` は `CHECK (timing IN ('clockIn', 'clockOut'))` とする
- コメントは必須とし、`VARCHAR(500) NOT NULL` とする。空文字および空白文字だけのコメントは `CHECK` 制約で拒否する
- 「未入力」は `conditions` に追加せず、該当する `health_records` が存在しない状態として判定する
- 「未入力」絞り込みは、指定日の `clockIn` が存在しないemployeeを対象とする仮仕様
- 担当変更時は `users.manager_id` を更新し、employeeアカウントと過去の体調記録を保持する
- employeeの無効化は `is_active = false` とし、物理削除しない

### 5.3 conditions初期データ

| code | 画面表示 | score | display_order |
| --- | --- | ---: | ---: |
| `excellent` | とても良好 | 2 | 1 |
| `good` | 良好 | 1 | 2 |
| `normal` | 普通 | 0 | 3 |
| `caution` | 注意 | -1 | 4 |
| `bad` | 悪化 | -2 | 5 |

`roles`には `manager`、`employee` を投入します。マスターデータは一意キーを使ったUPSERTで投入し、複数回実行しても重複しません。デモユーザーとデモ体調記録は投入しません。

### 5.4 マイグレーション

HTTPサーバー起動時には自動マイグレーションを行いません。Docker ComposeでDBを起動した後、次の専用コマンドを明示的に実行します。

```bash
docker compose run --rm backend /usr/local/bin/migrate
```

このコマンドはGORMのマイグレーションとマスターデータ投入を同じトランザクションで実行します。テーブル、外部キー、NOT NULL、一意、CHECK、文字数の各制約はPostgreSQL上の統合テストで確認します。外部キー列には索引を設定し、`health_records`の複合一意索引はemployee別・日付順の取得にも利用できる並びにしています。

Issue #138では、本番のSupabase PostgreSQLへ既存4テーブルのマイグレーションを2回実行し、どちらも成功しています。Issue #139で追加した`sessions`は本番DBへ自動適用せず、管理用CLIも本番で自動実行しません。

## 6. ER図（実装済み）

実装した5テーブルの関係を示します。以下のMermaid図を、現在の実装に対応するER図とします。

```mermaid
erDiagram
    ROLES ||--o{ USERS : "role_id"
    USERS o|--o{ USERS : "manager_id"
    USERS ||--o{ SESSIONS : "user_id"
    USERS ||--o{ HEALTH_RECORDS : "employee_id"
    CONDITIONS ||--o{ HEALTH_RECORDS : "condition_id"

    ROLES {
        bigint id PK
        varchar name UK
    }

    USERS {
        bigint id PK
        varchar name
        varchar email UK
        varchar password_hash
        bigint role_id FK
        bigint manager_id FK "nullable; users.id"
        boolean is_active
        timestamptz deactivated_at "nullable"
        timestamptz created_at
        timestamptz updated_at
    }

    SESSIONS {
        bigint id PK
        bigint user_id FK
        bytea token_digest UK "SHA-256; 32 bytes"
        timestamptz expires_at
        timestamptz created_at
    }

    CONDITIONS {
        bigint id PK
        varchar code UK
        varchar name UK
        smallint score UK
        smallint display_order UK
    }

    HEALTH_RECORDS {
        bigint id PK
        bigint employee_id FK
        date record_date
        varchar timing "clockIn or clockOut"
        bigint condition_id FK
        varchar comment
        timestamptz created_at
    }
```

Mermaidの属性表現では複合一意制約を表しにくいため、`HEALTH_RECORDS` の `UNIQUE (employee_id, record_date, timing)` は本文とテーブル定義を正とします。

## 7. 今後の実装順序

Render / Supabaseによる公開基盤の整備は完了しています。今後は次の順序で業務機能を実装します。

1. 確定したcondition・timingコード、ID境界、manager表現に基づいてDTOと入力検証を定義する
2. Gin上へMVP業務APIを画面単位で実装する
3. 業務APIへ本人・担当範囲に基づく認可を適用する
4. フロントエンドのモック状態を業務API接続へ置き換える
5. API・DB・画面を通したテストとCIを整備する

機能・設計変更時はREADMEと関連docsも同じIssueで更新します。独立した設計変更や、本書に残した未確定事項・未定義UI等の不整合は、アプリ実装とは分けてIssueで追跡します。
