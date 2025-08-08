# Telegram Bot Specification - Celebrum AI

## 1. Bot Overview

The Celebrum AI Telegram Bot serves as the primary user interface for the crypto arbitrage platform, providing instant access to arbitrage opportunities and seamless user onboarding without traditional registration barriers.

**Bot Features:**
- Instant registration using Telegram ID and username
- Real-time arbitrage opportunity notifications
- Command-based interface for easy navigation
- Subscription management and payment integration
- Optional website account linking

## 2. User Journey & Commands

### 2.1 Initial User Experience

**Discovery → Registration → First Alert**
1. User discovers bot via link or search
2. User sends `/start` command
3. Bot automatically registers user with Telegram ID
4. Bot sends welcome message with feature overview
5. User immediately starts receiving arbitrage alerts

### 2.2 Core Bot Commands

| Command | Description | User Access |
|---------|-------------|-------------|
| `/start` | Register new user, show welcome message | All users |
| `/opportunities` | View current arbitrage opportunities | All users |
| `/settings` | Configure alert preferences | All users |
| `/upgrade` | Upgrade to premium subscription | Free users |
| `/help` | Show available commands and features | All users |
| `/link` | Connect to website for advanced features | All users |
| `/status` | Check subscription and account info | All users |
| `/stop` | Pause all notifications | All users |
| `/resume` | Resume notifications | All users |

### 2.3 Command Implementations

**`/start` Command Flow:**
```
User: /start
Bot: 🚀 Welcome to Celebrum AI!

✅ You're now registered and ready to receive arbitrage alerts!

🔍 What you get:
• Real-time arbitrage opportunities
• Profit calculations across exchanges
• Instant notifications when opportunities arise

📊 Current opportunities: 3 active
💰 Best opportunity: 2.4% profit on BTC/USDT

Use /opportunities to see all current opportunities
Use /help to see all available commands

🎯 Want more features? /upgrade for premium access!
```

**`/opportunities` Command Flow:**
```
User: /opportunities
Bot: 📈 Current Arbitrage Opportunities:

🥇 BTC/USDT
💰 Profit: 2.4% ($240 per $10k)
📊 Buy: Binance ($41,250)
📊 Sell: Bybit ($42,240)
⏰ Valid for: 3 minutes

🥈 ETH/USDT
💰 Profit: 1.8% ($180 per $10k)
📊 Buy: OKX ($2,450)
📊 Sell: Coinbase ($2,494)
⏰ Valid for: 5 minutes

🥉 SOL/USDT
💰 Profit: 1.2% ($120 per $10k)
📊 Buy: Kraken ($98.50)
📊 Sell: Binance ($99.68)
⏰ Valid for: 2 minutes

🔄 Auto-refresh: ON
⚙️ Configure alerts: /settings
```

## 3. Alert System

### 3.1 Alert Types

**Free Tier Alerts:**
- Basic arbitrage opportunities (>1.5% profit)
- Maximum 10 alerts per day
- 5-minute delay on notifications

**Premium Tier Alerts:**
- All arbitrage opportunities (>0.5% profit)
- Unlimited alerts
- Instant notifications
- Custom profit thresholds
- Priority exchange pairs

### 3.2 Alert Format

```
🚨 ARBITRAGE ALERT 🚨

💎 BTC/USDT
💰 Profit: 2.1% ($210 per $10k)

📊 BUY: Binance
💵 Price: $41,250

📊 SELL: Bybit  
💵 Price: $42,115

⏰ Window: ~4 minutes
🔥 Confidence: High

📱 View details: /opportunities
⚙️ Settings: /settings
```

## 4. Subscription Management

### 4.1 Subscription Tiers

| Feature | Free | Premium | Enterprise |
|---------|------|---------|------------|
| Daily Alerts | 10 | Unlimited | Unlimited |
| Alert Delay | 5 minutes | Instant | Instant |
| Min Profit | 1.5% | 0.5% | Custom |
| Custom Thresholds | ❌ | ✅ | ✅ |
| Website Access | ❌ | ✅ | ✅ |
| API Access | ❌ | ❌ | ✅ |
| Priority Support | ❌ | ❌ | ✅ |

### 4.2 Payment Integration

**Supported Payment Methods:**
- Telegram Stars (native Telegram payments)
- Stripe integration for credit cards
- Crypto payments (USDT, BTC, ETH)

**Upgrade Flow:**
```
User: /upgrade
Bot: 🎯 Upgrade to Premium

✨ Premium Benefits:
• Unlimited alerts
• Instant notifications  
• Custom profit thresholds
• Website dashboard access
• Priority support

💰 Price: $29/month

💳 Payment Options:
1️⃣ Telegram Stars (⭐ 290)
2️⃣ Credit Card
3️⃣ Crypto (USDT)

Choose your payment method:
```

## 5. Website Linking

### 5.1 Account Linking Flow

```
User: /link
Bot: 🔗 Connect to Website Dashboard

Get access to:
📊 Advanced charts and analysis
📈 Portfolio tracking
⚙️ Detailed settings
📱 Multi-device access

🔐 Setup Process:
1. Visit: https://celebrum.ai/auth/telegram
2. Click "Connect Telegram"
3. Authorize the connection
4. Set email & password for website

✅ Your Telegram alerts will continue working
🎯 Plus you'll get advanced web features!

Ready to connect? Visit the link above!
```

### 5.2 Linked Account Benefits

**Telegram Bot (continues to work):**
- All existing alert functionality
- Command-based interface
- Mobile-first experience

**Website Dashboard (additional features):**
- Interactive charts and technical analysis
- Historical data and backtesting
- Portfolio tracking and P&L analysis
- Advanced alert configuration
- Export and reporting features

## 6. Technical Implementation

### 6.1 Bot Architecture

```
Telegram Bot API → Webhook → Go Backend → Database
                                    ↓
                              Alert Processor
                                    ↓
                              Telegram Notifications
```

### 6.2 Key Components

**Webhook Handler:**
- Process incoming Telegram updates
- Route commands to appropriate handlers
- Handle user registration and authentication

**Command Processors:**
- Individual handlers for each bot command
- Business logic for user interactions
- Integration with arbitrage detection system

**Alert Dispatcher:**
- Background service for sending notifications
- Rate limiting and user preference management
- Message formatting and delivery

### 6.3 Database Integration

**User Registration:**
```sql
INSERT INTO users (
    telegram_id, 
    telegram_username, 
    telegram_chat_id,
    subscription_tier,
    created_at
) VALUES ($1, $2, $3, 'free', NOW())
ON CONFLICT (telegram_id) DO NOTHING;
```

**Alert Preferences:**
```sql
CREATE TABLE user_alert_preferences (
    user_id UUID REFERENCES users(id),
    min_profit_threshold DECIMAL DEFAULT 1.5,
    max_alerts_per_day INTEGER DEFAULT 10,
    preferred_exchanges TEXT[],
    preferred_pairs TEXT[],
    alert_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 7. Error Handling & Edge Cases

### 7.1 Common Scenarios

**Bot Blocked by User:**
- Gracefully handle delivery failures
- Mark user as inactive in database
- Resume when user unblocks bot

**Rate Limiting:**
- Implement exponential backoff
- Queue messages for delayed delivery
- Prioritize high-value alerts

**Invalid Commands:**
```
User: /invalid
Bot: ❓ Unknown command!

Available commands:
• /opportunities - View current arbitrage
• /settings - Configure alerts
• /help - Show all commands

Need help? Use /help for full command list!
```

### 7.2 Monitoring & Analytics

**Key Metrics:**
- Daily active users (DAU)
- Command usage frequency
- Alert delivery success rate
- Conversion rate (free → premium)
- User retention (7-day, 30-day)

**Health Checks:**
- Bot API connectivity
- Webhook response times
- Alert delivery latency
- Database query performance

## 8. Security Considerations

### 8.1 User Authentication

- Telegram ID as primary identifier
- No sensitive data stored in bot messages
- Secure webhook with HTTPS only
- Rate limiting on all endpoints

### 8.2 Data Privacy

- Minimal data collection (only Telegram ID/username)
- No message history storage
- GDPR compliance for EU users
- Clear data retention policies

### 8.3 Bot Token Security

- Environment variable storage
- Webhook URL validation
- Regular token rotation
- Access logging and monitoring
