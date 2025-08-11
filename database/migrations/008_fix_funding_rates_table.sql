-- Migration: 008_fix_funding_rates_table.sql
-- Description: Add missing trading_pair_id column to funding_rates table
-- This fixes the missing trading_pair_id column error

-- Add trading_pair_id column to funding_rates table if it doesn't exist
DO $$
BEGIN
    -- Add trading_pair_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'funding_rates' AND column_name = 'trading_pair_id') THEN
        ALTER TABLE funding_rates ADD COLUMN trading_pair_id INTEGER;
    END IF;
END $$;

-- Create a function to get or create trading pair ID from symbol
CREATE OR REPLACE FUNCTION get_or_create_trading_pair_id(p_symbol VARCHAR(50), p_exchange_id INTEGER)
RETURNS INTEGER AS $$
DECLARE
    v_trading_pair_id INTEGER;
    v_base_currency VARCHAR(10);
    v_quote_currency VARCHAR(10);
BEGIN
    -- First try to find existing trading pair for this exchange and symbol
    SELECT id INTO v_trading_pair_id 
    FROM trading_pairs 
    WHERE exchange_id = p_exchange_id AND symbol = p_symbol;
    
    IF v_trading_pair_id IS NOT NULL THEN
        RETURN v_trading_pair_id;
    END IF;
    
    -- Parse symbol to extract base and quote currencies
    -- Handle common formats: BTC/USDT, BTC/USDT:USDT, BTCUSDT
    IF p_symbol LIKE '%/%' THEN
        -- Format: BTC/USDT or BTC/USDT:USDT
        v_base_currency := SPLIT_PART(SPLIT_PART(p_symbol, ':', 1), '/', 1);
        v_quote_currency := SPLIT_PART(SPLIT_PART(p_symbol, ':', 1), '/', 2);
    ELSE
        -- Assume format like BTCUSDT - try to extract common quote currencies
        IF p_symbol LIKE '%USDT' THEN
            v_base_currency := SUBSTRING(p_symbol FROM 1 FOR LENGTH(p_symbol) - 4);
            v_quote_currency := 'USDT';
        ELSIF p_symbol LIKE '%USD' THEN
            v_base_currency := SUBSTRING(p_symbol FROM 1 FOR LENGTH(p_symbol) - 3);
            v_quote_currency := 'USD';
        ELSIF p_symbol LIKE '%BTC' THEN
            v_base_currency := SUBSTRING(p_symbol FROM 1 FOR LENGTH(p_symbol) - 3);
            v_quote_currency := 'BTC';
        ELSIF p_symbol LIKE '%ETH' THEN
            v_base_currency := SUBSTRING(p_symbol FROM 1 FOR LENGTH(p_symbol) - 3);
            v_quote_currency := 'ETH';
        ELSE
            -- Default fallback
            v_base_currency := SUBSTRING(p_symbol FROM 1 FOR 3);
            v_quote_currency := SUBSTRING(p_symbol FROM 4);
        END IF;
    END IF;
    
    -- Create new trading pair
    INSERT INTO trading_pairs (exchange_id, symbol, base_currency, quote_currency, is_active)
    VALUES (p_exchange_id, p_symbol, v_base_currency, v_quote_currency, true)
    RETURNING id INTO v_trading_pair_id;
    
    RETURN v_trading_pair_id;
END;
$$ LANGUAGE plpgsql;

-- Update existing funding_rates records to populate trading_pair_id
UPDATE funding_rates 
SET trading_pair_id = get_or_create_trading_pair_id(symbol, exchange_id)
WHERE trading_pair_id IS NULL;

-- Make trading_pair_id NOT NULL and add foreign key constraint
DO $$
BEGIN
    -- Make trading_pair_id NOT NULL
    IF EXISTS (SELECT 1 FROM information_schema.columns 
               WHERE table_name = 'funding_rates' AND column_name = 'trading_pair_id') THEN
        ALTER TABLE funding_rates ALTER COLUMN trading_pair_id SET NOT NULL;
        
        -- Add foreign key constraint if it doesn't exist
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint 
            WHERE conname = 'funding_rates_trading_pair_id_fkey'
        ) THEN
            ALTER TABLE funding_rates 
            ADD CONSTRAINT funding_rates_trading_pair_id_fkey 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        END IF;
    END IF;
END $$;

-- Update the unique constraint to include trading_pair_id instead of symbol
DO $$
BEGIN
    -- Drop old unique constraint if it exists
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'funding_rates_exchange_id_symbol_funding_rate_timestamp_key'
    ) THEN
        ALTER TABLE funding_rates DROP CONSTRAINT funding_rates_exchange_id_symbol_funding_rate_timestamp_key;
    END IF;
    
    -- Add new unique constraint with trading_pair_id
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'funding_rates_exchange_trading_pair_timestamp_key'
    ) THEN
        ALTER TABLE funding_rates 
        ADD CONSTRAINT funding_rates_exchange_trading_pair_timestamp_key 
        UNIQUE (exchange_id, trading_pair_id, funding_rate_timestamp);
    END IF;
END $$;

-- Create additional indexes for better performance
CREATE INDEX IF NOT EXISTS idx_funding_rates_trading_pair ON funding_rates(trading_pair_id);
CREATE INDEX IF NOT EXISTS idx_funding_rates_exchange_pair ON funding_rates(exchange_id, trading_pair_id);

-- Drop the helper function as it's no longer needed
DROP FUNCTION IF EXISTS get_or_create_trading_pair_id(VARCHAR(50), INTEGER);