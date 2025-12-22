-- Migration: Add missing mark_price and index_price columns to funding_rates table
-- This fixes the PostgreSQL error: column fr.mark_price does not exist
-- These columns were originally defined in 005_add_funding_rate_support.sql but removed in 006_minimal_schema.sql

-- Add mark_price column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'funding_rates' 
        AND column_name = 'mark_price'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE funding_rates ADD COLUMN mark_price DECIMAL(20, 8);
        RAISE NOTICE 'Added mark_price column to funding_rates table';
    ELSE
        RAISE NOTICE 'mark_price column already exists in funding_rates table';
    END IF;
END $$;

-- Add index_price column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'funding_rates' 
        AND column_name = 'index_price'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE funding_rates ADD COLUMN index_price DECIMAL(20, 8);
        RAISE NOTICE 'Added index_price column to funding_rates table';
    ELSE
        RAISE NOTICE 'index_price column already exists in funding_rates table';
    END IF;
END $$;

-- Verify the timestamp column exists (it should already exist from 006_minimal_schema.sql)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'funding_rates' 
        AND column_name = 'timestamp'
        AND table_schema = 'public'
    ) THEN
        ALTER TABLE funding_rates ADD COLUMN timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
        RAISE NOTICE 'Added timestamp column to funding_rates table';
    ELSE
        RAISE NOTICE 'timestamp column already exists in funding_rates table';
    END IF;
END $$;

-- Create or update indexes for better query performance
-- Index for mark_price queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_funding_rates_mark_price 
ON funding_rates (mark_price) 
WHERE mark_price IS NOT NULL AND mark_price > 0;

-- Index for index_price queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_funding_rates_index_price 
ON funding_rates (index_price) 
WHERE index_price IS NOT NULL AND index_price > 0;

-- Composite index for timestamp and mark_price queries (used in arbitrage calculations)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_funding_rates_timestamp_mark_price 
ON funding_rates (timestamp DESC, mark_price) 
WHERE mark_price IS NOT NULL AND mark_price > 0;