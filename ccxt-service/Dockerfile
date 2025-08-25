# Use Bun image for better performance and consistency
FROM oven/bun:1-alpine as base

# Set working directory
WORKDIR /app

# Copy package files
COPY package.json bun.lock* ./

# Install dependencies including dev dependencies for build
RUN bun install --frozen-lockfile

# Copy source code
COPY . .

# Build TypeScript
RUN bun run build:bun

# Expose port
EXPOSE 3001

# Install curl for health check
RUN apk add --no-cache curl

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3001/health || exit 1

# Start the application with Bun runtime
CMD ["bun", "run", "start:bun"]