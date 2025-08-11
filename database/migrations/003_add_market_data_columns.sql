-- Add missing market data columns to match Go model structure
-- Created: 2025-08-10

-- Add new columns to market_data table to match MarketData struct
ALTER TABLE market_data 
ADD COLUMN IF NOT EXISTS last_price DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS bid DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS bid_volume DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS ask DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS ask_volume DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS high_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS low_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS open_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS close_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS volume_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS quote_volume_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS vwap DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS change_24h DECIMAL(20,8),
ADD COLUMN IF NOT EXISTS change_percentage_24h DECIMAL(8,4);

-- Update existing price column to be last_price for backward compatibility
UPDATE market_data SET last_price = price WHERE last_price IS NULL;

-- Create indexes for new columns for better query performance
CREATE INDEX IF NOT EXISTS idx_market_data_last_price ON market_data(last_price);
CREATE INDEX IF NOT EXISTS idx_market_data_bid ON market_data(bid);
CREATE INDEX IF NOT EXISTS idx_market_data_ask ON market_data(ask);
CREATE INDEX IF NOT EXISTS idx_market_data_high_24h ON market_data(high_24h);
CREATE INDEX IF NOT EXISTS idx_market_data_low_24h ON market_data(low_24h);
CREATE INDEX IF NOT EXISTS idx_market_data_volume_24h ON market_data(volume_24h);
CREATE INDEX IF NOT EXISTS idx_market_data_timestamp_range ON market_data(timestamp DESC) WHERE timestamp > NOW() - INTERVAL '7 days';