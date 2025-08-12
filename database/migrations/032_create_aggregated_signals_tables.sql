-- Migration: Create aggregated signals tables
-- Description: Creates tables for storing aggregated trading signals and signal fingerprints for deduplication
-- Date: 2025-01-21

-- Create aggregated_signals table
CREATE TABLE IF NOT EXISTS aggregated_signals (
    id VARCHAR(36) PRIMARY KEY,
    signal_type VARCHAR(20) NOT NULL CHECK (signal_type IN ('arbitrage', 'technical')),
    symbol VARCHAR(50) NOT NULL,
    action VARCHAR(10) NOT NULL CHECK (action IN ('buy', 'sell', 'hold')),
    strength VARCHAR(10) NOT NULL CHECK (strength IN ('weak', 'medium', 'strong')),
    confidence DECIMAL(5,4) NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    profit_potential DECIMAL(10,4) NOT NULL,
    risk_level DECIMAL(5,4) NOT NULL CHECK (risk_level >= 0 AND risk_level <= 1),
    exchanges TEXT[] NOT NULL,
    indicators TEXT[] NOT NULL,
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create signal_fingerprints table for deduplication
CREATE TABLE IF NOT EXISTS signal_fingerprints (
    id VARCHAR(36) PRIMARY KEY,
    hash VARCHAR(64) NOT NULL,
    signal_id VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_symbol ON aggregated_signals(symbol);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_signal_type ON aggregated_signals(signal_type);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_action ON aggregated_signals(action);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_strength ON aggregated_signals(strength);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_created_at ON aggregated_signals(created_at);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_expires_at ON aggregated_signals(expires_at);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_confidence ON aggregated_signals(confidence);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_profit_potential ON aggregated_signals(profit_potential);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_symbol_type ON aggregated_signals(symbol, signal_type);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_symbol_action ON aggregated_signals(symbol, action);
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_active ON aggregated_signals(expires_at) WHERE expires_at > NOW();

-- Indexes for signal_fingerprints
CREATE UNIQUE INDEX IF NOT EXISTS idx_signal_fingerprints_hash ON signal_fingerprints(hash);
CREATE INDEX IF NOT EXISTS idx_signal_fingerprints_signal_id ON signal_fingerprints(signal_id);
CREATE INDEX IF NOT EXISTS idx_signal_fingerprints_created_at ON signal_fingerprints(created_at);

-- Foreign key constraint (optional, based on project guidelines)
-- ALTER TABLE signal_fingerprints ADD CONSTRAINT fk_signal_fingerprints_signal_id 
--     FOREIGN KEY (signal_id) REFERENCES aggregated_signals(id) ON DELETE CASCADE;

-- Create function to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_aggregated_signals_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic timestamp updates
CREATE TRIGGER trigger_update_aggregated_signals_updated_at
    BEFORE UPDATE ON aggregated_signals
    FOR EACH ROW
    EXECUTE FUNCTION update_aggregated_signals_updated_at();

-- Create function for cleaning up expired signals
CREATE OR REPLACE FUNCTION cleanup_expired_signals()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete expired signals
    DELETE FROM aggregated_signals WHERE expires_at < NOW();
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Clean up orphaned fingerprints (signals that no longer exist)
    DELETE FROM signal_fingerprints 
    WHERE signal_id NOT IN (SELECT id FROM aggregated_signals);
    
    -- Clean up old fingerprints (older than 24 hours)
    DELETE FROM signal_fingerprints 
    WHERE created_at < NOW() - INTERVAL '24 hours';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function for signal quality scoring
CREATE OR REPLACE FUNCTION calculate_signal_quality_score(
    p_confidence DECIMAL,
    p_profit_potential DECIMAL,
    p_risk_level DECIMAL,
    p_signal_type VARCHAR,
    p_strength VARCHAR
)
RETURNS DECIMAL AS $$
DECLARE
    base_score DECIMAL := 0;
    type_multiplier DECIMAL := 1;
    strength_multiplier DECIMAL := 1;
BEGIN
    -- Base score from confidence (0-40 points)
    base_score := p_confidence * 40;
    
    -- Add profit potential score (0-30 points, capped at 10% profit)
    base_score := base_score + LEAST(p_profit_potential * 3, 30);
    
    -- Subtract risk penalty (0-20 points)
    base_score := base_score - (p_risk_level * 20);
    
    -- Signal type multiplier
    CASE p_signal_type
        WHEN 'arbitrage' THEN type_multiplier := 1.2;  -- Arbitrage is more reliable
        WHEN 'technical' THEN type_multiplier := 1.0;
        ELSE type_multiplier := 0.8;
    END CASE;
    
    -- Strength multiplier
    CASE p_strength
        WHEN 'strong' THEN strength_multiplier := 1.3;
        WHEN 'medium' THEN strength_multiplier := 1.0;
        WHEN 'weak' THEN strength_multiplier := 0.7;
        ELSE strength_multiplier := 0.5;
    END CASE;
    
    -- Apply multipliers and ensure score is between 0-100
    base_score := base_score * type_multiplier * strength_multiplier;
    
    RETURN GREATEST(0, LEAST(100, base_score));
END;
$$ LANGUAGE plpgsql;

-- Add computed column for quality score (PostgreSQL 12+)
-- ALTER TABLE aggregated_signals ADD COLUMN quality_score DECIMAL(5,2) 
--     GENERATED ALWAYS AS (
--         calculate_signal_quality_score(confidence, profit_potential, risk_level, signal_type, strength)
--     ) STORED;

-- Create view for active high-quality signals
CREATE OR REPLACE VIEW active_quality_signals AS
SELECT 
    *,
    calculate_signal_quality_score(confidence, profit_potential, risk_level, signal_type, strength) as quality_score
FROM aggregated_signals 
WHERE expires_at > NOW()
    AND confidence >= 0.6
    AND calculate_signal_quality_score(confidence, profit_potential, risk_level, signal_type, strength) >= 60
ORDER BY 
    calculate_signal_quality_score(confidence, profit_potential, risk_level, signal_type, strength) DESC,
    created_at DESC;

-- Create view for signal statistics
CREATE OR REPLACE VIEW signal_statistics AS
SELECT 
    signal_type,
    symbol,
    action,
    strength,
    COUNT(*) as signal_count,
    AVG(confidence) as avg_confidence,
    AVG(profit_potential) as avg_profit_potential,
    AVG(risk_level) as avg_risk_level,
    AVG(calculate_signal_quality_score(confidence, profit_potential, risk_level, signal_type, strength)) as avg_quality_score,
    MIN(created_at) as first_signal,
    MAX(created_at) as latest_signal
FROM aggregated_signals
WHERE created_at >= NOW() - INTERVAL '24 hours'
GROUP BY signal_type, symbol, action, strength
ORDER BY avg_quality_score DESC;

-- Grant permissions to application roles
GRANT SELECT, INSERT, UPDATE, DELETE ON aggregated_signals TO authenticated;
GRANT SELECT, INSERT, UPDATE, DELETE ON signal_fingerprints TO authenticated;
GRANT SELECT ON active_quality_signals TO authenticated;
GRANT SELECT ON signal_statistics TO authenticated;
GRANT EXECUTE ON FUNCTION cleanup_expired_signals() TO authenticated;
GRANT EXECUTE ON FUNCTION calculate_signal_quality_score(DECIMAL, DECIMAL, DECIMAL, VARCHAR, VARCHAR) TO authenticated;

-- Grant basic read access to anonymous users for public signal data
GRANT SELECT ON active_quality_signals TO anon;
GRANT SELECT ON signal_statistics TO anon;

-- Create additional performance indexes for complex queries
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_quality_filter ON aggregated_signals(signal_type, confidence, expires_at) 
    WHERE confidence >= 0.6 AND expires_at > NOW();
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_profit_risk ON aggregated_signals(profit_potential DESC, risk_level ASC) 
    WHERE expires_at > NOW();
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_metadata_gin ON aggregated_signals USING GIN(metadata) 
    WHERE metadata IS NOT NULL;

-- Create index for exchange-specific queries
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_exchanges_gin ON aggregated_signals USING GIN(exchanges);

-- Create partial index for recent signals
CREATE INDEX IF NOT EXISTS idx_aggregated_signals_recent ON aggregated_signals(created_at DESC, quality_score DESC) 
    WHERE created_at >= NOW() - INTERVAL '1 hour';

-- Create function for batch signal cleanup with logging
CREATE OR REPLACE FUNCTION batch_cleanup_signals(batch_size INTEGER DEFAULT 1000)
RETURNS TABLE(deleted_signals INTEGER, deleted_fingerprints INTEGER, execution_time INTERVAL) AS $$
DECLARE
    start_time TIMESTAMP := NOW();
    signals_deleted INTEGER := 0;
    fingerprints_deleted INTEGER := 0;
    total_signals_deleted INTEGER := 0;
    total_fingerprints_deleted INTEGER := 0;
BEGIN
    -- Process in batches to avoid long locks
    LOOP
        -- Delete expired signals in batches
        DELETE FROM aggregated_signals 
        WHERE id IN (
            SELECT id FROM aggregated_signals 
            WHERE expires_at < NOW() 
            LIMIT batch_size
        );
        GET DIAGNOSTICS signals_deleted = ROW_COUNT;
        total_signals_deleted := total_signals_deleted + signals_deleted;
        
        EXIT WHEN signals_deleted = 0;
        
        -- Small delay to prevent overwhelming the database
        PERFORM pg_sleep(0.1);
    END LOOP;
    
    -- Clean up orphaned fingerprints in batches
    LOOP
        DELETE FROM signal_fingerprints 
        WHERE id IN (
            SELECT sf.id FROM signal_fingerprints sf
            LEFT JOIN aggregated_signals ag ON sf.signal_id = ag.id
            WHERE ag.id IS NULL
            LIMIT batch_size
        );
        GET DIAGNOSTICS fingerprints_deleted = ROW_COUNT;
        total_fingerprints_deleted := total_fingerprints_deleted + fingerprints_deleted;
        
        EXIT WHEN fingerprints_deleted = 0;
        
        PERFORM pg_sleep(0.1);
    END LOOP;
    
    -- Clean up old fingerprints (older than 24 hours)
    LOOP
        DELETE FROM signal_fingerprints 
        WHERE id IN (
            SELECT id FROM signal_fingerprints 
            WHERE created_at < NOW() - INTERVAL '24 hours'
            LIMIT batch_size
        );
        GET DIAGNOSTICS fingerprints_deleted = ROW_COUNT;
        total_fingerprints_deleted := total_fingerprints_deleted + fingerprints_deleted;
        
        EXIT WHEN fingerprints_deleted = 0;
        
        PERFORM pg_sleep(0.1);
    END LOOP;
    
    RETURN QUERY SELECT total_signals_deleted, total_fingerprints_deleted, NOW() - start_time;
END;
$$ LANGUAGE plpgsql;

-- Create function for signal analytics and monitoring
CREATE OR REPLACE FUNCTION get_signal_analytics(time_window INTERVAL DEFAULT '24 hours')
RETURNS TABLE(
    total_signals BIGINT,
    active_signals BIGINT,
    expired_signals BIGINT,
    avg_quality_score DECIMAL,
    top_symbols TEXT[],
    signal_type_distribution JSONB,
    strength_distribution JSONB,
    hourly_signal_count JSONB
) AS $$
BEGIN
    RETURN QUERY
    WITH signal_stats AS (
        SELECT 
            COUNT(*) as total,
            COUNT(*) FILTER (WHERE expires_at > NOW()) as active,
            COUNT(*) FILTER (WHERE expires_at <= NOW()) as expired,
            AVG(calculate_signal_quality_score(confidence, profit_potential, risk_level, signal_type, strength)) as avg_quality
        FROM aggregated_signals 
        WHERE created_at >= NOW() - time_window
    ),
    top_symbols_cte AS (
        SELECT ARRAY_AGG(symbol ORDER BY signal_count DESC) as symbols
        FROM (
            SELECT symbol, COUNT(*) as signal_count
            FROM aggregated_signals 
            WHERE created_at >= NOW() - time_window
            GROUP BY symbol
            ORDER BY signal_count DESC
            LIMIT 10
        ) t
    ),
    type_dist AS (
        SELECT jsonb_object_agg(signal_type, count) as distribution
        FROM (
            SELECT signal_type, COUNT(*) as count
            FROM aggregated_signals 
            WHERE created_at >= NOW() - time_window
            GROUP BY signal_type
        ) t
    ),
    strength_dist AS (
        SELECT jsonb_object_agg(strength, count) as distribution
        FROM (
            SELECT strength, COUNT(*) as count
            FROM aggregated_signals 
            WHERE created_at >= NOW() - time_window
            GROUP BY strength
        ) t
    ),
    hourly_counts AS (
        SELECT jsonb_object_agg(hour_bucket, signal_count) as hourly_data
        FROM (
            SELECT 
                date_trunc('hour', created_at) as hour_bucket,
                COUNT(*) as signal_count
            FROM aggregated_signals 
            WHERE created_at >= NOW() - time_window
            GROUP BY date_trunc('hour', created_at)
            ORDER BY hour_bucket
        ) t
    )
    SELECT 
        s.total,
        s.active,
        s.expired,
        s.avg_quality,
        ts.symbols,
        td.distribution,
        sd.distribution,
        hc.hourly_data
    FROM signal_stats s
    CROSS JOIN top_symbols_cte ts
    CROSS JOIN type_dist td
    CROSS JOIN strength_dist sd
    CROSS JOIN hourly_counts hc;
END;
$$ LANGUAGE plpgsql;

-- Create function for signal performance tracking
CREATE OR REPLACE FUNCTION track_signal_performance(
    p_signal_id VARCHAR(36),
    p_actual_profit DECIMAL,
    p_execution_time TIMESTAMP DEFAULT NOW()
)
RETURNS BOOLEAN AS $$
DECLARE
    signal_exists BOOLEAN;
BEGIN
    -- Check if signal exists
    SELECT EXISTS(SELECT 1 FROM aggregated_signals WHERE id = p_signal_id) INTO signal_exists;
    
    IF NOT signal_exists THEN
        RETURN FALSE;
    END IF;
    
    -- Update metadata with performance data
    UPDATE aggregated_signals 
    SET metadata = COALESCE(metadata, '{}'::jsonb) || 
        jsonb_build_object(
            'actual_profit', p_actual_profit,
            'execution_time', p_execution_time,
            'performance_tracked', true,
            'tracked_at', NOW()
        )
    WHERE id = p_signal_id;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions for new functions
GRANT EXECUTE ON FUNCTION batch_cleanup_signals(INTEGER) TO authenticated;
GRANT EXECUTE ON FUNCTION get_signal_analytics(INTERVAL) TO authenticated;
GRANT EXECUTE ON FUNCTION track_signal_performance(VARCHAR, DECIMAL, TIMESTAMP) TO authenticated;

-- Add comments for documentation
COMMENT ON TABLE aggregated_signals IS 'Stores aggregated trading signals from various sources (arbitrage, technical analysis)';
COMMENT ON TABLE signal_fingerprints IS 'Stores signal fingerprints for deduplication purposes';
COMMENT ON COLUMN aggregated_signals.signal_type IS 'Type of signal: arbitrage or technical';
COMMENT ON COLUMN aggregated_signals.confidence IS 'Signal confidence level (0.0 to 1.0)';
COMMENT ON COLUMN aggregated_signals.profit_potential IS 'Expected profit percentage';
COMMENT ON COLUMN aggregated_signals.risk_level IS 'Risk level (0.0 to 1.0)';
COMMENT ON COLUMN aggregated_signals.exchanges IS 'Array of exchange names/IDs involved';
COMMENT ON COLUMN aggregated_signals.indicators IS 'Array of technical indicators used';
COMMENT ON COLUMN aggregated_signals.metadata IS 'Additional signal-specific data in JSON format';
COMMENT ON FUNCTION cleanup_expired_signals() IS 'Removes expired signals and orphaned fingerprints';
COMMENT ON FUNCTION calculate_signal_quality_score(DECIMAL, DECIMAL, DECIMAL, VARCHAR, VARCHAR) IS 'Calculates quality score (0-100) based on signal parameters';