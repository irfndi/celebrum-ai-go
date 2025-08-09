-- Add funding rate arbitrage support
-- This migration adds tables and functionality for funding rate arbitrage

BEGIN;

-- Add funding rate data table
CREATE TABLE IF NOT EXISTS funding_rates (
    id BIGSERIAL PRIMARY KEY,
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    funding_rate DECIMAL(10, 8) NOT NULL, -- Funding rate as percentage
    funding_time TIMESTAMP WITH TIME ZONE NOT NULL, -- When funding occurs
    next_funding_time TIMESTAMP WITH TIME ZONE, -- Next funding time
    mark_price DECIMAL(20, 8), -- Mark price used for funding
    index_price DECIMAL(20, 8), -- Index price
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(exchange_id, trading_pair_id, funding_time)
);

-- Add funding rate arbitrage opportunities table
CREATE TABLE IF NOT EXISTS funding_arbitrage_opportunities (
    id BIGSERIAL PRIMARY KEY,
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    long_exchange_id INTEGER NOT NULL REFERENCES exchanges(id), -- Exchange to go long (pay funding)
    short_exchange_id INTEGER NOT NULL REFERENCES exchanges(id), -- Exchange to go short (receive funding)
    long_funding_rate DECIMAL(10, 8) NOT NULL, -- Funding rate on long exchange
    short_funding_rate DECIMAL(10, 8) NOT NULL, -- Funding rate on short exchange
    net_funding_rate DECIMAL(10, 8) NOT NULL, -- Net funding rate (short - long)
    estimated_profit_8h DECIMAL(20, 8) NOT NULL, -- Estimated profit per 8 hours
    estimated_profit_daily DECIMAL(20, 8) NOT NULL, -- Estimated daily profit
    estimated_profit_percentage DECIMAL(8, 4) NOT NULL, -- Estimated profit percentage
    long_mark_price DECIMAL(20, 8), -- Mark price on long exchange
    short_mark_price DECIMAL(20, 8), -- Mark price on short exchange
    price_difference DECIMAL(20, 8), -- Price difference between exchanges
    price_difference_percentage DECIMAL(8, 4), -- Price difference percentage
    risk_score DECIMAL(4, 2) DEFAULT 1.0, -- Risk score (1.0 = low risk, 5.0 = high risk)
    is_active BOOLEAN DEFAULT true,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add Bybit exchange
INSERT INTO exchanges (name, display_name, ccxt_id, countries, rate_limit, has_cors, has_spot, has_futures, has_margin, status, website_url) VALUES
('bybit', 'Bybit', 'bybit', ARRAY['VG'], 600, true, true, true, true, 'active', 'https://www.bybit.com')
ON CONFLICT (ccxt_id) DO UPDATE SET
    has_futures = EXCLUDED.has_futures,
    has_margin = EXCLUDED.has_margin,
    status = EXCLUDED.status;

-- Add more trading pairs for funding rate arbitrage (futures pairs)
INSERT INTO trading_pairs (symbol, base_currency, quote_currency, is_futures) VALUES
('BTC/USDT:USDT', 'BTC', 'USDT', true),
('ETH/USDT:USDT', 'ETH', 'USDT', true),
('BNB/USDT:USDT', 'BNB', 'USDT', true),
('ADA/USDT:USDT', 'ADA', 'USDT', true),
('SOL/USDT:USDT', 'SOL', 'USDT', true),
('DOT/USDT:USDT', 'DOT', 'USDT', true),
('MATIC/USDT:USDT', 'MATIC', 'USDT', true),
('AVAX/USDT:USDT', 'AVAX', 'USDT', true),
('LINK/USDT:USDT', 'LINK', 'USDT', true),
('UNI/USDT:USDT', 'UNI', 'USDT', true),
-- Add more popular futures pairs
('XRP/USDT:USDT', 'XRP', 'USDT', true),
('DOGE/USDT:USDT', 'DOGE', 'USDT', true),
('LTC/USDT:USDT', 'LTC', 'USDT', true),
('BCH/USDT:USDT', 'BCH', 'USDT', true),
('ETC/USDT:USDT', 'ETC', 'USDT', true),
('FIL/USDT:USDT', 'FIL', 'USDT', true),
('ATOM/USDT:USDT', 'ATOM', 'USDT', true),
('NEAR/USDT:USDT', 'NEAR', 'USDT', true),
('ALGO/USDT:USDT', 'ALGO', 'USDT', true),
('VET/USDT:USDT', 'VET', 'USDT', true)
ON CONFLICT (symbol, is_futures) DO NOTHING;

-- Create indexes for funding rate tables
CREATE INDEX IF NOT EXISTS idx_funding_rates_exchange_pair_time ON funding_rates(exchange_id, trading_pair_id, funding_time DESC);
CREATE INDEX IF NOT EXISTS idx_funding_rates_timestamp ON funding_rates(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_funding_rates_next_funding ON funding_rates(next_funding_time) WHERE next_funding_time IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_active ON funding_arbitrage_opportunities(is_active, detected_at DESC) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_profit ON funding_arbitrage_opportunities(estimated_profit_percentage DESC);
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_risk ON funding_arbitrage_opportunities(risk_score ASC);
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_expires ON funding_arbitrage_opportunities(expires_at) WHERE expires_at IS NOT NULL;

-- Create view for active funding arbitrage opportunities
CREATE OR REPLACE VIEW active_funding_arbitrage_opportunities AS
SELECT 
    fao.*,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    le.name as long_exchange_name,
    se.name as short_exchange_name
FROM funding_arbitrage_opportunities fao
JOIN trading_pairs tp ON fao.trading_pair_id = tp.id
JOIN exchanges le ON fao.long_exchange_id = le.id
JOIN exchanges se ON fao.short_exchange_id = se.id
WHERE fao.is_active = true
  AND (fao.expires_at IS NULL OR fao.expires_at > CURRENT_TIMESTAMP)
ORDER BY fao.estimated_profit_percentage DESC;

-- Create view for latest funding rates
CREATE OR REPLACE VIEW latest_funding_rates AS
SELECT DISTINCT ON (fr.exchange_id, fr.trading_pair_id)
    fr.*,
    e.name as exchange_name,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency
FROM funding_rates fr
JOIN exchanges e ON fr.exchange_id = e.id
JOIN trading_pairs tp ON fr.trading_pair_id = tp.id
WHERE tp.is_futures = true
ORDER BY fr.exchange_id, fr.trading_pair_id, fr.timestamp DESC;

-- Add system configuration for funding rate arbitrage
INSERT INTO system_config (config_key, config_value, description) VALUES
('funding_rate_min_profit', '0.01', 'Minimum funding rate profit percentage for arbitrage (0.01 = 1% daily)'),
('funding_rate_max_risk', '3.0', 'Maximum risk score for funding rate arbitrage (1.0-5.0)'),
('funding_rate_collection_enabled', 'true', 'Enable funding rate data collection'),
('funding_rate_arbitrage_enabled', 'true', 'Enable funding rate arbitrage detection')
ON CONFLICT (config_key) DO NOTHING;

COMMIT;