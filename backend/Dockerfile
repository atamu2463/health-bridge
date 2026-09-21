FROM golang:1.26.5-alpine

WORKDIR /backend

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /usr/local/bin/health-bridge .

EXPOSE 8080

CMD ["/usr/local/bin/health-bridge"]
