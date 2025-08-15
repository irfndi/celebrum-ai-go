#!/bin/bash

# Load environment variables
if [ -f .env ]; then
    export $(grep -v '^#' .env | xargs)
fi

# Database connection parameters
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_NAME=${DB_NAME:-celebrum}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}  # Default for dev only - set DB_PASSWORD env var in production

echo "🔍 Checking Celebrum Database Status..."
echo "=====================================\n"

# Function to execute SQL and format output
execute_sql() {
    local query="$1"
    local description="$2"
    
    echo "📊 $description"
    echo "-----------------------------------"
    
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$query" 2>/dev/null | sed 's/^[ \