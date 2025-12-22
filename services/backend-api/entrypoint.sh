#!/bin/bash
set -e

# Run migrations
echo "Running database migrations..."
if [ -f "database/migrate.sh" ]; then
  chmod +x database/migrate.sh
  ./database/migrate.sh
else
  echo "Migration script not found at ./database/migrate.sh"
fi

# Start application
echo "Starting application..."
exec ./main
