import type { Exchange, Ticker, OrderBook, OHLCV } from 'ccxt';

// API Response Types
export interface HealthResponse {
  status: 'healthy' | 'unhealthy';
  timestamp: string;
  service: string;
  version: string;
}

export interface ExchangeInfo {
  id: string;
  name: string;
  countries: string[];
  urls: Record<string, any>;
}

export interface ExchangesResponse {
  exchanges: ExchangeInfo[];
}

export interface TickerResponse {
  exchange: string;
  symbol: string;
  ticker: Ticker;
  timestamp: string;
}

export interface OrderBookResponse {
  exchange: string;
  symbol: string;
  orderbook: OrderBook;
  timestamp: string;
}

export interface OHLCVResponse {
  exchange: string;
  symbol: string;
  timeframe: string;
  ohlcv: OHLCV[];
  timestamp: string;
}

export interface MarketsResponse {
  exchange: string;
  symbols: string[];
  count: number;
  timestamp: string;
}

export interface MultiTickerRequest {
  symbols: string[];
  exchanges?: string[];
}

export interface MultiTickerResponse {
  results: Record<string, Record<string, Ticker | { error: string }>>;
  timestamp: string;
}

export interface ErrorResponse {
  error: string;
  message?: string;
  timestamp: string;
}

// Funding Rate Types
export interface FundingRate {
  symbol: string;
  fundingRate: number;
  fundingTimestamp: number;
  nextFundingTime: number;
  markPrice: number;
  indexPrice: number;
  timestamp: number;
}

export interface FundingRateResponse {
  exchange: string;
  fundingRates: FundingRate[];
  count: number;
  timestamp: string;
}

export interface FundingRateQuery {
  symbols?: string[];
}

// Exchange Management
export interface ExchangeManager {
  [key: string]: Exchange;
}

// Query Parameters
export interface OHLCVQuery {
  timeframe?: string;
  limit?: string;
}

export interface OrderBookQuery {
  limit?: string;
}

// Environment Variables
export interface EnvConfig {
  PORT: string;
  NODE_ENV: string;
}