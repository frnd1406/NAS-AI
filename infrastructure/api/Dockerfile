FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN apk add --no-cache git && go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/api ./src/main.go

FROM alpine:3.19
# su-exec = winziges Tool zum Privilegien-Drop (root -> nobody) im entrypoint
RUN apk add --no-cache ca-certificates su-exec
WORKDIR /app
COPY --from=builder /app/bin/api /app/api
# Create mount points; ownership set at runtime entrypoint
RUN mkdir -p /mnt/data /mnt/backups /app/dev-certs

COPY src/entrypoint.sh /entrypoint.sh
# CRLF-robust: bei Windows-Checkout hat entrypoint.sh evtl. \r -> Shebang bricht
RUN sed -i 's/\r$//' /entrypoint.sh && chmod +x /entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
