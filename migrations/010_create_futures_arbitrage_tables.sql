-- Migration: Create futures arbitrage tables
-- This migration creates tables to support futures arbitrage opportunities
-- focusing on funding rate differentials, APY calculations, and risk management

-- Create futures arbitrage opportunities table
CREATE TABLE IF NOT EXISTS futures_arbitrage_opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) NOT NULL,
    base_currency VARCHAR(10) NOT NULL,
    quote_currency VARCHAR(10) NOT NULL,
    
    -- Exchange information
    long_exchange VARCHAR(50) NOT NULL,
    short_exchange VARCHAR(50) NOT NULL,
    long_exchange_id INTEGER REFERENCES exchanges(id),
    short_exchange_id INTEGER REFERENCES exchanges(id),
    
    -- Funding rate data
    long_funding_rate DECIMAL(20, 8) NOT NULL,
    short_funding_rate DECIMAL(20, 8) NOT NULL,
    net_funding_rate DECIMAL(20, 8) NOT NULL,
    funding_interval INTEGER NOT NULL DEFAULT 8, -- Hours between funding payments
    
    -- Price data
    long_mark_price DECIMAL(20, 8) NOT NULL,
    short_mark_price DECIMAL(20, 8) NOT NULL,
    price_difference DECIMAL(20, 8) NOT NULL,
    price_difference_percentage DECIMAL(10, 4) NOT NULL,
    
    -- Profit calculations
    hourly_rate DECIMAL(20, 8) NOT NULL,
    daily_rate DECIMAL(20, 8) NOT NULL,
    apy DECIMAL(10, 4) NOT NULL, -- Annual Percentage Yield
    estimated_profit_8h DECIMAL(20, 8) NOT NULL,
    estimated_profit_daily DECIMAL(20, 8) NOT NULL,
    estimated_profit_weekly DECIMAL(20, 8) NOT NULL,
    estimated_profit_monthly DECIMAL(20, 8) NOT NULL,
    
    -- Risk management
    risk_score DECIMAL(5, 2) NOT NULL, -- 0-100 scale
    volatility_score DECIMAL(5, 2) NOT NULL,
    liquidity_score DECIMAL(5, 2) NOT NULL,
    recommended_position_size DECIMAL(20, 8),
    max_leverage DECIMAL(5, 2),
    recommended_leverage DECIMAL(5, 2),
    stop_loss_percentage DECIMAL(5, 2),
    
    -- Position sizing recommendations
    min_position_size DECIMAL(20, 8),
    max_position_size DECIMAL(20, 8),
    optimal_position_size DECIMAL(20, 8),
    
    -- Timing and validity
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    next_funding_time TIMESTAMP WITH TIME ZONE NOT NULL,
    time_to_next_funding INTEGER NOT NULL, -- Minutes
    is_active BOOLEAN NOT NULL DEFAULT true,
    
    -- Market conditions
    market_trend VARCHAR(20) DEFAULT 'neutral', -- 'bullish', 'bearish', 'neutral'
    volume_24h DECIMAL(20, 8),
    open_interest DECIMAL(20, 8),
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create futures arbitrage risk metrics table
CREATE TABLE IF NOT EXISTS futures_arbitrage_risk_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID NOT NULL REFERENCES futures_arbitrage_opportunities(id) ON DELETE CASCADE,
    
    -- Price risk
    price_correlation DECIMAL(5, 4) NOT NULL, -- -1 to 1
    price_volatility DECIMAL(5, 2) NOT NULL, -- 0-100
    max_drawdown DECIMAL(5, 2), -- Percentage
    
    -- Funding rate risk
    funding_rate_volatility DECIMAL(5, 2) NOT NULL,
    funding_rate_stability DECIMAL(5, 2) NOT NULL,
    
    -- Liquidity risk
    bid_ask_spread DECIMAL(10, 6) NOT NULL,
    market_depth DECIMAL(5, 2) NOT NULL,
    slippage_risk DECIMAL(5, 2) NOT NULL,
    
    -- Exchange risk
    exchange_reliability DECIMAL(5, 2) NOT NULL,
    counterparty_risk DECIMAL(5, 2) NOT NULL,
    
    -- Overall assessment
    overall_risk_score DECIMAL(5, 2) NOT NULL,
    risk_category VARCHAR(20) NOT NULL, -- 'low', 'medium', 'high', 'extreme'
    recommendation TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create futures position sizing table
CREATE TABLE IF NOT EXISTS futures_position_sizing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    opportunity_id UUID NOT NULL REFERENCES futures_arbitrage_opportunities(id) ON DELETE CASCADE,
    
    -- Kelly Criterion based sizing
    kelly_percentage DECIMAL(5, 2) NOT NULL,
    kelly_position_size DECIMAL(20, 8) NOT NULL,
    
    -- Risk-based sizing
    conservative_size DECIMAL(20, 8) NOT NULL,
    moderate_size DECIMAL(20, 8) NOT NULL,
    aggressive_size DECIMAL(20, 8) NOT NULL,
    
    -- Leverage recommendations
    min_leverage DECIMAL(5, 2) NOT NULL,
    optimal_leverage DECIMAL(5, 2) NOT NULL,
    max_safe_leverage DECIMAL(5, 2) NOT NULL,
    
    -- Risk management
    stop_loss_price DECIMAL(20, 8),
    take_profit_price DECIMAL(20, 8),
    max_loss_percentage DECIMAL(5, 2) NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create futures arbitrage strategies table
CREATE TABLE IF NOT EXISTS futures_arbitrage_strategies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    opportunity_id UUID NOT NULL REFERENCES futures_arbitrage_opportunities(id) ON DELETE CASCADE,
    
    -- Strategy performance metrics
    expected_return DECIMAL(10, 4) NOT NULL,
    sharpe_ratio DECIMAL(10, 4),
    max_drawdown_expected DECIMAL(5, 2),
    
    -- Execution details
    execution_order JSONB, -- Array of execution steps
    estimated_execution_time INTEGER, -- Seconds
    
    -- Position details
    long_position JSONB, -- Position details for long side
    short_position JSONB, -- Position details for short side
    
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create funding rate history table for better risk calculations
CREATE TABLE IF NOT EXISTS funding_rate_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_id INTEGER NOT NULL REFERENCES exchanges(id),
    symbol VARCHAR(20) NOT NULL,
    funding_rate DECIMAL(20, 8) NOT NULL,
    mark_price DECIMAL(20, 8) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_symbol ON futures_arbitrage_opportunities(symbol);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_exchanges ON futures_arbitrage_opportunities(long_exchange, short_exchange);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_detected_at ON futures_arbitrage_opportunities(detected_at);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_is_active ON futures_arbitrage_opportunities(is_active);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_apy ON futures_arbitrage_opportunities(apy);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_risk_score ON futures_arbitrage_opportunities(risk_score);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_net_funding_rate ON futures_arbitrage_opportunities(net_funding_rate);

CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_risk_metrics_opportunity_id ON futures_arbitrage_risk_metrics(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_risk_metrics_risk_category ON futures_arbitrage_risk_metrics(risk_category);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_risk_metrics_overall_risk_score ON futures_arbitrage_risk_metrics(overall_risk_score);

CREATE INDEX IF NOT EXISTS idx_futures_position_sizing_opportunity_id ON futures_position_sizing(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_futures_position_sizing_optimal_leverage ON futures_position_sizing(optimal_leverage);

CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_strategies_opportunity_id ON futures_arbitrage_strategies(opportunity_id);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_strategies_expected_return ON futures_arbitrage_strategies(expected_return);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_strategies_is_active ON futures_arbitrage_strategies(is_active);

CREATE INDEX IF NOT EXISTS idx_funding_rate_history_exchange_symbol ON funding_rate_history(exchange_id, symbol);
CREATE INDEX IF NOT EXISTS idx_funding_rate_history_timestamp ON funding_rate_history(timestamp);
CREATE INDEX IF NOT EXISTS idx_funding_rate_history_symbol_timestamp ON funding_rate_history(symbol, timestamp);

-- Create view for active futures arbitrage opportunities with risk metrics
CREATE OR REPLACE VIEW active_futures_arbitrage_opportunities AS
SELECT 
    fao.*,
    farm.overall_risk_score,
    farm.risk_category,
    farm.recommendation,
    fps.optimal_leverage,
    fps.kelly_percentage,
    fps.max_safe_leverage
FROM futures_arbitrage_opportunities fao
LEFT JOIN futures_arbitrage_risk_metrics farm ON fao.id = farm.opportunity_id
LEFT JOIN futures_position_sizing fps ON fao.id = fps.opportunity_id
WHERE fao.is_active = true
  AND fao.expires_at > NOW()
  AND fao.net_funding_rate > 0.0001 -- Minimum 0.01% funding rate differential
ORDER BY fao.apy DESC, farm.overall_risk_score ASC;

-- Create view for high-yield low-risk opportunities
CREATE OR REPLACE VIEW high_yield_low_risk_futures_arbitrage AS
SELECT 
    fao.*,
    farm.overall_risk_score,
    farm.risk_category,
    fps.optimal_leverage,
    fps.kelly_percentage
FROM futures_arbitrage_opportunities fao
JOIN futures_arbitrage_risk_metrics farm ON fao.id = farm.opportunity_id
JOIN futures_position_sizing fps ON fao.id = fps.opportunity_id
WHERE fao.is_active = true
  AND fao.expires_at > NOW()
  AND fao.apy >= 10.0 -- Minimum 10% APY
  AND farm.overall_risk_score <= 50.0 -- Maximum 50% risk score
  AND farm.risk_category IN ('low', 'medium')
ORDER BY fao.apy DESC, farm.overall_risk_score ASC;

-- Create function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to automatically update updated_at
CREATE TRIGGER update_futures_arbitrage_opportunities_updated_at
    BEFORE UPDATE ON futures_arbitrage_opportunities
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_futures_arbitrage_risk_metrics_updated_at
    BEFORE UPDATE ON futures_arbitrage_risk_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_futures_position_sizing_updated_at
    BEFORE UPDATE ON futures_position_sizing
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_futures_arbitrage_strategies_updated_at
    BEFORE UPDATE ON futures_arbitrage_strategies
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function to clean up expired opportunities
CREATE OR REPLACE FUNCTION cleanup_expired_futures_arbitrage_opportunities()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    -- Delete opportunities older than 24 hours
    DELETE FROM futures_arbitrage_opportunities 
    WHERE detected_at < NOW() - INTERVAL '24 hours'
       OR expires_at < NOW();
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Also clean up old funding rate history (keep last 7 days)
    DELETE FROM funding_rate_history 
    WHERE timestamp < NOW() - INTERVAL '7 days';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Create function to calculate opportunity quality score
CREATE OR REPLACE FUNCTION calculate_opportunity_quality_score(
    apy_value DECIMAL,
    risk_score_value DECIMAL,
    liquidity_score_value DECIMAL
) RETURNS DECIMAL AS $$
BEGIN
    -- Quality score = (APY * 0.5) + ((100 - risk_score) * 0.3) + (liquidity_score * 0.2)
    -- Normalized to 0-100 scale
    RETURN LEAST(100, 
        (apy_value * 0.5) + 
        ((100 - risk_score_value) * 0.3) + 
        (liquidity_score_value * 0.2)
    );
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE futures_arbitrage_opportunities IS 'Stores futures arbitrage opportunities based on funding rate differentials';
COMMENT ON TABLE futures_arbitrage_risk_metrics IS 'Comprehensive risk assessment for futures arbitrage opportunities';
COMMENT ON TABLE futures_position_sizing IS 'Position sizing and leverage recommendations for futures arbitrage';
COMMENT ON TABLE futures_arbitrage_strategies IS 'Complete arbitrage strategies with execution plans';
COMMENT ON TABLE funding_rate_history IS 'Historical funding rate data for risk calculations and backtesting';

COMMENT ON COLUMN futures_arbitrage_opportunities.net_funding_rate IS 'Difference between short and long funding rates (profit driver)';
COMMENT ON COLUMN futures_arbitrage_opportunities.apy IS 'Annualized percentage yield based on funding rate differential';
COMMENT ON COLUMN futures_arbitrage_opportunities.risk_score IS 'Overall risk score from 0 (lowest risk) to 100 (highest risk)';
COMMENT ON COLUMN futures_arbitrage_opportunities.recommended_leverage IS 'Recommended leverage based on risk assessment';
COMMENT ON COLUMN futures_arbitrage_opportunities.time_to_next_funding IS 'Minutes until next funding payment';

COMMENT ON VIEW active_futures_arbitrage_opportunities IS 'Active opportunities with risk metrics and position sizing';
COMMENT ON VIEW high_yield_low_risk_futures_arbitrage IS 'High-yield opportunities with acceptable risk levels';

COMMENT ON FUNCTION cleanup_expired_futures_arbitrage_opportunities() IS 'Cleans up expired opportunities and old historical data';
COMMENT ON FUNCTION calculate_opportunity_quality_score(DECIMAL, DECIMAL, DECIMAL) IS 'Calculates a composite quality score for ranking opportunities';