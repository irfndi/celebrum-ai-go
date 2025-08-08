-- Minimal Celebrum AI Database Initialization Script
-- This script sets up a basic database schema for CI/CD environments

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Exchanges table
CREATE TABLE IF NOT EXISTS exchanges (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(100) NOT NULL,
    ccxt_id VARCHAR(50) NOT NULL UNIQUE,
    status exchange_status DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Trading pairs table
CREATE TABLE IF NOT EXISTS trading_pairs (
    id SERIAL PRIMARY KEY,
    symbol VARCHAR(20) NOT NULL,
    base_currency VARCHAR(10) NOT NULL,
    quote_currency VARCHAR(10) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol)
);

-- Market data table
CREATE TABLE IF NOT EXISTS market_data (
    id BIGSERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    last_price DECIMAL(20, 8),
    volume_24h DECIMAL(20, 8),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert basic test data
INSERT INTO exchanges (name, display_name, ccxt_id) VALUES 
('binance', 'Binance', 'binance'),
('coinbase', 'Coinbase Pro', 'coinbasepro')
ON CONFLICT (name) DO NOTHING;

INSERT INTO trading_pairs (symbol, base_currency, quote_currency) VALUES 
('BTC/USDT', 'BTC', 'USDT'),
('ETH/USDT', 'ETH', 'USDT')
ON CONFLICT (symbol) DO NOTHING;