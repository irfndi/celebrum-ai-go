#!/bin/bash

# Test core services only (postgres, redis, app) for CI/CD validation
# This script tests the essential services without CCXT to isolate issues

set -e

echo "🧪 Testing Celebrum AI core services..."

# Function to cleanup on exit
cleanup() {
    echo "🧹 Cleaning up..."
    docker-compose -f docker-compose.ci.yml down -v postgres redis app || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Start all services (postgres, redis, ccxt-service, app)
echo "🏗️  Starting all services..."
docker-compose -f docker-compose.ci.yml up -d

# Wait for services to be healthy
echo "⏳ Waiting for core services to be healthy..."
max_attempts=120  # Increased timeout to 10 minutes
attempt=0

while [ $attempt -lt $max_attempts ]; do
    postgres_health=$(docker inspect celebrum-postgres-ci --format='{{.State.Health.Status}}' 2>/dev/null || echo "starting")
    redis_health=$(docker inspect celebrum-redis-ci --format='{{.State.Health.Status}}' 2>/dev/null || echo "starting")
    ccxt_health=$(docker inspect celebrum-ccxt-ci --format='{{.State.Health.Status}}' 2>/dev/null || echo "starting")
    app_health=$(docker inspect celebrum-app-ci --format='{{.State.Health.Status}}' 2>/dev/null || echo "starting")
    
    echo "📊 Status: postgres=$postgres_health, redis=$redis_health, ccxt=$ccxt_health, app=$app_health"
    
    if [ "$postgres_health" = "healthy" ] && [ "$redis_health" = "healthy" ] && [ "$ccxt_health" = "healthy" ] && [ "$app_health" = "healthy" ]; then
        echo "✅ All core services are healthy!"
        break
    fi
    
    sleep 5
    attempt=$((attempt + 1))
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ Services failed to become healthy within timeout"
    echo "📋 Service status:"
    docker-compose -f docker-compose.ci.yml ps
    echo "📋 Service logs:"
    docker-compose -f docker-compose.ci.yml logs
    exit 1
fi

# Test health endpoint
echo "🔍 Testing health endpoint..."
if docker exec celebrum-app-ci wget --no-verbose --tries=1 -O /dev/null http://localhost:8080/health; then
    echo "✅ Health endpoint is responding"
else
    echo "❌ Health endpoint is not responding"
    docker-compose -f docker-compose.ci.yml logs app
    exit 1
fi

# Test database connection
echo "🔍 Testing database connection..."
if docker exec celebrum-postgres-ci pg_isready -U postgres; then
    echo "✅ Database is ready"
else
    echo "❌ Database is not ready"
    docker-compose -f docker-compose.ci.yml logs postgres
    exit 1
fi

# Test Redis connection
echo "🔍 Testing Redis connection..."
if docker exec celebrum-redis-ci redis-cli ping | grep -q "PONG"; then
    echo "✅ Redis is responding"
else
    echo "❌ Redis is not responding"
    docker-compose -f docker-compose.ci.yml logs redis
    exit 1
fi

echo "🎉 All services tests passed!"