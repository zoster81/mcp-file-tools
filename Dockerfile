FROM golang:1.26.5-alpine3.24 AS builder

ARG VERSION=dev
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X github.com/zoster81/mcp-file-tools/filetoolsserver.Version=${VERSION}" \
    -o /out/mcp-file-tools \
    ./cmd/mcp-file-tools

FROM alpine:3.24.1

RUN apk add --no-cache ca-certificates \
    && addgroup -S -g 10001 mcp \
    && adduser -S -D -H -u 10001 -G mcp mcp \
    && mkdir -p /data /tmp/mcp-file-tools \
    && chown -R 10001:10001 /data /tmp/mcp-file-tools

COPY --from=builder --chown=10001:10001 /out/mcp-file-tools /usr/local/bin/mcp-file-tools

USER 10001:10001
WORKDIR /data
ENV HOME=/tmp/mcp-file-tools \
    TMPDIR=/tmp/mcp-file-tools

STOPSIGNAL SIGTERM
ENTRYPOINT ["/usr/local/bin/mcp-file-tools"]
