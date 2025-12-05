# ==========================================
# Stage 1: Go Builder
# ==========================================
FROM golang:1.25-alpine AS go-builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates tzdata

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
# Base Image with Bun (shared)
# ==========================================
FROM alpine:3.19 AS bun-base
RUN apk add --no-cache bash curl unzip ca-certificates && \
    curl -fsSL https://bun.sh/install | bash -s "bun-v1.3.3" && \
    ln -s /root/.bun/bin/bun /usr/local/bin/bun

# Set PATH to include bun
ENV PATH="/root/.bun/bin:${PATH}"

# ==========================================
# Stage 2: CCXT Service Builder (Bun)
# ==========================================
FROM bun-base AS ccxt-builder
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
FROM bun-base AS production
RUN apk add --no-cache tzdata wget postgresql-client

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

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
