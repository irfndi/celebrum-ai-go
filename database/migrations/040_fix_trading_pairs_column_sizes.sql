-- Migration 040: Fix trading_pairs table column sizes
-- Description: Increase column sizes to accommodate longer symbol names, especially for futures contracts
-- This fixes the "value too long for type character varying(20)" error

BEGIN;

-- Drop dependent views first
DROP VIEW IF EXISTS active_funding_arbitrage_opportunities CASCADE;
DROP VIEW IF EXISTS active_futures_arbitrage_opportunities CASCADE;
DROP VIEW IF EXISTS futures_arbitrage_market_summary CASCADE;
DROP VIEW IF EXISTS active_exchange_trading_pairs CASCADE;
DROP VIEW IF EXISTS blacklisted_exchange_trading_pairs CASCADE;
DROP VIEW IF EXISTS v_trading_pairs_debug CASCADE;
DROP VIEW IF EXISTS latest_funding_rates CASCADE;

-- Increase symbol column size from VARCHAR(20) to VARCHAR(50)
ALTER TABLE trading_pairs ALTER COLUMN symbol TYPE VARCHAR(50);

-- Increase base_currency column size from VARCHAR(10) to VARCHAR(20)
ALTER TABLE trading_pairs ALTER COLUMN base_currency TYPE VARCHAR(20);

-- Increase quote_currency column size from VARCHAR(10) to VARCHAR(20)
ALTER TABLE trading_pairs ALTER COLUMN quote_currency TYPE VARCHAR(20);

-- Update any related tables that might have similar constraints
-- Check if exchange_trading_pairs table exists and update it too
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'exchange_trading_pairs') THEN
        -- Update exchange_trading_pairs table columns if they exist
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'exchange_trading_pairs' AND column_name = 'symbol') THEN
            ALTER TABLE exchange_trading_pairs ALTER COLUMN symbol TYPE VARCHAR(50);
        END IF;
        
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'exchange_trading_pairs' AND column_name = 'base_currency') THEN
            ALTER TABLE exchange_trading_pairs ALTER COLUMN base_currency TYPE VARCHAR(20);
        END IF;
        
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'exchange_trading_pairs' AND column_name = 'quote_currency') THEN
            ALTER TABLE exchange_trading_pairs ALTER COLUMN quote_currency TYPE VARCHAR(20);
        END IF;
    END IF;
END $$;

-- Recreate the active_funding_arbitrage_opportunities view
CREATE OR REPLACE VIEW active_funding_arbitrage_opportunities AS
SELECT 
    *,
    CASE 
        WHEN estimated_profit_percentage >= 20 AND risk_score <= 30 THEN 'excellent'
        WHEN estimated_profit_percentage >= 15 AND risk_score <= 50 THEN 'good'
        WHEN estimated_profit_percentage >= 10 AND risk_score <= 70 THEN 'moderate'
        ELSE 'poor'
    END as opportunity_quality
FROM funding_arbitrage_opportunities 
WHERE is_active = true 
  AND (expires_at IS NULL OR expires_at > NOW())
  AND estimated_profit_percentage > 0
ORDER BY estimated_profit_percentage DESC, risk_score ASC;

-- Recreate the active_futures_arbitrage_opportunities view
CREATE OR REPLACE VIEW active_futures_arbitrage_opportunities AS
SELECT 
    *,
    CASE 
        WHEN apy >= 20 AND risk_score <= 30 THEN 'excellent'
        WHEN apy >= 15 AND risk_score <= 50 THEN 'good'
        WHEN apy >= 10 AND risk_score <= 70 THEN 'moderate'
        ELSE 'poor'
    END as opportunity_quality,
    
    CASE 
        WHEN time_to_next_funding <= 60 THEN 'urgent'
        WHEN time_to_next_funding <= 180 THEN 'soon'
        ELSE 'later'
    END as funding_urgency
FROM futures_arbitrage_opportunities 
WHERE is_active = true 
  AND expires_at > NOW()
  AND apy > 0
ORDER BY apy DESC, risk_score ASC;

-- Recreate the futures_arbitrage_market_summary view
CREATE OR REPLACE VIEW futures_arbitrage_market_summary AS
SELECT 
    COUNT(*) as total_opportunities,
    AVG(apy) as average_apy,
    MAX(apy) as highest_apy,
    AVG(risk_score) as average_risk_score,
    AVG(volatility_score) as average_volatility,
    COUNT(DISTINCT symbol) as unique_symbols,
    COUNT(DISTINCT long_exchange || '-' || short_exchange) as unique_exchange_pairs,
    
    -- Market condition assessment
    CASE 
        WHEN AVG(apy) >= 15 AND AVG(risk_score) <= 50 THEN 'favorable'
        WHEN AVG(apy) >= 8 AND AVG(risk_score) <= 70 THEN 'neutral'
        ELSE 'unfavorable'
    END as market_condition,
    
    -- Funding rate trend
    CASE 
        WHEN AVG(net_funding_rate) > 0.01 THEN 'high_positive'
        WHEN AVG(net_funding_rate) > 0.005 THEN 'moderate_positive'
        WHEN AVG(net_funding_rate) > -0.005 THEN 'neutral'
        WHEN AVG(net_funding_rate) > -0.01 THEN 'moderate_negative'
        ELSE 'high_negative'
    END as funding_rate_trend
    
FROM active_futures_arbitrage_opportunities;

-- Recreate the active_exchange_trading_pairs view
CREATE OR REPLACE VIEW active_exchange_trading_pairs AS
SELECT 
    etp.*,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    tp.category
FROM exchange_trading_pairs etp
JOIN trading_pairs tp ON etp.trading_pair_id = tp.id
WHERE etp.is_active = true 
  AND etp.is_blacklisted = false;

-- Recreate the blacklisted_exchange_trading_pairs view
CREATE OR REPLACE VIEW blacklisted_exchange_trading_pairs AS
SELECT 
    etp.*,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    tp.category
FROM exchange_trading_pairs etp
JOIN trading_pairs tp ON etp.trading_pair_id = tp.id
WHERE etp.is_blacklisted = true;

-- Migration completion
INSERT INTO schema_migrations (filename, checksum, applied_at) VALUES ('040_fix_trading_pairs_column_sizes.sql', 'checksum_040', NOW())
ON CONFLICT (filename) DO UPDATE SET 
    checksum = EXCLUDED.checksum,
    applied_at = NOW();

COMMIT;