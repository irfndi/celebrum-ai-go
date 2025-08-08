-- Celebrum AI Database Initialization Script
-- This script sets up the initial database schema for the Celebrum AI platform

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create database if it doesn't exist (this would typically be run by a superuser)
-- CREATE DATABASE celebrum_ai;

-- Connect to the celebrum_ai database
-- \c celebrum_ai;

-- Create schemas
CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS analytics;
CREATE SCHEMA IF NOT EXISTS audit;

-- Create custom types
CREATE TYPE exchange_status AS ENUM ('active', 'inactive', 'maintenance');
CREATE TYPE order_side AS ENUM ('buy', 'sell');
CREATE TYPE alert_type AS ENUM ('price', 'arbitrage', 'technical');
CREATE TYPE alert_status AS ENUM ('active', 'triggered', 'expired', 'disabled');

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    telegram_user_id BIGINT UNIQUE,
    telegram_username VARCHAR(50),
    is_active BOOLEAN DEFAULT true,
    is_premium BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE
);

-- Exchanges table
CREATE TABLE IF NOT EXISTS exchanges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    ccxt_id VARCHAR(50) NOT NULL UNIQUE,
    countries TEXT[],
    rate_limit INTEGER DEFAULT 1000,
    has_cors BOOLEAN DEFAULT false,
    has_spot BOOLEAN DEFAULT true,
    has_futures BOOLEAN DEFAULT false,
    has_margin BOOLEAN DEFAULT false,
    has_option BOOLEAN DEFAULT false,
    sandbox BOOLEAN DEFAULT false,
    status exchange_status DEFAULT 'active',
    api_url VARCHAR(255),
    website_url VARCHAR(255),
    fees_url VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trading pairs table
CREATE TABLE IF NOT EXISTS trading_pairs (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    base_currency VARCHAR(10) NOT NULL,
    quote_currency VARCHAR(10) NOT NULL,
    is_futures BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol, is_futures)
);

-- Exchange trading pairs (many-to-many relationship)
CREATE TABLE IF NOT EXISTS exchange_trading_pairs (
    id SERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id) ON DELETE CASCADE,
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id) ON DELETE CASCADE,
    exchange_symbol VARCHAR(50) NOT NULL, -- Symbol as used by the exchange
    is_active BOOLEAN DEFAULT true,
    min_amount DECIMAL(20, 8),
    max_amount DECIMAL(20, 8),
    min_price DECIMAL(20, 8),
    max_price DECIMAL(20, 8),
    price_precision INTEGER DEFAULT 8,
    amount_precision INTEGER DEFAULT 8,
    taker_fee DECIMAL(6, 4) DEFAULT 0.001,
    maker_fee DECIMAL(6, 4) DEFAULT 0.001,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange_id, trading_pair_id)
);

-- Market data table for storing ticker information
CREATE TABLE IF NOT EXISTS market_data (
    id BIGSERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    bid DECIMAL(20, 8),
    bid_volume DECIMAL(20, 8),
    ask DECIMAL(20, 8),
    ask_volume DECIMAL(20, 8),
    last_price DECIMAL(20, 8),
    high_24h DECIMAL(20, 8),
    low_24h DECIMAL(20, 8),
    open_24h DECIMAL(20, 8),
    close_24h DECIMAL(20, 8),
    volume_24h DECIMAL(20, 8),
    quote_volume_24h DECIMAL(20, 8),
    vwap DECIMAL(20, 8),
    change_24h DECIMAL(20, 8),
    change_percentage_24h DECIMAL(8, 4),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Order book data
CREATE TABLE IF NOT EXISTS order_book_data (
    id BIGSERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    side order_side NOT NULL,
    price DECIMAL(20, 8) NOT NULL,
    amount DECIMAL(20, 8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- OHLCV data for technical analysis
CREATE TABLE IF NOT EXISTS ohlcv_data (
    id BIGSERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    timeframe VARCHAR(10) NOT NULL, -- 1m, 5m, 15m, 1h, 4h, 1d, etc.
    open_price DECIMAL(20, 8) NOT NULL,
    high_price DECIMAL(20, 8) NOT NULL,
    low_price DECIMAL(20, 8) NOT NULL,
    close_price DECIMAL(20, 8) NOT NULL,
    volume DECIMAL(20, 8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange_id, trading_pair_id, timeframe, timestamp)
);

-- Arbitrage opportunities
CREATE TABLE IF NOT EXISTS arbitrage_opportunities (
    id BIGSERIAL PRIMARY KEY,
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    buy_exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    sell_exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    buy_price DECIMAL(20, 8) NOT NULL,
    sell_price DECIMAL(20, 8) NOT NULL,
    profit_amount DECIMAL(20, 8) NOT NULL,
    profit_percentage DECIMAL(8, 4) NOT NULL,
    volume DECIMAL(20, 8) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Technical indicators
CREATE TABLE IF NOT EXISTS technical_indicators (
    id BIGSERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    timeframe VARCHAR(10) NOT NULL,
    indicator_name VARCHAR(50) NOT NULL, -- RSI, MACD, SMA, EMA, etc.
    indicator_value DECIMAL(20, 8),
    indicator_signal VARCHAR(20), -- buy, sell, neutral
    metadata JSONB, -- Additional indicator-specific data
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User alerts
CREATE TABLE IF NOT EXISTS user_alerts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trading_pair_id INTEGER REFERENCES trading_pairs(id),
    exchange_id INTEGER REFERENCES exchanges(id),
    alert_type alert_type NOT NULL,
    alert_name VARCHAR(100) NOT NULL,
    conditions JSONB NOT NULL, -- Flexible conditions storage
    target_price DECIMAL(20, 8),
    is_active BOOLEAN DEFAULT true,
    status alert_status DEFAULT 'active',
    triggered_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Alert notifications log
CREATE TABLE IF NOT EXISTS alert_notifications (
    id BIGSERIAL PRIMARY KEY,
    alert_id UUID NOT NULL REFERENCES user_alerts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    notification_type VARCHAR(20) NOT NULL, -- telegram, email, webhook
    message TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    delivery_status VARCHAR(20) DEFAULT 'pending', -- pending, sent, failed
    error_message TEXT
);

-- API keys for user integrations
CREATE TABLE IF NOT EXISTS user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL,
    permissions TEXT[] DEFAULT ARRAY['read'], -- read, write, admin
    is_active BOOLEAN DEFAULT true,
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- System configuration
CREATE TABLE IF NOT EXISTS system_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(100) UNIQUE NOT NULL,
    config_value TEXT NOT NULL,
    description TEXT,
    is_encrypted BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_market_data_exchange_pair_timestamp ON market_data(exchange_id, trading_pair_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_market_data_timestamp ON market_data(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_order_book_exchange_pair_timestamp ON order_book_data(exchange_id, trading_pair_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ohlcv_exchange_pair_timeframe_timestamp ON ohlcv_data(exchange_id, trading_pair_id, timeframe, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_arbitrage_opportunities_active ON arbitrage_opportunities(is_active, detected_at DESC) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_technical_indicators_exchange_pair_timeframe ON technical_indicators(exchange_id, trading_pair_id, timeframe, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_user_alerts_user_active ON user_alerts(user_id, is_active) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_alert_notifications_user_sent ON alert_notifications(user_id, sent_at DESC);
CREATE INDEX IF NOT EXISTS idx_users_telegram_user_id ON users(telegram_user_id) WHERE telegram_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_exchanges_status ON exchanges(status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_trading_pairs_active ON trading_pairs(is_active) WHERE is_active = true;

-- Create triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_exchanges_updated_at BEFORE UPDATE ON exchanges FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_trading_pairs_updated_at BEFORE UPDATE ON trading_pairs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_exchange_trading_pairs_updated_at BEFORE UPDATE ON exchange_trading_pairs FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_user_alerts_updated_at BEFORE UPDATE ON user_alerts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_system_config_updated_at BEFORE UPDATE ON system_config FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert initial system configuration
INSERT INTO system_config (config_key, config_value, description) VALUES
('app_version', '1.0.0', 'Application version'),
('maintenance_mode', 'false', 'Enable/disable maintenance mode'),
('max_arbitrage_opportunities', '100', 'Maximum number of arbitrage opportunities to store'),
('data_retention_days', '30', 'Number of days to retain market data'),
('rate_limit_per_minute', '60', 'API rate limit per minute per user'),
('telegram_bot_enabled', 'true', 'Enable/disable Telegram bot functionality')
ON CONFLICT (config_key) DO NOTHING;

-- Insert some initial exchanges
INSERT INTO exchanges (name, display_name, ccxt_id, countries, rate_limit, has_cors, has_spot, has_futures, status, website_url) VALUES
('binance', 'Binance', 'binance', ARRAY['CN', 'MT'], 1200, true, true, true, 'active', 'https://www.binance.com'),
('coinbase', 'Coinbase Pro', 'coinbasepro', ARRAY['US'], 10000, true, true, false, 'active', 'https://pro.coinbase.com'),
('kraken', 'Kraken', 'kraken', ARRAY['US'], 3000, true, true, true, 'active', 'https://www.kraken.com'),
('bitfinex', 'Bitfinex', 'bitfinex', ARRAY['VG'], 1500, true, true, false, 'active', 'https://www.bitfinex.com'),
('huobi', 'Huobi Global', 'huobi', ARRAY['SC'], 2000, true, true, true, 'active', 'https://www.huobi.com')
ON CONFLICT (ccxt_id) DO NOTHING;

-- Insert some common trading pairs
INSERT INTO trading_pairs (symbol, base_currency, quote_currency, is_futures) VALUES
('BTC/USDT', 'BTC', 'USDT', false),
('ETH/USDT', 'ETH', 'USDT', false),
('BNB/USDT', 'BNB', 'USDT', false),
('ADA/USDT', 'ADA', 'USDT', false),
('SOL/USDT', 'SOL', 'USDT', false),
('DOT/USDT', 'DOT', 'USDT', false),
('MATIC/USDT', 'MATIC', 'USDT', false),
('AVAX/USDT', 'AVAX', 'USDT', false),
('LINK/USDT', 'LINK', 'USDT', false),
('UNI/USDT', 'UNI', 'USDT', false)
ON CONFLICT (symbol, is_futures) DO NOTHING;

-- Create a view for active arbitrage opportunities
CREATE OR REPLACE VIEW active_arbitrage_opportunities AS
SELECT 
    ao.*,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    be.name as buy_exchange_name,
    se.name as sell_exchange_name
FROM arbitrage_opportunities ao
JOIN trading_pairs tp ON ao.trading_pair_id = tp.id
JOIN exchanges be ON ao.buy_exchange_id = be.id
JOIN exchanges se ON ao.sell_exchange_id = se.id
WHERE ao.is_active = true
  AND (ao.expires_at IS NULL OR ao.expires_at > CURRENT_TIMESTAMP)
ORDER BY ao.profit_percentage DESC;

-- Create a view for latest market data
CREATE OR REPLACE VIEW latest_market_data AS
SELECT DISTINCT ON (md.exchange_id, md.trading_pair_id)
    md.*,
    e.name as exchange_name,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency
FROM market_data md
JOIN exchanges e ON md.exchange_id = e.id
JOIN trading_pairs tp ON md.trading_pair_id = tp.id
ORDER BY md.exchange_id, md.trading_pair_id, md.timestamp DESC;

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO celebrum_ai_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO celebrum_ai_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO celebrum_ai_user;

COMMIT;