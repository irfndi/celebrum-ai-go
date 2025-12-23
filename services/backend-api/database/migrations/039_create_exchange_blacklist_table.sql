-- Migration: Create exchange_blacklist table for persistent blacklist storage
-- This migration creates the exchange_blacklist table to store blacklisted exchanges
-- with expiration support and audit trail

DO $$
BEGIN
    -- Create exchange_blacklist table if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_name = 'exchange_blacklist' 
        AND table_schema = 'public'
    ) THEN
        CREATE TABLE exchange_blacklist (
            id BIGSERIAL PRIMARY KEY,
            exchange_name VARCHAR(100) NOT NULL,
            reason TEXT NOT NULL,
            created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
            updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
            expires_at TIMESTAMP WITH TIME ZONE,
            is_active BOOLEAN DEFAULT true NOT NULL
        );
        
        RAISE NOTICE 'Created exchange_blacklist table';
    ELSE
        RAISE NOTICE 'exchange_blacklist table already exists';
    END IF;

    -- Create unique index on exchange_name for active entries
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'exchange_blacklist' 
        AND indexname = 'idx_exchange_blacklist_active_exchange'
    ) THEN
        CREATE UNIQUE INDEX idx_exchange_blacklist_active_exchange 
        ON exchange_blacklist (exchange_name) 
        WHERE is_active = true;
        
        RAISE NOTICE 'Created unique index on active exchange names';
    END IF;

    -- Create index on expires_at for cleanup operations
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'exchange_blacklist' 
        AND indexname = 'idx_exchange_blacklist_expires_at'
    ) THEN
        CREATE INDEX idx_exchange_blacklist_expires_at 
        ON exchange_blacklist (expires_at) 
        WHERE expires_at IS NOT NULL;
        
        RAISE NOTICE 'Created index on expires_at column';
    END IF;

    -- Create index on created_at for history queries
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes 
        WHERE tablename = 'exchange_blacklist' 
        AND indexname = 'idx_exchange_blacklist_created_at'
    ) THEN
        CREATE INDEX idx_exchange_blacklist_created_at 
        ON exchange_blacklist (created_at DESC);
        
        RAISE NOTICE 'Created index on created_at column';
    END IF;

    -- Create function to automatically update updated_at timestamp
    IF NOT EXISTS (
        SELECT 1 FROM pg_proc 
        WHERE proname = 'update_exchange_blacklist_updated_at'
    ) THEN
        CREATE OR REPLACE FUNCTION update_exchange_blacklist_updated_at()
        RETURNS TRIGGER AS $func$
        BEGIN
            NEW.updated_at = NOW();
            RETURN NEW;
        END;
        $func$ LANGUAGE plpgsql;
        
        RAISE NOTICE 'Created update_exchange_blacklist_updated_at function';
    END IF;

    -- Create trigger to automatically update updated_at
    IF NOT EXISTS (
        SELECT 1 FROM pg_trigger 
        WHERE tgname = 'trigger_exchange_blacklist_updated_at'
    ) THEN
        CREATE TRIGGER trigger_exchange_blacklist_updated_at
            BEFORE UPDATE ON exchange_blacklist
            FOR EACH ROW
            EXECUTE FUNCTION update_exchange_blacklist_updated_at();
        
        RAISE NOTICE 'Created trigger for automatic updated_at updates';
    END IF;

END $$;

-- Add comments to document the table and columns
COMMENT ON TABLE exchange_blacklist IS 'Stores blacklisted exchanges with expiration support and audit trail';
COMMENT ON COLUMN exchange_blacklist.id IS 'Primary key for the blacklist entry';
COMMENT ON COLUMN exchange_blacklist.exchange_name IS 'Name of the blacklisted exchange';
COMMENT ON COLUMN exchange_blacklist.reason IS 'Reason for blacklisting the exchange';
COMMENT ON COLUMN exchange_blacklist.created_at IS 'Timestamp when the blacklist entry was created';
COMMENT ON COLUMN exchange_blacklist.updated_at IS 'Timestamp when the blacklist entry was last updated';
COMMENT ON COLUMN exchange_blacklist.expires_at IS 'Optional expiration timestamp for temporary blacklists';
COMMENT ON COLUMN exchange_blacklist.is_active IS 'Whether this blacklist entry is currently active';