-- Migration: Fix index creation errors with IMMUTABLE function issues
-- This migration fixes indexes that use non-immutable functions like NOW() in WHERE clauses

-- Drop the problematic index that uses NOW() function
DROP INDEX IF EXISTS idx_market_data_timestamp_range;

-- Recreate the index without the non-immutable function
-- Instead of using NOW() in the WHERE clause, we'll create a simple index on timestamp
-- The application can handle the time-based filtering in queries
CREATE INDEX IF NOT EXISTS idx_market_data_timestamp_desc ON market_data(timestamp DESC);

-- Drop and recreate other potentially problematic indexes
DROP INDEX IF EXISTS idx_aggregated_signals_active;
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_expires_at ON aggregated_signals(expires_at DESC);

-- Fix funding rates indexes
DROP INDEX IF EXISTS idx_funding_arbitrage_expires;
CREATE INDEX IF NOT EXISTS idx_funding_arbitrage_expires_at ON funding_arbitrage_opportunities(expires_at DESC);

-- Add a comment explaining the change
COMMENT ON INDEX idx_market_data_timestamp_desc IS 'Index for market data timestamp ordering - application handles time-based filtering';
COMMENT ON INDEX idx_aggregated_signals_expires_at IS 'Index for aggregated signals expiration - application handles active signal filtering';
COMMENT ON INDEX idx_funding_arbitrage_expires_at IS 'Index for funding arbitrage expiration - application handles active opportunity filtering';