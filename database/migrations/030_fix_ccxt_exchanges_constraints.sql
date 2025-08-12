-- Migration 030: Fix ccxt_exchanges foreign key constraints
-- Add CASCADE options to prevent constraint violations

BEGIN;

-- Check if ccxt_exchanges table exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ccxt_exchanges' AND table_schema = 'public') THEN
        -- Drop existing foreign key constraint if it exists
        IF EXISTS (
            SELECT 1 FROM pg_constraint 
            WHERE conname = 'ccxt_exchanges_exchange_id_fkey'
        ) THEN
            ALTER TABLE ccxt_exchanges DROP CONSTRAINT ccxt_exchanges_exchange_id_fkey;
            RAISE NOTICE 'Dropped existing ccxt_exchanges_exchange_id_fkey constraint';
        END IF;
        
        -- Add new foreign key constraint with CASCADE options
        ALTER TABLE ccxt_exchanges 
        ADD CONSTRAINT ccxt_exchanges_exchange_id_fkey 
        FOREIGN KEY (exchange_id) REFERENCES exchanges(id) ON DELETE CASCADE ON UPDATE CASCADE;
        
        RAISE NOTICE 'Added ccxt_exchanges foreign key constraint with CASCADE options';
        
        -- Clean up any orphaned records in ccxt_exchanges that don't have matching exchanges
        DELETE FROM ccxt_exchanges 
        WHERE exchange_id NOT IN (SELECT id FROM exchanges);
        
        RAISE NOTICE 'Cleaned up orphaned ccxt_exchanges records';
        
    ELSE
        -- If ccxt_exchanges table doesn't exist, we can consider this migration complete
        -- since the newer schema (006_minimal_schema.sql) doesn't include this table
        RAISE NOTICE 'ccxt_exchanges table does not exist - migration not needed';
    END IF;
END $$;

-- Alternative approach: If we want to completely remove the ccxt_exchanges table
-- since it's not part of the current schema design, uncomment the following:

/*
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ccxt_exchanges' AND table_schema = 'public') THEN
        -- Drop the entire ccxt_exchanges table since it's not in the current schema
        DROP TABLE ccxt_exchanges CASCADE;
        RAISE NOTICE 'Dropped ccxt_exchanges table as it is not part of current schema';
    END IF;
END $$;
*/

-- Record this migration
INSERT INTO schema_migrations (version, filename, description) VALUES
    (30, '030_fix_ccxt_exchanges_constraints.sql', 'Fix ccxt_exchanges foreign key constraints with CASCADE options')
ON CONFLICT (version) DO NOTHING;

COMMIT;