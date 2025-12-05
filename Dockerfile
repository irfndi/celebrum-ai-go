# ==========================================
# Stage 1: Go Builder
# ==========================================
FROM golang:1.25.5 AS go-builder

# Install git and ca-certificates
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/server && \
    chmod +x main

# ==========================================
# Stage 2: CCXT Service Builder (Bun)
# ==========================================
FROM oven/bun:1 AS ccxt-builder
WORKDIR /app

# Copy ccxt-service files
COPY services/ccxt/package.json services/ccxt/bun.lock ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy source code
COPY services/ccxt/ .

# Build
RUN bun run build

# Prune dev dependencies
RUN bun install --frozen-lockfile --production

# ==========================================
# Stage 3: Unified Runtime (Go + Bun)
# ==========================================
FROM oven/bun:1 AS production

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata wget curl bash postgresql-client && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -m -s /bin/bash appuser

WORKDIR /app

# Copy Go binary and configs
COPY --from=go-builder /app/main .
COPY --from=go-builder /app/config.yml .
COPY --from=go-builder /app/scripts ./scripts
COPY --from=go-builder /app/database ./database

# Copy CCXT service
WORKDIR /app/ccxt
COPY --from=ccxt-builder /app/node_modules ./node_modules
COPY --from=ccxt-builder /app/dist ./dist
COPY --from=ccxt-builder /app/package.json .
COPY --from=ccxt-builder /app/tracing-utils.js ./dist/tracing.js

WORKDIR /app

# Permissions
RUN chown -R appuser:appgroup /app && \
    chmod +x scripts/entrypoint.sh

USER appuser

# Expose ports (Go: 8080, Bun: 3000)
EXPOSE 8080 3000

# Healthcheck (checks Go app)
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./scripts/entrypoint.sh"]
