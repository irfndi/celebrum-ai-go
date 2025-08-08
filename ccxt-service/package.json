FROM oven/bun:1-alpine

# Set working directory
WORKDIR /app

# Install dependencies for native modules (still needed for some CCXT dependencies)
RUN apk add --no-cache python3 make g++

# Copy package files
COPY package*.json ./
COPY tsconfig.json ./

# Install dependencies
RUN bun install --frozen-lockfile

# Copy application code
COPY . .

# Create non-root user
RUN addgroup -g 1001 -S nodejs
RUN adduser -S ccxt -u 1001

# Change ownership
RUN chown -R ccxt:nodejs /app
USER ccxt

# Expose port
EXPOSE 3001

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:3001/health || exit 1

# Start the application
CMD ["bun", "run", "index.ts"]