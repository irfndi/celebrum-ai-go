-- Migration: Improve symbol parsing function
-- This migration replaces the fragile get_or_create_trading_pair_id function
-- with a more robust version that handles various symbol formats

-- Drop the existing function first
DROP FUNCTION IF EXISTS get_or_create_trading_pair_id(VARCHAR);

-- Create improved version of get_or_create_trading_pair_id function
CREATE OR REPLACE FUNCTION get_or_create_trading_pair_id(symbol_input VARCHAR(50))
RETURNS INTEGER AS $$
DECLARE
    trading_pair_id INTEGER;
    base_currency VARCHAR(20);
    quote_currency VARCHAR(20);
    delimiter_pos INTEGER;
BEGIN
    -- First, try to find existing trading pair
    SELECT id INTO trading_pair_id 
    FROM trading_pairs 
    WHERE symbol = symbol_input;
    
    IF trading_pair_id IS NOT NULL THEN
        RETURN trading_pair_id;
    END IF;
    
    -- Parse symbol into base and quote currencies
    -- Handle symbols with common separators first
    IF position('/' in symbol_input) > 0 THEN
        delimiter_pos := position('/' in symbol_input);
        base_currency := substring(symbol_input from 1 for delimiter_pos - 1);
        quote_currency := split_part(substring(symbol_input from delimiter_pos + 1), ':', 1); -- Remove settlement currency
    ELSIF position('_' in symbol_input) > 0 THEN
        delimiter_pos := position('_' in symbol_input);
        base_currency := substring(symbol_input from 1 for delimiter_pos - 1);
        quote_currency := substring(symbol_input from delimiter_pos + 1);
    ELSIF position('-' in symbol_input) > 0 AND symbol_input NOT LIKE '%-%-%' THEN -- Avoid options contracts
        delimiter_pos := position('-' in symbol_input);
        base_currency := substring(symbol_input from 1 for delimiter_pos - 1);
        quote_currency := substring(symbol_input from delimiter_pos + 1);
    ELSE
        -- Handle symbols without separators using improved logic
        -- Try common quote currencies in order of specificity (longer first)
        CASE 
            WHEN symbol_input LIKE '%USDT' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'USDT';
            WHEN symbol_input LIKE '%USDC' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'USDC';
            WHEN symbol_input LIKE '%BUSD' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'BUSD';
            WHEN symbol_input LIKE '%TUSD' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'TUSD';
            WHEN symbol_input LIKE '%FDUSD' AND length(symbol_input) > 5 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 5);
                quote_currency := 'FDUSD';
            WHEN symbol_input LIKE '%DOGE' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'DOGE';
            WHEN symbol_input LIKE '%SHIB' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'SHIB';
            WHEN symbol_input LIKE '%MATIC' AND length(symbol_input) > 5 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 5);
                quote_currency := 'MATIC';
            WHEN symbol_input LIKE '%AVAX' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'AVAX';
            WHEN symbol_input LIKE '%LINK' AND length(symbol_input) > 4 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 4);
                quote_currency := 'LINK';
            WHEN symbol_input LIKE '%BTC' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'BTC';
            WHEN symbol_input LIKE '%ETH' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'ETH';
            WHEN symbol_input LIKE '%BNB' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'BNB';
            WHEN symbol_input LIKE '%ADA' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'ADA';
            WHEN symbol_input LIKE '%DOT' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'DOT';
            WHEN symbol_input LIKE '%SOL' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'SOL';
            WHEN symbol_input LIKE '%USD' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'USD';
            WHEN symbol_input LIKE '%EUR' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'EUR';
            WHEN symbol_input LIKE '%GBP' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'GBP';
            WHEN symbol_input LIKE '%JPY' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'JPY';
            WHEN symbol_input LIKE '%AUD' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'AUD';
            WHEN symbol_input LIKE '%CAD' AND length(symbol_input) > 3 THEN
                base_currency := substring(symbol_input from 1 for length(symbol_input) - 3);
                quote_currency := 'CAD';
            ELSE
                -- If no pattern matches, return NULL to indicate parsing failure
                -- This prevents data corruption from incorrect parsing
                RETURN NULL;
        END CASE;
    END IF;
    
    -- Validate parsed currencies
    IF base_currency IS NULL OR quote_currency IS NULL OR 
       length(base_currency) = 0 OR length(quote_currency) = 0 OR
       length(base_currency) > 20 OR length(quote_currency) > 20 THEN
        RETURN NULL;
    END IF;
    
    -- Create new trading pair
    INSERT INTO trading_pairs (symbol, base_currency, quote_currency, created_at, updated_at)
    VALUES (symbol_input, base_currency, quote_currency, NOW(), NOW())
    RETURNING id INTO trading_pair_id;
    
    RETURN trading_pair_id;
EXCEPTION
    WHEN OTHERS THEN
        -- Log error and return NULL instead of failing
        RAISE WARNING 'Failed to create trading pair for symbol %: %', symbol_input, SQLERRM;
        RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Add comment explaining the function
COMMENT ON FUNCTION get_or_create_trading_pair_id(VARCHAR) IS 
'Improved function to parse trading symbols and create trading pairs with robust error handling and validation';