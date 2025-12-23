-- Migration 051: Recreate v_trading_pairs_debug view for diagnostics
-- Safely drop and recreate the view to ensure it exists with a consistent definition

DROP VIEW IF EXISTS v_trading_pairs_debug;

CREATE VIEW v_trading_pairs_debug AS
SELECT 
    tp.id AS trading_pair_id,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    tp.is_active,
    tp.exchange_id,
    e.name AS exchange_name
FROM trading_pairs tp
LEFT JOIN exchanges e ON tp.exchange_id = e.id
ORDER BY e.name, tp.symbol;