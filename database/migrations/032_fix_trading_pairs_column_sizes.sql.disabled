-- Migration 032: Fix trading_pairs table column sizes
-- Description: Increase column sizes to accommodate longer symbol names, especially for futures contracts
-- This fixes the "value too long for type character varying(20)" error

BEGIN;

-- Drop dependent views first
DROP VIEW IF EXISTS active_funding_arbitrage_opportunities CASCADE;
DROP VIEW IF EXISTS active_futures_arbitrage_opportunities CASCADE;
DROP VIEW IF EXISTS futures_arbitrage_market_summary CASCADE;
DROP VIEW IF EXISTS active_exchange_trading_pairs CASCADE;
DROP VIEW IF EXISTS blacklisted_exchange_trading_pairs CASCADE;

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

-- Recreate the views
-- Active funding arbitrage opportunities view
CREATE VIEW active_funding_arbitrage_opportunities AS
SELECT 
    fao.id,
    fao.trading_pair_id,
    fao.long_exchange_id,
    fao.short_exchange_id,
    fao.long_funding_rate,
    fao.short_funding_rate,
    fao.net_funding_rate,
    fao.estimated_profit_8h,
    fao.estimated_profit_daily,
    fao.estimated_profit_percentage,
    fao.long_mark_price,
    fao.short_mark_price,
    fao.price_difference,
    fao.price_difference_percentage,
    fao.risk_score,
    fao.is_active,
    fao.detected_at,
    fao.expires_at,
    fao.created_at,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    le.name AS long_exchange_name,
    se.name AS short_exchange_name
FROM funding_arbitrage_opportunities fao
JOIN trading_pairs tp ON fao.trading_pair_id = tp.id
JOIN exchanges le ON fao.long_exchange_id = le.id
JOIN exchanges se ON fao.short_exchange_id = se.id
WHERE fao.is_active = true 
  AND (fao.expires_at IS NULL OR fao.expires_at > CURRENT_TIMESTAMP)
ORDER BY fao.estimated_profit_percentage DESC;

COMMIT;