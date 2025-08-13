-- Migration: Add is_active column to trading_pairs table
-- This fixes the PostgreSQL error where INSERT statements fail due to missing is_active column

-- Add is_active column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'is_active'
    ) THEN
        ALTER TABLE trading_pairs 
        ADD COLUMN is_active BOOLEAN DEFAULT true;
        
        -- Update existing records to have is_active = true
        UPDATE trading_pairs SET is_active = true WHERE is_active IS NULL;
        
        RAISE NOTICE 'Added is_active column to trading_pairs table';
    ELSE
        RAISE NOTICE 'is_active column already exists in trading_pairs table';
    END IF;
END $$;

-- Create index on is_active for better query performance
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_trading_pairs_is_active 
ON trading_pairs (is_active) 
WHERE is_active = true;