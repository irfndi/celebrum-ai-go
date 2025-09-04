-- Create funding arbitrage opportunities table
-- Migration 011: Add missing funding_arbitrage_opportunities table
-- Created: 2025-01-17

BEGIN;

-- Create funding arbitrage opportunities table
CREATE TABLE IF NOT EXISTS funding_arbitrage_opportunities (
    id BIGSERIAL PRIMARY KEY,
    trading_pair_id INTEGER NOT NULL REFERENCES trading_pairs(id),
    long_exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    short_exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    long_funding_rate DECIMAL(10, 8) NOT NULL,
    short_funding_rate DECIMAL(10, 8) NOT NULL,
    net_funding_rate DECIMAL(10, 8) NOT NULL,
    estimated_profit_8h DECIMAL(20, 8) NOT NULL,
    estimated_profit_daily DECIMAL(20, 8) NOT NULL,
    estimated_profit_percentage DECIMAL(8, 4) NOT NULL,
    long_mark_price DECIMAL(20, 8),
    short_mark_price DECIMAL(20, 8),
    price_difference DECIMAL(20, 8),
    price_difference_percentage DECIMAL(8, 4),
    risk_score DECIMAL(4, 2) DEFAULT 1.0,
    is_active BOOLEAN DEFAULT true,
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for funding arbitrage opportunities
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_active ON funding_arbitrage_opportunities(is_active, detected_at DESC) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_profit ON funding_arbitrage_opportunities(estimated_profit_percentage DESC);
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_active_filter ON funding_arbitrage_opportunities(is_active) WHERE is_active = true;
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

COMMIT;