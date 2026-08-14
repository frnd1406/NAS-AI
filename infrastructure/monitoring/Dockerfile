FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/monitoring-agent main.go

FROM alpine:3.19
RUN set -e; \
    echo "https://mirror1.hs-esslingen.de/pub/Mirrors/alpine/v3.19/main" > /etc/apk/repositories; \
    echo "https://mirror1.hs-esslingen.de/pub/Mirrors/alpine/v3.19/community" >> /etc/apk/repositories; \
    for i in 1 2 3; do \
      apk update && apk add --no-cache ca-certificates && break || { echo \"Retry $i/3 ...\"; sleep 2; }; \
    done; \
    rm -rf /var/cache/apk/*
COPY --from=builder /app/monitoring-agent /usr/local/bin/monitoring-agent
ENTRYPOINT ["monitoring-agent"]
