FROM golang:1.26.5-alpine

WORKDIR /backend

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 起動時に再コンパイルせず実行できるよう、Goアプリをバイナリとしてビルドする。
RUN go build -o /usr/local/bin/health-bridge .

# HTTPサーバーとは分けて明示的に実行できるよう、マイグレーション用バイナリを作成する。
RUN go build -o /usr/local/bin/migrate ./cmd/migrate

EXPOSE 8080

# ビルド済みのGoアプリを直接起動し、Dockerの終了シグナルを受け取れるようにする。
CMD ["/usr/local/bin/health-bridge"]
