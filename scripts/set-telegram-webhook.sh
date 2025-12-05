#!/bin/bash
# Set Telegram Bot Webhook
# This script configures the webhook URL for the Telegram bot

set -e

echo "🔧 Telegram Webhook Setup"
echo "========================="
echo ""

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if .env file exists
if [ ! -f .env ]; then
    echo -e "${RED}❌ .env file not found${NC}"
    echo "Please copy .env.example to .env and configure your settings"
    exit 1
fi

# Load environment variables
set -a
source .env
set +a

# Check if bot token is set
if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo -e "${RED}❌ TELEGRAM_BOT_TOKEN is not set in .env${NC}"
    exit 1
fi

echo "Bot Token: ${TELEGRAM_BOT_TOKEN:0:10}... (length: ${#TELEGRAM_BOT_TOKEN})"
echo ""

# Check command line argument for webhook URL
if [[ -n "$1" ]]; then
    WEBHOOK_URL="$1"
elif [[ -n "$TELEGRAM_WEBHOOK_URL" ]]; then
    WEBHOOK_URL="$TELEGRAM_WEBHOOK_URL"
else
    echo "Usage: $0 <webhook_url>"
    echo ""
    echo "Example:"
    echo "  $0 https://your-domain.com/api/v1/telegram/webhook"
    echo ""
    echo "Or set TELEGRAM_WEBHOOK_URL in .env file"
    exit 1
fi

echo "Webhook URL: $WEBHOOK_URL"
echo ""

# Validate webhook URL format
if [[ ! "$WEBHOOK_URL" =~ ^https:// ]]; then
    echo -e "${RED}❌ Webhook URL must use HTTPS${NC}"
    echo "Telegram requires HTTPS for webhooks"
    exit 1
fi

# Ask for confirmation
read -p "Set this webhook URL? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Cancelled"
    exit 0
fi

echo ""
echo "📡 Setting webhook..."

# Set webhook
RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook" \
    -d "url=${WEBHOOK_URL}" \
    -d "drop_pending_updates=true")

# Check response
if echo "$RESPONSE" | grep -q '"ok":true'; then
    echo -e "${GREEN}✅ Webhook set successfully${NC}"
    DESCRIPTION=$(echo "$RESPONSE" | grep -o '"description":"[^"]*"' | cut -d'"' -f4)
    echo "   $DESCRIPTION"
else
    echo -e "${RED}❌ Failed to set webhook${NC}"
    ERROR_DESC=$(echo "$RESPONSE" | grep -o '"description":"[^"]*"' | cut -d'"' -f4)
    echo "   Error: $ERROR_DESC"
    exit 1
fi

echo ""
echo "🔍 Verifying webhook..."

# Get webhook info to verify
WEBHOOK_INFO=$(curl -s "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/getWebhookInfo")

CURRENT_URL=$(echo "$WEBHOOK_INFO" | grep -o '"url":"[^"]*"' | cut -d'"' -f4)
PENDING_COUNT=$(echo "$WEBHOOK_INFO" | grep -o '"pending_update_count":[0-9]*' | cut -d':' -f2)

echo "Current webhook URL: $CURRENT_URL"
echo "Pending updates: ${PENDING_COUNT:-0}"

echo ""
echo -e "${GREEN}✅ Webhook setup complete!${NC}"
echo ""
echo "📝 Next steps:"
echo "1. Ensure your application is running and accessible at $WEBHOOK_URL"
echo "2. Test the bot by sending /start in Telegram"
echo "3. Check application logs for incoming webhook requests"
echo ""
