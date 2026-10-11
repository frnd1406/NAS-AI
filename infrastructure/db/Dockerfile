# PostgreSQL + pgvector with patched OS packages and a freshly built gosu.
# The upstream image ships Debian packages and a gosu binary (built with an old
# Go toolchain) that lag behind security fixes; this overlay refreshes them
# without touching Postgres or pgvector themselves.
FROM golang:1.26.9-alpine AS gosu
ENV CGO_ENABLED=0 GOFLAGS=-trimpath
RUN go install -ldflags '-s -w' github.com/tianon/gosu@1.19 \
 && cp "$(go env GOPATH)/bin/gosu" /gosu

FROM pgvector/pgvector:0.8.7-pg16
COPY --from=gosu /gosu /usr/local/bin/gosu
RUN apt-get update \
 && apt-get upgrade -y --no-install-recommends \
 && rm -rf /var/lib/apt/lists/* \
 && gosu nobody true
