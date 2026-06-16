FROM golang:1.26.4-alpine AS builder

WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN printf '%s\n' \
      'https://mirrors.aliyun.com/alpine/v3.24/main' \
      'https://mirrors.aliyun.com/alpine/v3.24/community' \
      > /etc/apk/repositories \
    && apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,direct
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/komari .

FROM alpine:3.21

WORKDIR /app

ARG TARGETARCH=amd64

RUN printf '%s\n' \
      'https://mirrors.aliyun.com/alpine/v3.21/main' \
      'https://mirrors.aliyun.com/alpine/v3.21/community' \
      > /etc/apk/repositories \
    && apk add --no-cache ca-certificates curl tzdata

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) cloudflared_arch="amd64" ;; \
      386) cloudflared_arch="386" ;; \
      arm64) cloudflared_arch="arm64" ;; \
      arm) cloudflared_arch="arm" ;; \
      *) echo "Unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${cloudflared_arch}" -o /usr/local/bin/cloudflared; \
    chmod +x /usr/local/bin/cloudflared

COPY --from=builder /out/komari /app/komari

RUN chmod +x /app/komari

ENV GIN_MODE=release
ENV KOMARI_DB_TYPE=sqlite
ENV KOMARI_DB_FILE=/app/data/komari.db
ENV KOMARI_DB_HOST=localhost
ENV KOMARI_DB_PORT=3306
ENV KOMARI_DB_USER=root
ENV KOMARI_DB_PASS=
ENV KOMARI_DB_NAME=komari
ENV KOMARI_LISTEN=0.0.0.0:25774

EXPOSE 25774

CMD ["/app/komari", "server"]
