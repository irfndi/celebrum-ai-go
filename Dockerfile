# Build stage
FROM golang:1.25-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o main ./cmd/server

# Production stage
FROM alpine:latest AS production

# Install only essential runtime dependencies
RUN apk --no-cache add ca-certificates tzdata wget curl

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/main .

# Copy configuration files
COPY --from=builder /app/configs ./configs
COPY --from=builder /app/config.yaml ./config.yaml

# Copy scripts directory
COPY --from=builder /app/scripts ./scripts

# Change ownership to non-root user
RUN chown -R appuser:appgroup /root

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]

# Binary extraction stage - for artifact deployment
FROM scratch AS binary

# Copy only the essential files for deployment
COPY --from=builder /app/main .
COPY --from=builder /app/configs ./configs/
COPY --from=builder /app/config.yaml ./config.yaml

# Set proper permissions for binary
RUN chmod +x main

# Debug stage (includes debugging tools)
FROM production AS debug

# Switch to root to install debugging tools
USER root

# Install debugging and monitoring tools for development/debugging
RUN apk --no-cache add procps htop strace lsof tcpdump

# Switch back to non-root user for debug stage
USER appuser