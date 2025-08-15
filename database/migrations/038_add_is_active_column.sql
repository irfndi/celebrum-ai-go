-- Migration: Add is_active column to trading_pairs table
-- This migration ensures the trading_pairs table has the is_active column
-- Safe to run multiple times

DO $$
BEGIN
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

    -- Ensure the column has the correct default value
    ALTER TABLE trading_pairs ALTER COLUMN is_active SET DEFAULT true;
    
    -- Update any NULL values to true (in case column existed but had NULLs)
    UPDATE trading_pairs SET is_active = true WHERE is_active IS NULL;
    
    -- Add NOT NULL constraint if not already present
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'trading_pairs' 
        AND column_name = 'is_active'
        AND is_nullable = 'YES'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE trading_pairs ALTER COLUMN is_active SET NOT NULL;
        RAISE NOTICE 'Set is_active column to NOT NULL';
    END IF;

END $$;

-- Add comment to document the column purpose
COMMENT ON COLUMN trading_pairs.is_active IS 'Indicates whether this trading pair is currently active and should be processed';