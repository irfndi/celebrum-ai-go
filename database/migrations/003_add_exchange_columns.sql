-- Migration: Add missing columns to exchanges table
-- Created: 2025-08-10
-- Description: Add has_spot and has_futures columns to exchanges table

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
UPDATE exchanges 
SET ccxt_id = LOWER(name)
WHERE ccxt_id IS NULL;

-- Add status column if it doesn't exist (referenced in error logs)
ALTER TABLE exchanges 
ADD COLUMN IF NOT EXISTS status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'maintenance'));

-- Create indexes for new columns
CREATE INDEX IF NOT EXISTS idx_exchanges_has_spot ON exchanges(has_spot);
CREATE INDEX IF NOT EXISTS idx_exchanges_has_futures ON exchanges(has_futures);
CREATE INDEX IF NOT EXISTS idx_exchanges_status ON exchanges(status);