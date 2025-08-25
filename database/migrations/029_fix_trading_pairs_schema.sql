-- Migration 029: Fix trading_pairs table schema
-- Add exchange_id column and fix primary key type

BEGIN;

-- Check if trading_pairs table exists and has the correct structure
DO $$
BEGIN
    -- Check if exchange_id column exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'exchange_id'
        AND table_schema = 'public'
    ) THEN
        -- Add exchange_id column if it doesn't exist
        ALTER TABLE trading_pairs ADD COLUMN exchange_id INTEGER;
        
        -- Add updated_at column if it doesn't exist (needed before UPDATE due to trigger)
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns 
            WHERE table_name = 'trading_pairs' 
            AND column_name = 'updated_at'
            AND table_schema = 'public'
        ) THEN
            ALTER TABLE trading_pairs ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
            RAISE NOTICE 'Added updated_at column to trading_pairs table';
        END IF;
        
        -- Set a default exchange_id for existing records (use first exchange)
        UPDATE trading_pairs SET exchange_id = (
            SELECT id FROM exchanges ORDER BY id LIMIT 1
        ) WHERE exchange_id IS NULL;
        
        -- Make exchange_id NOT NULL and add foreign key constraint
        ALTER TABLE trading_pairs ALTER COLUMN exchange_id SET NOT NULL;
        ALTER TABLE trading_pairs ADD CONSTRAINT fk_trading_pairs_exchange 
            FOREIGN KEY (exchange_id) REFERENCES exchanges(id) ON DELETE CASCADE;
            
        RAISE NOTICE 'Added exchange_id column to trading_pairs table';
    END IF;
    
    -- Check if primary key is UUID type and needs to be changed to SERIAL
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'id'
        AND data_type = 'uuid'
        AND table_schema = 'public'
    ) THEN
        -- Drop existing constraints and indexes that depend on the id column
        ALTER TABLE market_data DROP CONSTRAINT IF EXISTS market_data_trading_pair_id_fkey;
        ALTER TABLE technical_indicators DROP CONSTRAINT IF EXISTS technical_indicators_trading_pair_id_fkey;
        
        -- Create a new temporary table with the correct structure
        -- Fixed: Use VARCHAR(20) for base_currency and quote_currency to match schema consistency
        CREATE TABLE trading_pairs_new (
            id SERIAL PRIMARY KEY,
            exchange_id INTEGER NOT NULL REFERENCES exchanges(id) ON DELETE CASCADE,
            symbol VARCHAR(50) NOT NULL,
            base_currency VARCHAR(20) NOT NULL,
            quote_currency VARCHAR(20) NOT NULL,
            is_active BOOLEAN DEFAULT true,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
            UNIQUE(exchange_id, symbol)
        );
        
        -- Copy data from old table to new table (assign new sequential IDs)
        INSERT INTO trading_pairs_new (exchange_id, symbol, base_currency, quote_currency, is_active, created_at, updated_at)
        SELECT 
            COALESCE(exchange_id, (SELECT id FROM exchanges ORDER BY id LIMIT 1)),
            symbol, 
            base_currency, 
            quote_currency, 
            COALESCE(is_active, true),
            COALESCE(created_at, NOW()),
            COALESCE(updated_at, NOW())
        FROM trading_pairs;
        
        -- Create a mapping table for old UUID to new integer IDs
        CREATE TEMP TABLE id_mapping AS
        SELECT 
            old.id as old_id,
            new.id as new_id
        FROM trading_pairs old
        JOIN trading_pairs_new new ON (
            old.symbol = new.symbol AND 
            COALESCE(old.exchange_id, (SELECT id FROM exchanges ORDER BY id LIMIT 1)) = new.exchange_id
        );
        
        -- Update foreign key references in market_data
        UPDATE market_data SET trading_pair_id = (
            SELECT new_id FROM id_mapping WHERE old_id::text = trading_pair_id::text
        ) WHERE trading_pair_id::text IN (SELECT old_id::text FROM id_mapping);
        
        -- Update foreign key references in technical_indicators
        UPDATE technical_indicators SET trading_pair_id = (
            SELECT new_id FROM id_mapping WHERE old_id::text = trading_pair_id::text
        ) WHERE trading_pair_id::text IN (SELECT old_id::text FROM id_mapping);
        
        -- Drop old table and rename new table
        DROP TABLE trading_pairs;
        ALTER TABLE trading_pairs_new RENAME TO trading_pairs;
        
        -- Recreate foreign key constraints
        ALTER TABLE market_data ADD CONSTRAINT market_data_trading_pair_id_fkey 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        ALTER TABLE technical_indicators ADD CONSTRAINT technical_indicators_trading_pair_id_fkey 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
            
        RAISE NOTICE 'Converted trading_pairs primary key from UUID to SERIAL';
    END IF;
    
    -- Ensure unique constraint exists on exchange_id and symbol
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'trading_pairs_exchange_id_symbol_key'
    ) THEN
        ALTER TABLE trading_pairs ADD CONSTRAINT trading_pairs_exchange_id_symbol_key 
            UNIQUE(exchange_id, symbol);
        RAISE NOTICE 'Added unique constraint on exchange_id and symbol';
    END IF;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_trading_pairs_exchange ON trading_pairs(exchange_id);
CREATE INDEX IF NOT EXISTS idx_trading_pairs_symbol ON trading_pairs(symbol);

-- Create index on is_active only if the column exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'is_active'
        AND table_schema = 'public'
    ) THEN
        CREATE INDEX IF NOT EXISTS idx_trading_pairs_active ON trading_pairs(is_active) WHERE is_active = true;
    END IF;
END $$;

-- Add updated_at column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'updated_at'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE trading_pairs ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
        RAISE NOTICE 'Added updated_at column to trading_pairs table';
    END IF;
END $$;

-- Add updated_at trigger if it doesn't exist
DROP TRIGGER IF EXISTS update_trading_pairs_updated_at ON trading_pairs;
CREATE TRIGGER update_trading_pairs_updated_at BEFORE UPDATE ON trading_pairs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migration already recorded in schema_migrations table

COMMIT;