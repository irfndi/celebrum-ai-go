CREATE TABLE IF NOT EXISTS futures_arbitrage_opportunities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(50) NOT NULL,
    base_currency VARCHAR(20) NOT NULL,
    quote_currency VARCHAR(20) NOT NULL,
    long_exchange VARCHAR(50) NOT NULL,
    short_exchange VARCHAR(50) NOT NULL,
    long_exchange_id INTEGER,
    short_exchange_id INTEGER,
    long_funding_rate DECIMAL(10, 8) NOT NULL,
    short_funding_rate DECIMAL(10, 8) NOT NULL,
    net_funding_rate DECIMAL(10, 8) NOT NULL,
    funding_interval INTEGER NOT NULL DEFAULT 8,
    long_mark_price DECIMAL(20, 8) NOT NULL,
    short_mark_price DECIMAL(20, 8) NOT NULL,
    price_difference DECIMAL(20, 8) NOT NULL,
    price_difference_percentage DECIMAL(8, 4) NOT NULL,
    hourly_rate DECIMAL(8, 4) NOT NULL,
    daily_rate DECIMAL(8, 4) NOT NULL,
    apy DECIMAL(8, 4) NOT NULL,
    estimated_profit_8h DECIMAL(8, 4) NOT NULL,
    estimated_profit_daily DECIMAL(8, 4) NOT NULL,
    estimated_profit_weekly DECIMAL(8, 4) NOT NULL,
    estimated_profit_monthly DECIMAL(8, 4) NOT NULL,
    risk_score DECIMAL(5, 2) NOT NULL DEFAULT 0,
    volatility_score DECIMAL(5, 2) NOT NULL DEFAULT 0,
    liquidity_score DECIMAL(5, 2) NOT NULL DEFAULT 0,
    recommended_position_size DECIMAL(20, 8),
    max_leverage DECIMAL(5, 2) NOT NULL DEFAULT 1.0,
    recommended_leverage DECIMAL(5, 2) DEFAULT 1.0,
    stop_loss_percentage DECIMAL(5, 2),
    min_position_size DECIMAL(20, 8),
    max_position_size DECIMAL(20, 8),
    optimal_position_size DECIMAL(20, 8),
    detected_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW() + INTERVAL '1 hour',
    next_funding_time TIMESTAMP WITH TIME ZONE,
    time_to_next_funding INTEGER,
    is_active BOOLEAN NOT NULL DEFAULT true
);

-- Create unique constraint to prevent duplicates
CREATE UNIQUE INDEX IF NOT EXISTS idx_futures_arbitrage_unique 
ON futures_arbitrage_opportunities(symbol, long_exchange, short_exchange);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_symbol ON futures_arbitrage_opportunities(symbol);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_apy ON futures_arbitrage_opportunities(apy DESC);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_active ON futures_arbitrage_opportunities(is_active, expires_at);
CREATE INDEX IF NOT EXISTS idx_futures_arbitrage_opportunities_detected_at ON futures_arbitrage_opportunities(detected_at DESC);