FROM golang:1.22 AS builder

RUN apt-get update && apt-get install -y gcc sqlite3 libsqlite3-dev libpcap0.8-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum* ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go env -w GOSUMDB=off && go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates sqlite3 libpcap0.8 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /app/server ./server

EXPOSE 8080
CMD ["./server"]