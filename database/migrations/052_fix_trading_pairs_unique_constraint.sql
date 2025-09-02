-- Migration 052: Fix trading_pairs unique constraints to support per-exchange symbols
-- Context: Earlier schema set UNIQUE(symbol) which conflicts with desired UNIQUE(exchange_id, symbol)
-- This migration drops the legacy unique constraint on symbol-only and ensures the composite unique exists.

BEGIN;

-- Drop legacy unique constraint on symbol if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE t.relname = 'trading_pairs' AND c.conname = 'trading_pairs_symbol_key'
    ) THEN
        ALTER TABLE trading_pairs DROP CONSTRAINT trading_pairs_symbol_key;
        RAISE NOTICE 'Dropped legacy unique constraint trading_pairs_symbol_key';
    ELSE
        RAISE NOTICE 'Legacy unique constraint trading_pairs_symbol_key not found, skipping';
    END IF;
END
$$;

-- Ensure non-unique index on symbol exists for lookups
CREATE INDEX IF NOT EXISTS idx_trading_pairs_symbol ON trading_pairs(symbol);

-- Ensure composite unique constraint exists on (exchange_id, symbol)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint c
        JOIN pg_class t ON c.conrelid = t.oid
        WHERE t.relname = 'trading_pairs' AND c.conname = 'trading_pairs_exchange_id_symbol_key'
    ) THEN
        ALTER TABLE trading_pairs ADD CONSTRAINT trading_pairs_exchange_id_symbol_key UNIQUE(exchange_id, symbol);
        RAISE NOTICE 'Added composite unique constraint trading_pairs_exchange_id_symbol_key';
    ELSE
        RAISE NOTICE 'Composite unique constraint trading_pairs_exchange_id_symbol_key already exists, skipping';
    END IF;
END
$$;

COMMIT;