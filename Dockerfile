# syntax=docker/dockerfile:1

# --- frontend ---
FROM node:22-bookworm-slim AS frontend
WORKDIR /src/frontend
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# --- backend ---
FROM golang:1.24-bookworm AS backend
WORKDIR /src/backend
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend /src/frontend/out/ ./internal/web/dist/
RUN go build -trimpath -ldflags="-s -w -X github.com/zninggo/grokforge/internal/buildinfo.Version=${VERSION:-0.1.0}" \
    -o /out/grokforge ./cmd/grokforge

# --- runtime ---
FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata curl \
 && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /out/grokforge /app/grokforge
ENV GROKFORGE_HTTP_ADDR=:17890 \
    TZ=Asia/Shanghai
EXPOSE 17890
HEALTHCHECK --interval=10s --timeout=5s --start-period=15s --retries=5 \
  CMD curl -fsS http://127.0.0.1:17890/healthz >/dev/null || exit 1
USER nobody
ENTRYPOINT ["/app/grokforge"]
