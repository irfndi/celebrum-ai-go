#!/bin/bash
# Debug script for Database Connectivity
# Run this on the server or inside the container

echo "=== Database Connection Debugger ==="
echo "Time: $(date)"
echo "User: $(whoami)"
echo "Hostname: $(hostname)"
echo "===================================="

# 1. Check Environment Variables
echo "Checking Environment Variables..."
echo "DATABASE_HOST=${DATABASE_HOST}"
echo "DATABASE_PORT=${DATABASE_PORT}"
echo "DATABASE_USER=${DATABASE_USER}"
echo "DATABASE_DBNAME=${DATABASE_DBNAME}"
if [ -n "$DATABASE_PASSWORD" ]; then
  echo "DATABASE_PASSWORD is SET (Length: ${#DATABASE_PASSWORD})"
else
  echo "DATABASE_PASSWORD is NOT SET"
fi
echo "DATABASE_URL=${DATABASE_URL}"
echo ""
# 2. DNS Resolution Check
echo "Checking DNS Resolution..."
if [ -n "$DATABASE_HOST" ]; then
  echo "Resolving $DATABASE_HOST..."
  getent hosts "$DATABASE_HOST" || echo "getent failed"
  echo "Detailed DNS Info:"
  cat /etc/resolv.conf
  echo "Hosts File:"
  cat /etc/hosts
  ping -c 1 "$DATABASE_HOST" || echo "FAILED to ping $DATABASE_HOST"
fi
echo ""

# 3. Connectivity Check
echo "Checking TCP Connectivity..."
if [ -n "$DATABASE_HOST" ] && [ -n "$DATABASE_PORT" ]; then
  nc -zv "$DATABASE_HOST" "$DATABASE_PORT" || echo "FAILED to connect to $DATABASE_HOST:$DATABASE_PORT via TCP"
fi
echo ""

# 4. PostgreSQL Connection Check
echo "Checking PostgreSQL Connection (pg_isready)..."
if [ -n "$DATABASE_HOST" ]; then
  PGPASSWORD="${DATABASE_PASSWORD}" pg_isready -h "${DATABASE_HOST}" -p "${DATABASE_PORT:-5432}" -U "${DATABASE_USER:-postgres}" -d "${DATABASE_DBNAME:-postgres}"
  EXIT_CODE=$?
  echo "pg_isready exit code: $EXIT_CODE"
fi
echo ""

# 5. PSQL Login Check
echo "Checking PSQL Login..."
if [ -n "$DATABASE_HOST" ]; then
  PGPASSWORD="${DATABASE_PASSWORD}" psql -h "${DATABASE_HOST}" -p "${DATABASE_PORT:-5432}" -U "${DATABASE_USER:-postgres}" -d "${DATABASE_DBNAME:-postgres}" -c "SELECT 1"
  PSQL_EXIT_CODE=$?
  echo "psql exit code: $PSQL_EXIT_CODE"

  if [ $PSQL_EXIT_CODE -ne 0 ]; then
    echo "!!! PSQL LOGIN FAILED !!!"
    echo "If error is 'password authentication failed', your DATABASE_PASSWORD does not match the database."
    echo "Try resetting the password in the database container or updating the environment variable."
  fi
fi

echo "===================================="
