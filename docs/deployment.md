# HealthBridge デプロイ・運用手順

## 1. この文書について

本書は、HealthBridgeの本番環境を更新・確認する際の運用情報をまとめたものです。設計上の判断は[システム構成・設計](architecture.md)、ローカルでの起動方法は[ローカル開発手順](development.md)を正とします。

実際の接続文字列、パスワード、APIキー、Cookie、セッショントークンは本書やログへ記録しません。

## 2. 本番構成

| 対象 | サービス | 役割 |
| --- | --- | --- |
| フロントエンド | Vercel | Next.jsアプリを公開し、Renderの認証APIを呼び出す |
| バックエンド | Render Docker Web Service | Go / Gin APIを実行する |
| データベース | Supabase PostgreSQL | ユーザー、セッション、将来の業務データを保存する |

- 公開フロントエンド：<https://health-bridge-management.vercel.app/>
- Health Check：<https://health-bridge-p3kx.onrender.com/health>
- Renderは `main` をデプロイ対象とし、Root Directoryは `backend`、Health Check Pathは `/health`、Auto-Deployは無効です。
- `backend/Dockerfile` からHTTPサーバー、migration、create-userの各バイナリをビルドし、DockerfileのCMDを使用します。
- RenderからSupabaseのSession poolerへTLS必須で接続します。
- Supabase Auth、Data API、`supabase-js` は使用しません。

## 3. 環境変数

値は各サービスの安全な設定画面で管理し、リポジトリへ追加しません。

### Vercel

| 変数 | 目的 |
| --- | --- |
| `NEXT_PUBLIC_API_BASE_URL` | Render上のバックエンドURL |

Production環境に設定済みです。変更後は対象デプロイへ反映されていることを確認します。

### Render

| 変数 | 目的 |
| --- | --- |
| `DATABASE_URL` | Supabase PostgreSQLの接続文字列 |
| `ALLOWED_ORIGINS` | credential付きCORSとOrigin検証で許可するVercel Origin |
| `COOKIE_SECURE` | 本番Cookieを `Secure`・`SameSite=None` で発行する設定 |
| `PORT` | HTTPサーバーの待受ポート。未指定時は8080 |

Renderが設定する `RENDER=true` を利用し、信頼するクライアントIPヘッダーをRender環境に限定しています。

### Render Freeの運用上の補足

公開デモの初回待ち時間を抑えるため、リポジトリ外で管理するGoogle Apps Scriptから定期的に `GET /health` を実行しています。ただし、これは可用性を保証するものではなく、GASの実行状況やRender Freeの利用上限によって停止する可能性があります。停止時は、最初のアクセスから応答まで時間がかかる場合があります。

## 4. 本番migrationとユーザー作成

HTTPサーバー起動時にmigrationは実行しません。本番migrationは、デプロイ対象のコミットを取得したローカル環境から、Supabase PostgreSQLへ直接接続して実行します。

実行前に、現在のブランチ、コミット、作業ツリーが本番へ反映する内容と一致していることを確認します。

```bash
git switch main
git pull --ff-only origin main
git status --short

cd backend
```

`DATABASE_URL`はコマンドライン引数へ記載せず、入力内容を画面へ表示しない一時的な環境変数としてサブシェル内で渡します。サブシェル終了後、変数は親シェルへ残りません。

```bash
(
  read -rsp "Supabase DATABASE_URL: " DATABASE_URL
  echo
  export DATABASE_URL

  go run ./cmd/migrate
)
```

終了コードが0となり、次のメッセージが表示されることを確認します。

```text
マイグレーションとマスターデータ投入が完了しました
```

migrationはトランザクション内で実行され、複数回実行してもマスターデータが重複しない構成です。2026年9月24日に本番環境で2回実行し、どちらも正常終了しました。その後、manager／employeeのログインとセッション復元を本番環境で確認しています。

デモ用manager／employeeも、同じローカル環境から管理用CLIを使って手動作成します。パスワードは対話入力し、コマンドライン引数、環境変数、作業ログへ残しません。公開デモ資格情報は本番の管理用資格情報と分け、破棄・再作成可能な専用アカウントとして扱います。

## 5. デプロイ後の確認

次の順に、秘密情報や個人情報をログ・スクリーンショットへ残さず確認します。

1. Vercelのトップ、従業員ログイン、管理者ログインが表示される
2. `GET /health` がHTTP 200と `{"status":"ok"}` を返す
3. Vercel Originから認証APIへのpreflightがHTTP 204を返す
4. manager／employeeそれぞれでログインできる
5. リロード後にセッションが復元される
6. role不一致の保護画面で403画面へ遷移する
7. ログアウトでき、ログアウト後の保護画面でログイン画面へ遷移する
8. RenderログにDB接続文字列、パスワード、Cookie、トークン、メールアドレス等が出ていない

2026年9月24日に上記の主要E2E導線を本番環境で確認済みです。Issue #140と#141も本番E2E確認完了としてクローズされています。

## 6. CORS・Cookie・Originの確認

- `ALLOWED_ORIGINS` にはOriginだけを登録し、パスや末尾スラッシュ、ワイルドカードを含めません。
- 許可Originへの応答だけに `Access-Control-Allow-Origin` と `Access-Control-Allow-Credentials: true` が付くことを確認します。
- `POST`、`PUT`、`PATCH`、`DELETE` は、許可した `Origin` がない場合に拒否されることを確認します。
- 本番セッションCookieは `HttpOnly`、`Secure`、`SameSite=None`、`Path=/api` を使用します。

## 7. 運用上の注意

- RenderのFreeプランでは、スリープ後の初回アクセスに時間がかかる場合があります。
- バックエンドのルート `/` は未実装のため404が正常です。稼働確認には `/health` を使用します。
- `/health` はHTTPプロセスの確認用で、DB疎通を保証しません。認証E2Eも併せて確認します。
- 業務画面はモック中心です。認証が成功しても、体調記録や従業員管理がDBへ保存されたことにはなりません。
- 本番設定を変更した場合は、READMEと設計文書の「確認済み」記述も実際の結果に合わせて更新します。
