# HealthBridge ローカル開発手順

## 1. この文書について

本書は、HealthBridgeをローカルで起動・検証する開発者向けの手順です。アプリの目的と公開デモは[README](../README.md)、機能要件は[要件定義](requirements.md)、技術設計は[システム構成・設計](architecture.md)を参照してください。

## 2. 前提ツール

- Git
- Docker
- Docker Compose

Dockerを使用しない場合は、フロントエンドにNode.js 20、バックエンドにGo 1.26.5、データベースにPostgreSQL 16が必要です。バージョンは現在のDockerfileと `go.mod` に基づきます。

## 3. 初期設定

リポジトリを取得し、ルートの環境変数ファイルを作成します。

```bash
git clone https://github.com/atamu2463/health-bridge.git
cd health-bridge
cp .default.env .env
```

`.env` にはローカル開発専用の値を設定します。

```env
POSTGRES_USER=任意のユーザー名
POSTGRES_PASSWORD=任意のパスワード
POSTGRES_DB=health_bridge
BACKEND_PORT=8080
ALLOWED_ORIGINS=http://localhost:3000
```

フロントエンドがローカルAPIを呼び出せるよう、`frontend/.env.local` を作成します。

```env
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

`.env` と `frontend/.env.local` に本番値や共有すべきでない認証情報を記載・コミットしないでください。

## 4. 起動とmigration

```bash
docker compose up -d --build
docker compose exec backend /usr/local/bin/migrate
```

migrationは `roles`、`users`、`sessions`、`conditions`、`health_records` の5テーブルと制約を作成し、`roles` と `conditions` のマスターデータを投入します。複数回実行してもマスターデータは重複しません。HTTPサーバーの起動時には自動実行されません。

起動確認先は次のとおりです。

- フロントエンド：<http://localhost:3000>
- Health Check：<http://localhost:8080/health>

`GET /health` はHTTPプロセスの応答確認用で、DBへの疎通確認は行いません。

## 5. 管理用ユーザー作成CLI

managerを先に作成します。

```bash
docker compose exec backend /usr/local/bin/create-user \
  -name "任意の管理者名" \
  -email "manager@example.com" \
  -role manager
```

employeeには、有効なmanagerのメールアドレスを指定します。

```bash
docker compose exec backend /usr/local/bin/create-user \
  -name "任意の従業員名" \
  -email "employee@example.com" \
  -role employee \
  -manager-email "manager@example.com"
```

パスワードは実行後に2回入力します。CLI引数や環境変数では受け取らず、DBにはbcryptハッシュを保存します。migrationやコンテナ起動時にデモユーザーを自動作成しません。

## 6. 検証

フロントエンド：

```bash
cd frontend
npm run format:check
npm run lint
npm run build
cd ..
```

バックエンド：

```bash
cd backend
gofmt -l .
go test ./...
go vet ./...
cd ..
```

PostgreSQLを利用する統合テストは `TEST_DATABASE_URL` が設定されている場合に実行されます。実行先には、テスト専用で破棄可能なデータベースを使用してください。

Docker Compose設定と差分の確認：

```bash
docker compose --env-file .env config --quiet
git diff --check
git status --short
```

## 7. 停止

```bash
docker compose down
```

PostgreSQLのデータはDocker volumeに残ります。`docker compose down -v` はvolumeも削除するため、データを破棄してよい場合だけ使用してください。

## 8. ローカル環境での注意

- `.env` の `ALLOWED_ORIGINS` と、ブラウザで開くフロントエンドOriginを一致させます。
- `frontend/.env.local` を変更した後は、フロントエンドを再起動します。
- バックエンドコードを変更した後は、必要に応じて `docker compose up -d --build` でイメージを更新します。
- 通常の起動でmigrationやユーザー作成は自動実行されません。
- 業務画面の入力・編集結果はブラウザ内のモック状態であり、再読み込みすると初期化されます。
