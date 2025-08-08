#!/bin/bash

# Test deployment script for CI/CD validation
# This script tests the docker-compose setup locally before deployment

set -e

echo "🧪 Testing Celebrum AI deployment..."

# Function to cleanup on exit
cleanup() {
    echo "🧹 Cleaning up..."
    docker compose -f docker-compose.ci.yml down -v || true
    docker system prune -f || true
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Build and start services
echo "🏗️  Building and starting services..."
docker compose -f docker-compose.ci.yml build --no-cache
docker compose -f docker-compose.ci.yml up -d

# Wait for services to be healthy
echo "⏳ Waiting for services to be healthy..."
max_attempts=60
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if docker compose -f docker-compose.ci.yml ps | grep -q "(healthy)"; then
        healthy_count=$(docker compose -f docker-compose.ci.yml ps | grep -c "(healthy)" || echo "0")
        total_services=4  # postgres, redis, ccxt-service, app
        
        if [ "$healthy_count" -eq "$total_services" ]; then
            echo "✅ All services are healthy!"
            break
        else
            echo "⏳ $healthy_count/$total_services services healthy, waiting..."
        fi
    else
        echo "⏳ Waiting for services to start..."
    fi
    
    sleep 5
    attempt=$((attempt + 1))
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ Services failed to become healthy within timeout"
    echo "📋 Service status:"
    docker compose -f docker-compose.ci.yml ps
    echo "📋 Service logs:"
    docker compose -f docker-compose.ci.yml logs
    exit 1
fi

# Test health endpoint
echo "🔍 Testing health endpoint..."
if docker exec celebrum-app-ci wget -q --spider http://localhost:8080/health; then
    echo "✅ Health endpoint is responding"
else
    echo "❌ Health endpoint is not responding"
    docker compose -f docker-compose.ci.yml logs app
    exit 1
fi

# Test database connection
echo "🔍 Testing database connection..."
if docker exec celebrum-postgres-ci pg_isready -U postgres; then
    echo "✅ Database is ready"
else
    echo "❌ Database is not ready"
    docker compose -f docker-compose.ci.yml logs postgres
    exit 1
fi

# Test Redis connection
echo "🔍 Testing Redis connection..."
if docker exec celebrum-redis-ci redis-cli ping | grep -q "PONG"; then
    echo "✅ Redis is responding"
else
    echo "❌ Redis is not responding"
    docker compose -f docker-compose.ci.yml logs redis
    exit 1
fi

echo "🎉 All tests passed! Deployment is ready."