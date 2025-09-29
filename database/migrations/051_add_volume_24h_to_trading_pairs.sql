-- Add volume_24h column to trading_pairs table
-- This column is needed by the signal processor for ordering trading pairs by volume

-- Add the column with a default value of 0
ALTER TABLE trading_pairs ADD COLUMN IF NOT EXISTS volume_24h DECIMAL(20, 8) DEFAULT 0.0;

-- Add an index for better performance on volume-based queries
CREATE INDEX IF NOT EXISTS idx_trading_pairs_volume_24h ON trading_pairs(volume_24h);

-- Add comment for documentation
COMMENT ON COLUMN trading_pairs.volume_24h IS '24-hour trading volume in quote currency';