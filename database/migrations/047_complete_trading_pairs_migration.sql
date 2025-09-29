-- Migration: Complete trading_pairs migration
-- This migration completes the work started in migration 029 by updating all tables
-- that reference trading_pairs but were missed in the original migration

-- Create a function to safely update trading_pair_id references
CREATE OR REPLACE FUNCTION update_trading_pair_references()
RETURNS void AS $$
DECLARE
    update_count integer;
BEGIN
    -- Check if we have a mapping table from a previous migration
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'trading_pair_id_mapping') THEN
        RAISE NOTICE 'Found existing trading_pair_id_mapping table, using it for updates';
        
        -- Update funding_rates table if it exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'funding_rates') THEN
            UPDATE funding_rates 
            SET trading_pair_id = mapping.new_id::integer
            FROM trading_pair_id_mapping mapping
            WHERE funding_rates.trading_pair_id::text = mapping.old_id::text;
            
            GET DIAGNOSTICS update_count = ROW_COUNT;
            RAISE NOTICE 'Updated % rows in funding_rates table', update_count;
        END IF;
        
        -- Update funding_arbitrage_opportunities table if it exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'funding_arbitrage_opportunities') THEN
            UPDATE funding_arbitrage_opportunities 
            SET trading_pair_id = mapping.new_id::integer
            FROM trading_pair_id_mapping mapping
            WHERE funding_arbitrage_opportunities.trading_pair_id::text = mapping.old_id::text;
            
            GET DIAGNOSTICS update_count = ROW_COUNT;
            RAISE NOTICE 'Updated % rows in funding_arbitrage_opportunities table', update_count;
        END IF;
        
        -- Update exchange_trading_pairs table if it exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'exchange_trading_pairs') THEN
            UPDATE exchange_trading_pairs 
            SET trading_pair_id = mapping.new_id::integer
            FROM trading_pair_id_mapping mapping
            WHERE exchange_trading_pairs.trading_pair_id::text = mapping.old_id::text;
            
            GET DIAGNOSTICS update_count = ROW_COUNT;
            RAISE NOTICE 'Updated % rows in exchange_trading_pairs table', update_count;
        END IF;
        
        -- Update order_book_data table if it exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'order_book_data') THEN
            UPDATE order_book_data 
            SET trading_pair_id = mapping.new_id::integer
            FROM trading_pair_id_mapping mapping
            WHERE order_book_data.trading_pair_id::text = mapping.old_id::text;
            
            GET DIAGNOSTICS update_count = ROW_COUNT;
            RAISE NOTICE 'Updated % rows in order_book_data table', update_count;
        END IF;
        
        -- Update ohlcv_data table if it exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'ohlcv_data') THEN
            UPDATE ohlcv_data 
            SET trading_pair_id = mapping.new_id::integer
            FROM trading_pair_id_mapping mapping
            WHERE ohlcv_data.trading_pair_id::text = mapping.old_id::text;
            
            GET DIAGNOSTICS update_count = ROW_COUNT;
            RAISE NOTICE 'Updated % rows in ohlcv_data table', update_count;
        END IF;
        
        -- Update arbitrage_opportunities table if it exists
        IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'arbitrage_opportunities') THEN
            UPDATE arbitrage_opportunities 
            SET trading_pair_id = mapping.new_id::integer
            FROM trading_pair_id_mapping mapping
            WHERE arbitrage_opportunities.trading_pair_id::text = mapping.old_id::text;
            
            GET DIAGNOSTICS update_count = ROW_COUNT;
            RAISE NOTICE 'Updated % rows in arbitrage_opportunities table', update_count;
        END IF;
        
        -- Skip user_alerts table - it doesn't have a trading_pair_id column
        -- user_alerts uses alert conditions in JSONB format instead
        
        -- Skip futures_arbitrage_opportunities table - it uses 'symbol' column instead of trading_pair_id
        
    ELSE
        RAISE NOTICE 'No trading_pair_id_mapping table found - migration 029 may have already completed successfully';
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Execute the update function
SELECT update_trading_pair_references();

-- Drop the function after use
DROP FUNCTION update_trading_pair_references();

-- Add foreign key constraints for tables that reference trading_pairs
-- This ensures referential integrity going forward

-- Add foreign key constraint for funding_rates if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'funding_rates') THEN
        -- Drop existing constraint if it exists
        ALTER TABLE funding_rates DROP CONSTRAINT IF EXISTS fk_funding_rates_trading_pair;
        
        -- Add new constraint
        ALTER TABLE funding_rates ADD CONSTRAINT fk_funding_rates_trading_pair 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Added foreign key constraint for funding_rates.trading_pair_id';
    END IF;
END $$;

-- Add foreign key constraint for funding_arbitrage_opportunities if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'funding_arbitrage_opportunities') THEN
        -- Drop existing constraint if it exists
        ALTER TABLE funding_arbitrage_opportunities DROP CONSTRAINT IF EXISTS fk_funding_arbitrage_opportunities_trading_pair;
        
        -- Add new constraint
        ALTER TABLE funding_arbitrage_opportunities ADD CONSTRAINT fk_funding_arbitrage_opportunities_trading_pair 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Added foreign key constraint for funding_arbitrage_opportunities.trading_pair_id';
    END IF;
END $$;

-- Add foreign key constraint for exchange_trading_pairs if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'exchange_trading_pairs') THEN
        -- Drop existing constraint if it exists
        ALTER TABLE exchange_trading_pairs DROP CONSTRAINT IF EXISTS fk_exchange_trading_pairs_trading_pair;
        
        -- Add new constraint
        ALTER TABLE exchange_trading_pairs ADD CONSTRAINT fk_exchange_trading_pairs_trading_pair 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Added foreign key constraint for exchange_trading_pairs.trading_pair_id';
    END IF;
END $$;

-- Add foreign key constraint for order_book_data if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'order_book_data') THEN
        -- Drop existing constraint if it exists
        ALTER TABLE order_book_data DROP CONSTRAINT IF EXISTS fk_order_book_data_trading_pair;
        
        -- Add new constraint
        ALTER TABLE order_book_data ADD CONSTRAINT fk_order_book_data_trading_pair 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Added foreign key constraint for order_book_data.trading_pair_id';
    END IF;
END $$;

-- Add foreign key constraint for ohlcv_data if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'ohlcv_data') THEN
        -- Drop existing constraint if it exists
        ALTER TABLE ohlcv_data DROP CONSTRAINT IF EXISTS fk_ohlcv_data_trading_pair;
        
        -- Add new constraint
        ALTER TABLE ohlcv_data ADD CONSTRAINT fk_ohlcv_data_trading_pair 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Added foreign key constraint for ohlcv_data.trading_pair_id';
    END IF;
END $$;

-- Add foreign key constraint for arbitrage_opportunities if table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE information_schema.tables.table_name = 'arbitrage_opportunities') THEN
        -- Drop existing constraint if it exists
        ALTER TABLE arbitrage_opportunities DROP CONSTRAINT IF EXISTS fk_arbitrage_opportunities_trading_pair;
        
        -- Add new constraint
        ALTER TABLE arbitrage_opportunities ADD CONSTRAINT fk_arbitrage_opportunities_trading_pair 
            FOREIGN KEY (trading_pair_id) REFERENCES trading_pairs(id) ON DELETE CASCADE;
        
        RAISE NOTICE 'Added foreign key constraint for arbitrage_opportunities.trading_pair_id';
    END IF;
END $$;

-- Skip user_alerts - it doesn't have a trading_pair_id column

-- Skip futures_arbitrage_opportunities - it uses 'symbol' column instead of trading_pair_id

-- Clean up any temporary mapping table if it still exists
DROP TABLE IF EXISTS trading_pair_id_mapping;
DROP TABLE IF EXISTS id_mapping;

-- Migration 047 completed the trading_pairs schema migration by updating all referencing tables
-- (Schema comment skipped due to ownership requirements)

-- Migration completion is handled automatically by the migration system