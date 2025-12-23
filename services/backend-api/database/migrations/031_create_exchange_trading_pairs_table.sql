-- Migration 031: Add blacklist functionality to exchange_trading_pairs table
-- This migration adds blacklisting and error tracking functionality to the existing table

BEGIN;

-- Add blacklist-related columns to the existing exchange_trading_pairs table
-- Fixed: Use VARCHAR(20) for base_currency and quote_currency to match schema consistency
ALTER TABLE exchange_trading_pairs 
    ADD COLUMN IF NOT EXISTS symbol VARCHAR(50),
    ADD COLUMN IF NOT EXISTS base_currency VARCHAR(20),
    ADD COLUMN IF NOT EXISTS quote_currency VARCHAR(20),
    ADD COLUMN IF NOT EXISTS is_blacklisted BOOLEAN DEFAULT false,
    ADD COLUMN IF NOT EXISTS blacklist_reason TEXT,
    ADD COLUMN IF NOT EXISTS blacklisted_at TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS blacklisted_until TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS last_price_update TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS last_successful_fetch TIMESTAMP WITH TIME ZONE,
    ADD COLUMN IF NOT EXISTS error_count INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error TEXT,
    ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMP WITH TIME ZONE;

-- Update symbol column from exchange_symbol if it exists and symbol is null
UPDATE exchange_trading_pairs 
SET symbol = exchange_symbol 
WHERE symbol IS NULL AND exchange_symbol IS NOT NULL;

-- Update base_currency and quote_currency from trading_pairs if they are null
UPDATE exchange_trading_pairs etp
SET 
    base_currency = tp.base_currency,
    quote_currency = tp.quote_currency
FROM trading_pairs tp
WHERE etp.trading_pair_id = tp.id 
  AND (etp.base_currency IS NULL OR etp.quote_currency IS NULL);

-- Add constraints if they don't exist
DO $$
BEGIN
    -- Add unique constraint on exchange and symbol for faster lookups
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'unique_exchange_symbol' 
        AND conrelid = 'exchange_trading_pairs'::regclass
    ) THEN
        ALTER TABLE exchange_trading_pairs 
        ADD CONSTRAINT unique_exchange_symbol UNIQUE(exchange_id, symbol);
    END IF;
EXCEPTION
    WHEN duplicate_table THEN
        -- Constraint already exists, ignore
        NULL;
END $$;

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_exchange_trading_pairs_symbol ON exchange_trading_pairs(symbol) WHERE symbol IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_exchange_trading_pairs_blacklisted ON exchange_trading_pairs(is_blacklisted) WHERE is_blacklisted = true;
CREATE INDEX IF NOT EXISTS idx_exchange_trading_pairs_error_count ON exchange_trading_pairs(error_count) WHERE error_count > 0;
CREATE INDEX IF NOT EXISTS idx_exchange_trading_pairs_last_update ON exchange_trading_pairs(last_price_update) WHERE last_price_update IS NOT NULL;

-- Add updated_at trigger
DROP TRIGGER IF EXISTS update_exchange_trading_pairs_updated_at ON exchange_trading_pairs;
CREATE TRIGGER update_exchange_trading_pairs_updated_at BEFORE UPDATE ON exchange_trading_pairs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Update existing records with missing data from trading_pairs
UPDATE exchange_trading_pairs etp
SET 
    symbol = COALESCE(etp.symbol, tp.symbol),
    base_currency = COALESCE(etp.base_currency, tp.base_currency),
    quote_currency = COALESCE(etp.quote_currency, tp.quote_currency)
FROM trading_pairs tp
WHERE etp.trading_pair_id = tp.id 
  AND (etp.symbol IS NULL OR etp.base_currency IS NULL OR etp.quote_currency IS NULL);

-- Create a view for easy querying of active exchange-trading pair relationships
CREATE OR REPLACE VIEW active_exchange_trading_pairs AS
SELECT 
    etp.id,
    e.name as exchange_name,
    e.display_name as exchange_display_name,
    e.ccxt_id as exchange_ccxt_id,
    COALESCE(etp.symbol, tp.symbol) as symbol,
    COALESCE(etp.base_currency, tp.base_currency) as base_currency,
    COALESCE(etp.quote_currency, tp.quote_currency) as quote_currency,
    etp.is_active,
    etp.is_blacklisted,
    etp.blacklist_reason,
    etp.error_count,
    etp.last_price_update,
    etp.last_successful_fetch,
    etp.last_error,
    etp.last_error_at
FROM exchange_trading_pairs etp
JOIN exchanges e ON etp.exchange_id = e.id
JOIN trading_pairs tp ON etp.trading_pair_id = tp.id
WHERE etp.is_active = true 
  AND e.is_active = true;

-- Create a view for blacklisted trading pairs
CREATE OR REPLACE VIEW blacklisted_exchange_trading_pairs AS
SELECT 
    etp.id,
    e.name as exchange_name,
    e.display_name as exchange_display_name,
    e.ccxt_id as exchange_ccxt_id,
    COALESCE(etp.symbol, tp.symbol) as symbol,
    COALESCE(etp.base_currency, tp.base_currency) as base_currency,
    COALESCE(etp.quote_currency, tp.quote_currency) as quote_currency,
    etp.blacklist_reason,
    etp.blacklisted_at,
    etp.blacklisted_until,
    etp.error_count,
    etp.last_error,
    etp.last_error_at
FROM exchange_trading_pairs etp
JOIN exchanges e ON etp.exchange_id = e.id
JOIN trading_pairs tp ON etp.trading_pair_id = tp.id
WHERE etp.is_blacklisted = true;

-- Create blacklist configuration table
CREATE TABLE IF NOT EXISTS blacklist_config (
    id INTEGER PRIMARY KEY DEFAULT 1,
    error_threshold INTEGER DEFAULT 10,
    blacklist_duration INTERVAL DEFAULT '1 hour',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT single_config_row CHECK (id = 1)
);

-- Insert default configuration
INSERT INTO blacklist_config (id, error_threshold, blacklist_duration)
VALUES (1, 10, '1 hour'::INTERVAL)
ON CONFLICT (id) DO NOTHING;

-- Function to automatically blacklist trading pairs with high error counts
DO $$
BEGIN
    -- Only create function if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM pg_proc WHERE proname = 'auto_blacklist_high_error_pairs') THEN
        EXECUTE '
        CREATE FUNCTION auto_blacklist_high_error_pairs()
        RETURNS TRIGGER AS $func$
        DECLARE
            v_error_threshold INTEGER;
            v_blacklist_duration INTERVAL;
        BEGIN
            -- Get configurable thresholds
            SELECT error_threshold, blacklist_duration
            INTO v_error_threshold, v_blacklist_duration
            FROM blacklist_config WHERE id = 1;
            
            -- Auto-blacklist if error count reaches threshold
            IF NEW.error_count >= v_error_threshold AND OLD.is_blacklisted = false THEN
                NEW.is_blacklisted = true;
                NEW.blacklist_reason = ''Auto-blacklisted due to high error count ('' || NEW.error_count || '')'';
                NEW.blacklisted_at = NOW();
                NEW.blacklisted_until = NOW() + v_blacklist_duration;
            END IF;
            
            -- Auto-unblacklist if blacklist period has expired
            IF NEW.is_blacklisted = true AND NEW.blacklisted_until IS NOT NULL AND NOW() > NEW.blacklisted_until THEN
                NEW.is_blacklisted = false;
                NEW.blacklist_reason = NULL;
                NEW.blacklisted_at = NULL;
                NEW.blacklisted_until = NULL;
                NEW.error_count = 0; -- Reset error count
            END IF;
            
            RETURN NEW;
        END;
        $func$ LANGUAGE plpgsql';
        RAISE NOTICE 'Created auto_blacklist_high_error_pairs function';
    ELSE
        RAISE NOTICE 'Function auto_blacklist_high_error_pairs already exists';
    END IF;
END $$;

-- Create trigger for auto-blacklisting
DROP TRIGGER IF EXISTS auto_blacklist_trigger ON exchange_trading_pairs;
CREATE TRIGGER auto_blacklist_trigger
    BEFORE UPDATE ON exchange_trading_pairs
    FOR EACH ROW
    EXECUTE FUNCTION auto_blacklist_high_error_pairs();

COMMIT;