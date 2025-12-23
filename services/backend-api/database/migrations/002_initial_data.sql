-- Initial data for Celebrum AI Platform
-- Created: 2025-01-17

-- Insert initial exchanges
INSERT INTO exchanges (name, api_url) VALUES 
('Binance', 'https://api.binance.com'),
('Bybit', 'https://api.bybit.com'),
('OKX', 'https://www.okx.com/api'),
('Coinbase', 'https://api.exchange.coinbase.com'),
('Kraken', 'https://api.kraken.com');

-- Insert CCXT exchange configurations
INSERT INTO ccxt_exchanges (exchange_id, ccxt_id, rate_limit, has_futures, websocket_enabled) VALUES 
(1, 'binance', 1200, true, true),
(2, 'bybit', 600, true, true),
(3, 'okx', 600, true, true),
(4, 'coinbasepro', 300, false, true),
(5, 'kraken', 300, true, false);

-- Insert initial trading pairs
INSERT INTO trading_pairs (symbol, base_currency, quote_currency, is_futures) VALUES 
('BTCUSDT', 'BTC', 'USDT', false),
('ETHUSDT', 'ETH', 'USDT', false),
('ADAUSDT', 'ADA', 'USDT', false),
('SOLUSDT', 'SOL', 'USDT', false),
('DOTUSDT', 'DOT', 'USDT', false),
('LINKUSDT', 'LINK', 'USDT', false),
('AVAXUSDT', 'AVAX', 'USDT', false),
('MATICUSDT', 'MATIC', 'USDT', false),
('ATOMUSDT', 'ATOM', 'USDT', false),
('NEARUSDT', 'NEAR', 'USDT', false),
-- Futures pairs
('BTCUSDT-PERP', 'BTC', 'USDT', true),
('ETHUSDT-PERP', 'ETH', 'USDT', true),
('SOLUSDT-PERP', 'SOL', 'USDT', true);