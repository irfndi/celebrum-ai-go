#!/bin/bash

# Validation script for required environment variables
# This script checks if all critical environment variables are set

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Validating environment variables..."

# Critical variables that must be set
CRITICAL_VARS=(
  "POSTGRES_USER"
  "POSTGRES_PASSWORD"
  "POSTGRES_DB"
  "REDIS_PASSWORD"
  "JWT_SECRET"
)

# Optional variables with warnings
OPTIONAL_VARS=(
  "TELEGRAM_BOT_TOKEN"
  "TELEGRAM_WEBHOOK_SECRET"
)

MISSING_CRITICAL=0
MISSING_OPTIONAL=0

# Check critical variables
for var in "${CRITICAL_VARS[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo -e "${RED}❌ Missing critical variable: $var${NC}"
    MISSING_CRITICAL=1
  else
    echo -e "${GREEN}✅ $var is set${NC}"
  fi
done

# Check optional variables
for var in "${OPTIONAL_VARS[@]}"; do
  if [[ -z "${!var}" ]]; then
    echo -e "${YELLOW}⚠️  Missing optional variable: $var${NC}"
    MISSING_OPTIONAL=1
  else
    echo -e "${GREEN}✅ $var is set${NC}"
  fi
done

# Summary
if [[ $MISSING_CRITICAL -eq 1 ]]; then
  echo -e "${RED}❌ Critical environment variables are missing!${NC}"
  echo "Please set all required variables in your .env file."
  echo "Copy .env.template to .env and fill in your values."
  exit 1
else
  echo -e "${GREEN}✅ All critical environment variables are set!${NC}"
  if [[ $MISSING_OPTIONAL -eq 1 ]]; then
    echo -e "${YELLOW}⚠️  Some optional variables are missing, but this won't prevent startup.${NC}"
  fi
  exit 0
fi
