-- Migration 037: Fix cache warming queries and ensure trading_pairs schema consistency
-- This migration addresses the column name mismatches found during debugging
-- where cache_warming.go was using base_asset/quote_asset instead of base_currency/quote_currency

-- Ensure trading_pairs table has the correct schema
-- This is a safety check to make sure all required columns exist
DO $$
BEGIN
    -- Check if base_currency column exists, if not add it
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'trading_pairs' AND column_name = 'base_currency' AND table_schema = 'public') THEN
        ALTER TABLE trading_pairs ADD COLUMN base_currency VARCHAR(20);
        -- Update existing rows with default values if needed
        UPDATE trading_pairs SET base_currency = 'UNKNOWN' WHERE base_currency IS NULL;
        -- Now add NOT NULL constraint
        ALTER TABLE trading_pairs ALTER COLUMN base_currency SET NOT NULL;
    ELSE
        -- Column exists, but ensure no NULL values before setting NOT NULL
        UPDATE trading_pairs SET base_currency = 'UNKNOWN' WHERE base_currency IS NULL;
        -- Only set NOT NULL if it's not already set
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'trading_pairs' AND column_name = 'base_currency' 
                   AND table_schema = 'public' AND is_nullable = 'YES') THEN
            ALTER TABLE trading_pairs ALTER COLUMN base_currency SET NOT NULL;
        END IF;
    END IF;
    
    -- Check if quote_currency column exists, if not add it
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'trading_pairs' AND column_name = 'quote_currency' AND table_schema = 'public') THEN
        ALTER TABLE trading_pairs ADD COLUMN quote_currency VARCHAR(20);
        -- Update existing rows with default values if needed
        UPDATE trading_pairs SET quote_currency = 'UNKNOWN' WHERE quote_currency IS NULL;
        -- Now add NOT NULL constraint
        ALTER TABLE trading_pairs ALTER COLUMN quote_currency SET NOT NULL;
    ELSE
        -- Column exists, but ensure no NULL values before setting NOT NULL
        UPDATE trading_pairs SET quote_currency = 'UNKNOWN' WHERE quote_currency IS NULL;
        -- Only set NOT NULL if it's not already set
        IF EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name = 'trading_pairs' AND column_name = 'quote_currency' 
                   AND table_schema = 'public' AND is_nullable = 'YES') THEN
            ALTER TABLE trading_pairs ALTER COLUMN quote_currency SET NOT NULL;
        END IF;
    END IF;
    
    -- Add exchange_id column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'exchange_id'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE trading_pairs ADD COLUMN exchange_id INTEGER;
        RAISE NOTICE 'Added exchange_id column to trading_pairs table';
    ELSE
        RAISE NOTICE 'exchange_id column already exists in trading_pairs table';
    END IF;

    -- Add is_active column if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'is_active'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE trading_pairs ADD COLUMN is_active BOOLEAN DEFAULT true;
        RAISE NOTICE 'Added is_active column to trading_pairs table';
    ELSE
        RAISE NOTICE 'is_active column already exists in trading_pairs table';
    END IF;

    -- Add foreign key constraint if exchanges table exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'exchanges') THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.table_constraints 
            WHERE constraint_name = 'fk_trading_pairs_exchange_id'
            AND table_name = 'trading_pairs'
        ) THEN
            ALTER TABLE trading_pairs ADD CONSTRAINT fk_trading_pairs_exchange_id 
                FOREIGN KEY (exchange_id) REFERENCES exchanges(id);
        END IF;
    END IF;
    
    -- Ensure proper indexes exist for performance
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'trading_pairs' AND indexname = 'idx_trading_pairs_exchange_symbol') THEN
        CREATE INDEX idx_trading_pairs_exchange_symbol ON trading_pairs(exchange_id, symbol);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename = 'trading_pairs' AND indexname = 'idx_trading_pairs_base_quote') THEN
        CREATE INDEX idx_trading_pairs_base_quote ON trading_pairs(base_currency, quote_currency);
    END IF;
END $$;

-- Create a view to help with debugging and verification
CREATE OR REPLACE VIEW v_trading_pairs_debug AS
SELECT 
    tp.id,
    tp.exchange_id,
    e.name as exchange_name,
    tp.symbol,
    tp.base_currency,
    tp.quote_currency,
    tp.is_active,
    tp.created_at,
    tp.updated_at
FROM trading_pairs tp
LEFT JOIN exchanges e ON tp.exchange_id = e.id
ORDER BY e.name, tp.symbol;

-- Add a comment to document this migration
COMMENT ON TABLE trading_pairs IS 'Trading pairs table - fixed schema consistency issues in migration 037';

-- Migration completion handled automatically by migration system