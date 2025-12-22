FROM oven/bun:1 AS builder
WORKDIR /app

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
RUN bun run build:bun
RUN bun install --frozen-lockfile --production

FROM oven/bun:1 AS production

# Create non-root user
RUN groupadd -g 1001 appgroup && \
    useradd -u 1001 -g appgroup -m -s /bin/bash appuser

WORKDIR /app

COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/package.json .

USER appuser

ENV TELEGRAM_PORT=3002

EXPOSE 3002

CMD ["bun", "run", "dist/index.js"]
