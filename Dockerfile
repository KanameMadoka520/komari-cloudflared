FROM node:22-alpine AS frontend
WORKDIR /src/frontend/komari-web
COPY frontend/komari-web/package*.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/komari-web/ ./
RUN npm run build

FROM golang:1.27-alpine AS builder
WORKDIR /src
RUN apk add --no-cache gcc musl-dev bash tar zstd
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /src/frontend/komari-web/dist ./frontend/komari-web/dist
RUN bash scripts/pack-frontend.sh
ARG VERSION=1.5.1-main.9812acf-cloudflared.1
ARG BUILD_HASH=unknown
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -trimpath \
    -ldflags "-s -w -extldflags=-Wl,--strip-all -X github.com/komari-monitor/komari/utils.CurrentVersion=${VERSION} -X github.com/komari-monitor/komari/utils.VersionHash=${BUILD_HASH}" \
    -o /out/komari .

FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates curl tzdata
ARG TARGETARCH=amd64
ARG CLOUDFLARED_VERSION=2026.6.0
RUN case "${TARGETARCH}" in amd64|386|arm64|arm) ;; *) exit 1 ;; esac \
    && curl -fsSL "https://github.com/cloudflare/cloudflared/releases/download/${CLOUDFLARED_VERSION}/cloudflared-linux-${TARGETARCH}" -o /usr/local/bin/cloudflared \
    && chmod 755 /usr/local/bin/cloudflared
COPY --from=builder --chmod=755 /out/komari /app/komari
ENV GIN_MODE=release \
    KOMARI_DB_TYPE=sqlite \
    KOMARI_DB_FILE=/app/data/komari.db \
    KOMARI_LISTEN=0.0.0.0:25774 \
    GODEBUG=disablethp=1
EXPOSE 25774
CMD ["/app/komari", "server"]
