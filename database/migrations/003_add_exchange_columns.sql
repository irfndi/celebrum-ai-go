-- Migration: Add missing columns to exchanges table
-- Created: 2025-08-10
-- Description: Add has_spot and has_futures columns to exchanges table

BEGIN;

-- Add has_spot column to exchanges table
ALTER TABLE exchanges 
ADD COLUMN IF NOT EXISTS has_spot BOOLEAN DEFAULT true;

-- Add has_futures column to exchanges table
ALTER TABLE exchanges 
ADD COLUMN IF NOT EXISTS has_futures BOOLEAN DEFAULT false;

-- Add display_name column to exchanges table (also missing based on error logs)
ALTER TABLE exchanges 
ADD COLUMN IF NOT EXISTS display_name VARCHAR(100);

-- Update existing exchanges with default values
UPDATE exchanges 
SET has_spot = true, 
    has_futures = false,
    display_name = INITCAP(name)
WHERE has_spot IS NULL OR has_futures IS NULL OR display_name IS NULL;

-- Add ccxt_id column if it doesn't exist (referenced in error logs)
ALTER TABLE exchanges 
ADD COLUMN IF NOT EXISTS ccxt_id VARCHAR(50);

-- Update ccxt_id for existing exchanges based on name
-- Handle potential duplicates by appending a suffix
UPDATE exchanges 
SET ccxt_id = LOWER(name)
WHERE ccxt_id IS NULL;

-- Ensure no duplicate ccxt_id values exist before creating unique index
-- This will update duplicates by appending _dupN suffix
WITH duplicate_cte AS (
    SELECT id, ccxt_id, 
           ROW_NUMBER() OVER (PARTITION BY ccxt_id ORDER BY id) as rn
    FROM exchanges
    WHERE ccxt_id IS NOT NULL
)
UPDATE exchanges 
SET ccxt_id = ccxt_id || '_dup' || (rn - 1)
FROM duplicate_cte
WHERE exchanges.id = duplicate_cte.id AND rn > 1;

-- Add status column if it doesn't exist (referenced in error logs)
-- Remove inline CHECK constraint and handle it separately
ALTER TABLE exchanges 
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active';

-- Add named CHECK constraint for status column, only if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'exchanges_status_check' AND conrelid = 'exchanges'::regclass
    ) THEN
        ALTER TABLE exchanges 
        ADD CONSTRAINT exchanges_status_check 
        CHECK (status IN ('active', 'inactive', 'maintenance'));
    END IF;
END $$;

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_exchanges_has_spot ON exchanges(has_spot);
CREATE INDEX IF NOT EXISTS idx_exchanges_has_futures ON exchanges(has_futures);
CREATE INDEX IF NOT EXISTS idx_exchanges_status ON exchanges(status);

-- Create unique index for ccxt_id only after ensuring uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_exchanges_ccxt_id_unique ON exchanges(ccxt_id) WHERE ccxt_id IS NOT NULL;

COMMIT;