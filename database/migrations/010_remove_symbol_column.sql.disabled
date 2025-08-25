-- Migration: 010_remove_symbol_column.sql
-- Description: Remove the redundant symbol column from funding_rates table
-- The trading_pair_id column now serves this purpose

-- Drop indexes that reference the symbol column
DROP INDEX IF EXISTS idx_funding_rates_exchange_symbol;

-- Remove the symbol column from funding_rates table
ALTER TABLE funding_rates DROP COLUMN IF EXISTS symbol;