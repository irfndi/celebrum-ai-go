# Aggregated Trading Signals Implementation Plan

## 1. Executive Summary

This document outlines the implementation plan for building an aggregated trading signals system that provides simplified, high-quality arbitrage and technical analysis alerts via Telegram. The system prioritizes quality over quantity by deduplicating similar opportunities and presenting actionable signals in an easy-to-understand format.

## 2. Current System Analysis

### 2.1 Market Data Collection Status
- **Active Exchanges**: 13 exchanges discovered dynamically from CCXT
- **Data Volume**: 28,908 market data records collected in the last hour
- **Coverage**: 6 exchanges actively providing data
- **Quality**: Real-time price data with proper validation and blacklisting

### 2.2 Existing Arbitrage Detection
- **Strategies**: 4 implemented strategies (cross-exchange, technical analysis, volatility, bid-ask spread)
- **Database**: Robust schema with `funding_arbitrage_opportunities` table
- **Status**: Functional but needs aggregation layer

### 2.3 Technical Analysis Capabilities
- **Implementation**: Custom TA indicators in Go
- **Recommendation**: Integrate `cinar` library for enhanced calculations
- **Current Features**: Basic price analysis, & advanced indicators

## 3. Aggregated Signal Requirements

### 3.1 Arbitrage Signal Format
**Target Output Example:**
```
🔄 ARBITRAGE ALERT: BTC/USDT
💰 Profit: 1.2% ($240 on $20k)
📈 BUY: $41,800 (Binance, Kraken, OKX, Bybit, KuCoin)
📉 SELL: $42,300 (Coinbase, Gate.io, MEXC)
⏰ Valid for: 5 minutes
🎯 Min Volume: $10,000
```

**Key Features:**
- Single alert per trading pair
- Aggregated buy/sell prices across multiple exchanges
- Clear profit calculation and risk assessment
- Minimum volume requirements
- Time-sensitive validity

### 3.2 Technical Analysis Signal Format
**Target Output Example:**
```
📊 TA SIGNAL: ETH/USDT
🎯 Signal: MA20 crossed above MA50 (Golden Cross) + RSI oversold recovery
💲 Current Price: $3,200
📈 Entry: $3,180 - $3,220
🎯 Target 1: $3,350 (4.7% profit)
🎯 Target 2: $3,500 (9.4% profit)
🛑 Stop Loss: $3,050 (4.7% risk)
📊 Risk/Reward: 1:2
🏪 Exchanges: Binance, Coinbase, Kraken
⏰ Timeframe: 4H
```

**Key Features:**
- Clear signal identification
- Multiple entry/exit levels
- Risk management calculations
- Exchange availability
- Timeframe specification

## 4. Technical Architecture

### 4.1 Signal Aggregation Engine
```
Market Data → Signal Detection → Aggregation → Deduplication → Telegram
     ↓              ↓              ↓            ↓           ↓
  Real-time     Individual      Group by     Remove       Format &
   Prices       Signals        Symbol      Duplicates     Send
```

### 4.2 Core Components

#### 4.2.1 Signal Aggregator Service
- **Purpose**: Collect and group similar signals
- **Location**: `internal/services/signal_aggregator.go`
- **Functions**:
  - `AggregateArbitrageSignals()`
  - `AggregateTechnicalSignals()`
  - `DeduplicateSignals()`

#### 4.2.2 Signal Quality Filter
- **Purpose**: Ensure only high-quality signals are sent
- **Criteria**:
  - Minimum profit threshold (arbitrage: 0.5%)
  - Minimum volume requirements
  - Signal confidence score
  - Exchange reliability rating

#### 4.2.3 Telegram Formatter
- **Purpose**: Convert aggregated signals to user-friendly messages
- **Features**:
  - Emoji-rich formatting
  - Clear action items
  - Risk warnings
  - Exchange-specific instructions

### 4.3 Database Schema Extensions

#### 4.3.1 Aggregated Signals Table
```sql
CREATE TABLE aggregated_signals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_type VARCHAR(20) NOT NULL, -- 'arbitrage' or 'technical'
    symbol VARCHAR(20) NOT NULL,
    base_currency VARCHAR(10) NOT NULL,
    quote_currency VARCHAR(10) NOT NULL,
    signal_data JSONB NOT NULL, -- Flexible signal details
    quality_score DECIMAL(5,2) NOT NULL,
    exchanges TEXT[] NOT NULL,
    profit_potential DECIMAL(10,4),
    risk_level VARCHAR(10), -- 'low', 'medium', 'high'
    valid_until TIMESTAMP WITH TIME ZONE,
    sent_to_telegram BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

#### 4.3.2 Signal Deduplication Table
```sql
CREATE TABLE signal_fingerprints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint_hash VARCHAR(64) UNIQUE NOT NULL,
    signal_type VARCHAR(20) NOT NULL,
    symbol VARCHAR(20) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
```

## 5. Implementation Phases

### Phase 1: Foundation (Week 1-2)
**Priority: HIGH**

1. **Install Technical Analysis Library**
   - Add `github.com/sdcoffey/techan` to go.mod
   - Create TA calculation service
   - Implement basic indicators (MA, RSI, MACD)

2. **Create Signal Aggregation Service**
   - Implement `SignalAggregator` struct
   - Add signal grouping logic
   - Create deduplication mechanism

3. **Database Schema Updates**
   - Create migration for aggregated signals tables
   - Add indexes for performance
   - Implement cleanup procedures

### Phase 2: Arbitrage Aggregation (Week 3)
**Priority: HIGH**

1. **Arbitrage Signal Aggregator**
   - Group arbitrage opportunities by symbol
   - Calculate aggregated buy/sell prices
   - Implement profit threshold filtering

2. **Quality Scoring System**
   - Exchange reliability scoring
   - Volume-based quality assessment
   - Historical accuracy tracking

3. **Telegram Integration**
   - Create arbitrage message formatter
   - Implement rate limiting
   - Add user subscription management

### Phase 3: Technical Analysis Aggregation (Week 4)
**Priority: HIGH**

1. **TA Signal Detection**
   - Implement moving average crossovers
   - Add RSI overbought/oversold signals
   - Create MACD divergence detection

2. **Risk Management Calculator**
   - Entry/exit level calculation
   - Stop-loss positioning
   - Risk/reward ratio analysis

3. **Multi-Exchange TA Analysis**
   - Cross-exchange signal validation
   - Exchange-specific signal strength
   - Consensus-based signal generation

### Phase 4: Advanced Features (Week 5-6)
**Priority: MEDIUM**

1. **Signal Backtesting**
   - Historical performance analysis
   - Signal accuracy tracking
   - Performance metrics dashboard

2. **User Customization**
   - Personalized signal preferences
   - Risk tolerance settings
   - Exchange preference filtering

3. **Advanced TA Indicators**
   - Bollinger Bands
   - Fibonacci retracements
   - Volume-based indicators

### Phase 5: Optimization & Monitoring (Week 7-8)
**Priority: LOW**

1. **Performance Optimization**
   - Signal processing speed improvements
   - Database query optimization
   - Caching implementation

2. **Monitoring & Alerting**
   - Signal quality monitoring
   - System health checks
   - Performance metrics tracking

## 6. Testing Strategy

### 6.1 Unit Testing Requirements
**Target Coverage: 95%+**

#### 6.1.1 Signal Aggregation Tests
```go
// Test file: internal/services/signal_aggregator_test.go
func TestAggregateArbitrageSignals(t *testing.T)
func TestAggregateTechnicalSignals(t *testing.T)
func TestDeduplicateSignals(t *testing.T)
func TestSignalQualityScoring(t *testing.T)
```

#### 6.1.2 Technical Analysis Tests
```go
// Test file: internal/services/technical_analysis_test.go
func TestMovingAverageCrossover(t *testing.T)
func TestRSISignalDetection(t *testing.T)
func TestMACDDivergence(t *testing.T)
func TestRiskManagementCalculation(t *testing.T)
```

#### 6.1.3 Telegram Integration Tests
```go
// Test file: internal/services/telegram_formatter_test.go
func TestArbitrageMessageFormatting(t *testing.T)
func TestTechnicalAnalysisMessageFormatting(t *testing.T)
func TestMessageRateLimiting(t *testing.T)
```

### 6.2 Integration Testing

#### 6.2.1 End-to-End Signal Flow
- Market data → Signal detection → Aggregation → Telegram delivery
- Test with real market data samples
- Verify signal accuracy and timing

#### 6.2.2 Database Integration
- Signal persistence and retrieval
- Deduplication effectiveness
- Performance under load

### 6.3 Performance Testing

#### 6.3.1 Load Testing
- 1000+ concurrent signal processing
- Database performance under high load
- Telegram API rate limit handling

#### 6.3.2 Accuracy Testing
- Historical signal backtesting
- Profit/loss tracking
- False positive rate measurement

## 7. Quality Assurance

### 7.1 Signal Quality Metrics
- **Accuracy Rate**: % of profitable signals
- **False Positive Rate**: % of unprofitable signals
- **Response Time**: Signal detection to delivery time
- **Coverage**: % of market opportunities captured

### 7.2 Deduplication Strategy
- **Fingerprint Generation**: Hash of symbol + signal type + key parameters
- **Time Window**: 15-minute deduplication window
- **Similarity Threshold**: 95% parameter similarity

### 7.3 Risk Management
- **Maximum Signal Frequency**: 5 signals per symbol per hour
- **Profit Threshold**: Minimum 0.5% for arbitrage, 2% for TA
- **Volume Requirements**: Minimum $10,000 liquidity

## 8. Monitoring & Maintenance

### 8.1 Key Performance Indicators
- Signal generation rate
- User engagement metrics
- Profit accuracy tracking
- System uptime and reliability

### 8.2 Maintenance Tasks
- Daily signal quality review
- Weekly performance analysis
- Monthly strategy optimization
- Quarterly system updates

## 9. Risk Considerations

### 9.1 Technical Risks
- Exchange API rate limits
- Market data latency
- Signal processing delays
- Database performance bottlenecks

### 9.2 Financial Risks
- False signal generation
- Market volatility impact
- Exchange reliability issues
- Regulatory compliance

### 9.3 Mitigation Strategies
- Comprehensive testing
- Conservative signal thresholds
- Multiple data source validation
- Clear risk disclaimers

## 10. Success Criteria

### 10.1 Technical Success
- 95%+ test coverage
- <2 second signal processing time
- 99.9% system uptime
- Zero data loss incidents

### 10.2 Business Success
- 80%+ signal accuracy rate
- <5% false positive rate
- High user engagement
- Positive user feedback

## 11. Next Steps

1. **Immediate Actions**:
   - Review and approve implementation plan
   - Set up development environment
   - Create project timeline
   - Assign development resources

2. **Week 1 Deliverables**:
   - Technical analysis library integration
   - Basic signal aggregation framework
   - Database schema updates
   - Initial test suite setup

This implementation plan provides a comprehensive roadmap for building a high-quality, aggregated trading signals system that prioritizes user experience and signal accuracy over quantity.