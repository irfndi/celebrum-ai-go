-- Enhanced Initial Schema for Celebrum AI Platform
-- Migration 004: Enhanced database structure with comprehensive features
-- Created: 2025-01-17
-- Based on: scripts/init.sql (enhanced version)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create schemas for better organization
CREATE SCHEMA IF NOT EXISTS public;
CREATE SCHEMA IF NOT EXISTS analytics;
CREATE SCHEMA IF NOT EXISTS audit;

-- Create custom types for better data integrity
CREATE TYPE exchange_status AS ENUM ('active', 'inactive', 'maintenance');
CREATE TYPE order_side AS ENUM ('buy', 'sell');
CREATE TYPE alert_type AS ENUM ('price', 'arbitrage', 'technical');
CREATE TYPE alert_status AS ENUM ('active', 'triggered', 'expired', 'disabled');

-- Users table (enhanced version)
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

-- Exchanges table (comprehensive)
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
    exchange_symbol VARCHAR(50) NOT NULL,
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
    timeframe VARCHAR(10) NOT NULL,
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
    indicator_name VARCHAR(50) NOT NULL,
    indicator_value DECIMAL(20, 8),
    indicator_signal VARCHAR(20),
    metadata JSONB,
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
    conditions JSONB NOT NULL,
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
    notification_type VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    sent_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    delivery_status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT
);

-- API keys for user integrations
CREATE TABLE IF NOT EXISTS user_api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_name VARCHAR(100) NOT NULL,
    api_key_hash VARCHAR(255) NOT NULL,
    permissions TEXT[] DEFAULT ARRAY['read'],
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

-- Migration completed
INSERT INTO system_config (config_key, config_value, description) VALUES
('migration_004_completed', 'true', 'Enhanced initial schema migration completed'),
('schema_version', '004', 'Current database schema version')
ON CONFLICT (config_key) DO UPDATE SET
    config_value = EXCLUDED.config_value,
    updated_at = CURRENT_TIMESTAMP;