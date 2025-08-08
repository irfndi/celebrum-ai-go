# Celebrum AI - Crypto Arbitrage & Technical Analysis Platform

## 1. Product Overview

Celebrum AI is a comprehensive crypto trading platform that identifies arbitrage opportunities across futures markets and provides advanced technical analysis tools. The platform will evolve to include AI-powered analytics and trading signals, helping traders maximize profits through automated opportunity detection and analysis.

- **Target Market**: Professional crypto traders, institutional investors, and trading firms seeking automated arbitrage and technical analysis solutions.
- **Market Value**: Addresses the $2.3 trillion crypto market with focus on futures arbitrage opportunities and technical analysis automation.

## 2. Core Features

### 2.1 User Roles

| Role | Registration Method | Core Permissions |
|------|---------------------|------------------|
| Free User | Telegram ID + username (instant) | View basic market data, limited arbitrage alerts via Telegram |
| Premium User | Subscription upgrade via Telegram | Full arbitrage alerts, advanced technical analysis, priority notifications |
| Enterprise User | Custom onboarding | API access, custom alerts, priority support, website dashboard access |

**Registration Flow:**
- **Primary**: Users start with `/start` command in Telegram bot using only Telegram ID and username
- **Optional**: Users can later connect email/password for website access
- **Seamless**: No initial barriers - immediate access to core features through Telegram

### 2.2 Feature Module

Our crypto arbitrage platform consists of the following main components (Telegram-first approach):

1. **Telegram Bot Interface**: Primary user interface, instant arbitrage alerts, command-based navigation, user onboarding via `/start`.
2. **Arbitrage Opportunities**: Real-time opportunity detection, profit calculations, Telegram notifications, risk assessment.
3. **Market Data Collection**: Background data feeds from multiple exchanges, futures contract monitoring, price aggregation.
4. **User Management**: Telegram-based authentication, optional website linking, subscription management.
5. **Web Dashboard** (Optional): Advanced charts, detailed analysis, portfolio tracking for users who connect via website.

### 2.3 Interface Details

| Interface Type | Module Name | Feature description |
|----------------|-------------|---------------------|
| Telegram Bot | User Onboarding | `/start` command registration, welcome message, feature introduction, subscription options |
| Telegram Bot | Arbitrage Alerts | Real-time opportunity notifications, profit calculations, exchange pair details, quick action buttons |
| Telegram Bot | Command Interface | `/opportunities` - view current arbitrage, `/settings` - configure alerts, `/help` - command list |
| Telegram Bot | Subscription Management | `/upgrade` for premium features, payment integration, feature unlocking |
| Web Dashboard (Optional) | Market Data View | Advanced charts, multiple timeframes, technical indicators for connected users |
| Web Dashboard (Optional) | Portfolio Tracking | Detailed P&L analysis, position monitoring, performance metrics |
| Web Dashboard (Optional) | Account Linking | Connect Telegram account, email/password setup, sync preferences |
| Background Services | Data Collection | Continuous market data fetching, opportunity detection, alert generation |

## 3. Core Process

**Telegram Bot User Journey:**
User discovers bot → `/start` command → Instant registration with Telegram ID → Welcome message with feature overview → User receives arbitrage alerts → Optional upgrade to premium → Optional website connection for advanced features.

**Arbitrage Detection Flow:**
System continuously monitors exchanges → Price differences detected → Profit calculations performed → Alerts sent to Telegram users → Users receive formatted notifications with actionable data → Optional detailed analysis via website.

**Account Linking Flow (Optional):**
Telegram user → Website visit → "Connect Telegram" option → Authentication via Telegram → Account linking → Access to advanced web features while maintaining Telegram as primary interface.

```mermaid
graph TD
    A[User Discovers Bot] --> B[/start Command]
    B --> C[Instant Registration via Telegram ID]
    C --> D[Welcome & Feature Overview]
    D --> E[Background: Market Data Collection]
    E --> F[Opportunity Detection]
    F --> G[Telegram Alert Sent]
    G --> H[User Receives Notification]
    H --> I{User Action}
    I --> J[View More Details]
    I --> K[Upgrade to Premium]
    I --> L[Connect to Website]
    L --> M[Advanced Web Features]
    J --> N[Trading Decision]
    K --> N
    M --> N
```

## 4. User Interface Design

### 4.1 Design Style

- **Primary Colors**: Dark theme with #1a1a1a background, #00ff88 for profits, #ff4444 for losses
- **Secondary Colors**: #333333 for cards, #666666 for borders, #ffffff for text
- **Button Style**: Rounded corners (8px), gradient backgrounds, hover animations
- **Font**: Inter font family, 14px base size, 16px for headers, 12px for small text
- **Layout Style**: Card-based design, sidebar navigation, responsive grid system
- **Icons**: Feather icons for consistency, crypto-specific icons for currencies

### 4.2 Page Design Overview

| Page Name | Module Name | UI Elements |
|-----------|-------------|-------------|
| Market Data Dashboard | Real-time Feed | Dark cards with green/red price indicators, real-time updating numbers, exchange logos |
| Market Data Dashboard | Chart Display | TradingView-style charts, dark theme, multiple timeframe tabs |
| Arbitrage Opportunities | Opportunity List | Table with sortable columns, profit percentage highlights, exchange pair indicators |
| Technical Analysis | Chart Interface | Full-screen chart view, indicator sidebar, drawing tools toolbar |
| Alert Management | Settings Panel | Toggle switches, input fields, telegram connection status |

### 4.3 Responsiveness

Desktop-first design with mobile-adaptive layouts. Touch-optimized interactions for mobile users, collapsible sidebar navigation, and responsive chart scaling.
