FROM golang:1.22-alpine AS builder

RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.aliyun.com|g' /etc/apk/repositories \
    && apk add --no-cache gcc musl-dev sqlite-dev libpcap-dev

WORKDIR /app
COPY go.mod go.sum* ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go env -w GOSUMDB=off && go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o server ./cmd/server

FROM alpine:3.20
RUN sed -i 's|dl-cdn.alpinelinux.org|mirrors.aliyun.com|g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates sqlite-libs libpcap curl
WORKDIR /app
COPY --from=builder /app/server ./server

EXPOSE 8080
CMD ["./server"]