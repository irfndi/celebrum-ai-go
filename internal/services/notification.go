package services

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	userModels "github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

type NotificationService struct {
	db    *database.PostgresDB
	redis *database.RedisClient
	bot   *bot.Bot
}

type ArbitrageOpportunity struct {
	Symbol          string    `json:"symbol"`
	BuyExchange     string    `json:"buy_exchange"`
	SellExchange    string    `json:"sell_exchange"`
	BuyPrice        float64   `json:"buy_price"`
	SellPrice       float64   `json:"sell_price"`
	ProfitPercent   float64   `json:"profit_percent"`
	ProfitAmount    float64   `json:"profit_amount"`
	Volume          float64   `json:"volume"`
	Timestamp       time.Time `json:"timestamp"`
	OpportunityType string    `json:"opportunity_type"` // "arbitrage", "technical", "ai_generated"
}

// TechnicalSignalNotification represents a technical analysis signal for notifications
type TechnicalSignalNotification struct {
	Symbol       string    `json:"symbol"`
	SignalType   string    `json:"signal_type"`
	Action       string    `json:"action"`
	SignalText   string    `json:"signal_text"`
	CurrentPrice float64   `json:"current_price"`
	EntryRange   string    `json:"entry_range"`
	Targets      []Target  `json:"targets"`
	StopLoss     StopLoss  `json:"stop_loss"`
	RiskReward   string    `json:"risk_reward"`
	Exchanges    []string  `json:"exchanges"`
	Timeframe    string    `json:"timeframe"`
	Confidence   float64   `json:"confidence"`
	Timestamp    time.Time `json:"timestamp"`
}

type Target struct {
	Price  float64 `json:"price"`
	Profit float64 `json:"profit"`
}

type StopLoss struct {
	Price float64 `json:"price"`
	Risk  float64 `json:"risk"`
}

func NewNotificationService(db *database.PostgresDB, redis *database.RedisClient, telegramBotToken string) *NotificationService {
	// Initialize Telegram bot if token is provided
	var telegramBot *bot.Bot
	if telegramBotToken != "" {
		telegramBot, _ = bot.New(telegramBotToken)
	}

	return &NotificationService{
		db:    db,
		redis: redis,
		bot:   telegramBot,
	}
}

// PublishOpportunityUpdate publishes arbitrage opportunity updates via Redis pub/sub
func (ns *NotificationService) PublishOpportunityUpdate(ctx context.Context, opportunities []ArbitrageOpportunity) {
	if ns.redis == nil || len(opportunities) == 0 {
		return
	}

	channel := "arbitrage_opportunities"

	// Note: Redis pub/sub would require additional Redis client methods
	// For now, we'll use the cache mechanism as the primary distribution method
	log.Printf("Would publish %d opportunities to Redis channel '%s'", len(opportunities), channel)
}

// GetCacheStats returns statistics about Redis cache usage
func (ns *NotificationService) GetCacheStats(ctx context.Context) map[string]interface{} {
	stats := make(map[string]interface{})

	if ns.redis == nil {
		stats["redis_available"] = false
		return stats
	}

	stats["redis_available"] = true

	// Check if eligible users are cached
	usersCacheKey := "eligible_users:arbitrage"
	if exists, err := ns.redis.Exists(ctx, usersCacheKey); err == nil {
		stats["users_cached"] = exists
	}

	// Check if opportunities are cached
	oppsCacheKey := "arbitrage_opportunities:latest"
	if exists, err := ns.redis.Exists(ctx, oppsCacheKey); err == nil {
		stats["opportunities_cached"] = exists
	}

	return stats
}

// NotifyArbitrageOpportunities sends notifications about arbitrage opportunities to eligible users
func (ns *NotificationService) NotifyArbitrageOpportunities(ctx context.Context, opportunities []ArbitrageOpportunity) error {
	// Cache opportunities for faster subsequent access
	ns.cacheArbitrageOpportunities(ctx, opportunities)

	// Publish opportunities for real-time updates
	ns.PublishOpportunityUpdate(ctx, opportunities)

	// Get eligible users (those with Telegram chat IDs and arbitrage alerts enabled)
	users, err := ns.getEligibleUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(users) == 0 {
		log.Printf("No eligible users found for arbitrage notifications")
		return nil
	}

	// Group opportunities by type
	arbitrageOpps := make([]ArbitrageOpportunity, 0)
	technicalOpps := make([]ArbitrageOpportunity, 0)

	for _, opp := range opportunities {
		// Categorize opportunity based on exchanges
		if opp.BuyExchange != opp.SellExchange {
			opp.OpportunityType = "arbitrage"
			arbitrageOpps = append(arbitrageOpps, opp)
		} else {
			opp.OpportunityType = "technical"
			technicalOpps = append(technicalOpps, opp)
		}
	}

	// Send notifications to each user
	for _, user := range users {
		// Send true arbitrage opportunities
		if len(arbitrageOpps) > 0 {
			if err := ns.sendArbitrageAlert(ctx, user, arbitrageOpps); err != nil {
				log.Printf("Failed to send arbitrage alert to user %s: %v", user.ID, err)
			} else {
				log.Printf("Sent arbitrage alert to user %s", user.ID)
			}
		}

		// Send technical analysis opportunities (if any)
		if len(technicalOpps) > 0 {
			if err := ns.sendArbitrageAlert(ctx, user, technicalOpps); err != nil {
				log.Printf("Failed to send technical alert to user %s: %v", user.ID, err)
			} else {
				log.Printf("Sent technical alert to user %s", user.ID)
			}
		}
	}

	log.Printf("Sent notifications to %d users: %d arbitrage, %d technical opportunities",
		len(users), len(arbitrageOpps), len(technicalOpps))
	return nil
}

// cacheArbitrageOpportunities stores arbitrage opportunities in Redis with 30-second TTL
func (ns *NotificationService) cacheArbitrageOpportunities(ctx context.Context, opportunities []ArbitrageOpportunity) {
	if ns.redis == nil || len(opportunities) == 0 {
		return
	}

	cacheKey := "arbitrage_opportunities:latest"
	oppsJSON, err := json.Marshal(opportunities)
	if err != nil {
		log.Printf("Failed to marshal opportunities for caching: %v", err)
		return
	}

	if err := ns.redis.Set(ctx, cacheKey, string(oppsJSON), 30*time.Second); err != nil {
		log.Printf("Failed to cache arbitrage opportunities: %v", err)
	} else {
		log.Printf("Cached %d arbitrage opportunities in Redis for 30 seconds", len(opportunities))
	}
}

// CacheMarketData stores market data in Redis with 10-second TTL for API performance
func (ns *NotificationService) CacheMarketData(ctx context.Context, exchange string, data interface{}) {
	if ns.redis == nil {
		return
	}

	cacheKey := fmt.Sprintf("market_data:%s", exchange)
	dataJSON, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal market data for caching: %v", err)
		return
	}

	if err := ns.redis.Set(ctx, cacheKey, string(dataJSON), 10*time.Second); err != nil {
		log.Printf("Failed to cache market data for %s: %v", exchange, err)
	} else {
		log.Printf("Cached market data for %s in Redis for 10 seconds", exchange)
	}
}

// GetCachedMarketData retrieves cached market data from Redis
func (ns *NotificationService) GetCachedMarketData(ctx context.Context, exchange string, result interface{}) error {
	if ns.redis == nil {
		return fmt.Errorf("redis not available")
	}

	cacheKey := fmt.Sprintf("market_data:%s", exchange)
	cachedData, err := ns.redis.Get(ctx, cacheKey)
	if err != nil || cachedData == "" {
		return fmt.Errorf("no cached market data found for %s", exchange)
	}

	if err := json.Unmarshal([]byte(cachedData), result); err != nil {
		return fmt.Errorf("failed to unmarshal cached market data: %w", err)
	}

	log.Printf("Retrieved cached market data for %s from Redis", exchange)
	return nil
}

// InvalidateUserCache invalidates the eligible users cache when user settings change
func (ns *NotificationService) InvalidateUserCache(ctx context.Context) {
	if ns.redis == nil {
		return
	}

	cacheKey := "eligible_users:arbitrage"
	if err := ns.redis.Delete(ctx, cacheKey); err != nil {
		log.Printf("Failed to invalidate user cache: %v", err)
	} else {
		log.Printf("Invalidated eligible users cache")
	}
}

// InvalidateOpportunityCache invalidates the arbitrage opportunities cache
func (ns *NotificationService) InvalidateOpportunityCache(ctx context.Context) {
	if ns.redis == nil {
		return
	}

	cacheKey := "arbitrage_opportunities:latest"
	if err := ns.redis.Delete(ctx, cacheKey); err != nil {
		log.Printf("Failed to invalidate opportunity cache: %v", err)
	} else {
		log.Printf("Invalidated arbitrage opportunities cache")
	}
}

// formatTechnicalSignalMessage creates a formatted message for technical analysis signals
func (ns *NotificationService) formatTechnicalSignalMessage(signals []TechnicalSignalNotification) string {
	if len(signals) == 0 {
		return "No technical analysis signals found."
	}

	// Take top 3 signals for the alert
	topSignals := signals
	if len(signals) > 3 {
		topSignals = signals[:3]
	}

	header := "📊 *Technical Analysis Signals*\n\n"
	message := header
	message += fmt.Sprintf("Found %d high-confidence signals:\n\n", len(signals))

	for i, signal := range topSignals {
		message += fmt.Sprintf("📊 *TA SIGNAL: %s*\n", signal.Symbol)
		message += fmt.Sprintf("🎯 *Signal:* %s\n", signal.SignalText)
		message += fmt.Sprintf("💲 *Current Price:* $%.4f\n", signal.CurrentPrice)
		message += fmt.Sprintf("📈 *Entry:* %s\n", signal.EntryRange)

		// Add targets
		for j, target := range signal.Targets {
			message += fmt.Sprintf("🎯 *Target %d:* $%.4f (%.1f%% profit)\n", j+1, target.Price, target.Profit)
		}

		// Add stop loss
		message += fmt.Sprintf("🛑 *Stop Loss:* $%.4f (%.1f%% risk)\n", signal.StopLoss.Price, signal.StopLoss.Risk)
		message += fmt.Sprintf("📊 *Risk/Reward:* %s\n", signal.RiskReward)

		// Add exchanges
		if len(signal.Exchanges) > 0 {
			exchangeList := strings.Join(signal.Exchanges, ", ")
			message += fmt.Sprintf("🏪 *Exchanges:* %s\n", exchangeList)
		}

		message += fmt.Sprintf("⏰ *Timeframe:* %s\n", signal.Timeframe)
		message += fmt.Sprintf("🎯 *Confidence:* %.1f%%\n", signal.Confidence*100)

		if i < len(topSignals)-1 {
			message += "\n---\n\n"
		}
	}

	if len(signals) > 3 {
		message += fmt.Sprintf("\n...and %d more signals\n\n", len(signals)-3)
	}

	message += "\n⚡ *Trade wisely!* Always manage your risk and position size.\n\n"
	message += "Use /signals to see all current technical signals\n"
	message += "Use /stop to pause these alerts"

	return message
}

// ConvertAggregatedSignalToNotification converts an AggregatedSignal to TechnicalSignalNotification
func (ns *NotificationService) ConvertAggregatedSignalToNotification(signal *AggregatedSignal) *TechnicalSignalNotification {
	// Extract current price from metadata if available
	currentPrice := 0.0
	if signal.Metadata != nil {
		if price, ok := signal.Metadata["current_price"].(float64); ok {
			currentPrice = price
		}
	}

	// Calculate entry range based on current price and action
	entryRange := ""
	if currentPrice > 0 {
		switch signal.Action {
		case "buy":
			lowEntry := currentPrice * 0.995  // 0.5% below current
			highEntry := currentPrice * 1.005 // 0.5% above current
			entryRange = fmt.Sprintf("$%.4f - $%.4f", lowEntry, highEntry)
		case "sell":
			lowEntry := currentPrice * 0.995
			highEntry := currentPrice * 1.005
			entryRange = fmt.Sprintf("$%.4f - $%.4f", lowEntry, highEntry)
		}
	}

	// Calculate targets based on profit potential
	targets := []Target{}
	if currentPrice > 0 {
		profitFloat, _ := signal.ProfitPotential.Float64()
		switch signal.Action {
		case "buy":
			// Target 1: Half of profit potential
			target1Price := currentPrice * (1 + (profitFloat/2)/100)
			target1Profit := (profitFloat / 2)
			targets = append(targets, Target{Price: target1Price, Profit: target1Profit})

			// Target 2: Full profit potential
			target2Price := currentPrice * (1 + profitFloat/100)
			target2Profit := profitFloat
			targets = append(targets, Target{Price: target2Price, Profit: target2Profit})
		case "sell":
			// For sell signals, targets are lower prices
			target1Price := currentPrice * (1 - (profitFloat/2)/100)
			target1Profit := (profitFloat / 2)
			targets = append(targets, Target{Price: target1Price, Profit: target1Profit})

			target2Price := currentPrice * (1 - profitFloat/100)
			target2Profit := profitFloat
			targets = append(targets, Target{Price: target2Price, Profit: target2Profit})
		}
	}

	// Calculate stop loss based on risk level
	stopLoss := StopLoss{}
	if currentPrice > 0 {
		riskFloat, _ := signal.RiskLevel.Float64()
		switch signal.Action {
		case "buy":
			stopLoss.Price = currentPrice * (1 - riskFloat)
			stopLoss.Risk = riskFloat * 100
		case "sell":
			stopLoss.Price = currentPrice * (1 + riskFloat)
			stopLoss.Risk = riskFloat * 100
		}
	}

	// Calculate risk/reward ratio
	riskReward := "1:1"
	if len(targets) > 0 && stopLoss.Risk > 0 {
		avgProfit := (targets[0].Profit + targets[len(targets)-1].Profit) / 2
		ratio := avgProfit / stopLoss.Risk
		riskReward = fmt.Sprintf("1:%.1f", ratio)
	}

	// Extract signal description from metadata or create from indicators
	signalText := ""
	if signal.Metadata != nil {
		if desc, ok := signal.Metadata["description"].(string); ok {
			signalText = desc
		}
	}
	if signalText == "" && len(signal.Indicators) > 0 {
		signalText = strings.Join(signal.Indicators, " + ")
	}

	// Default timeframe
	timeframe := "4H"
	if signal.Metadata != nil {
		if tf, ok := signal.Metadata["timeframe"].(string); ok {
			timeframe = tf
		}
	}

	confidence, _ := signal.Confidence.Float64()

	return &TechnicalSignalNotification{
		Symbol:       signal.Symbol,
		SignalType:   string(signal.SignalType),
		Action:       signal.Action,
		SignalText:   signalText,
		CurrentPrice: currentPrice,
		EntryRange:   entryRange,
		Targets:      targets,
		StopLoss:     stopLoss,
		RiskReward:   riskReward,
		Exchanges:    signal.Exchanges,
		Timeframe:    timeframe,
		Confidence:   confidence,
		Timestamp:    signal.CreatedAt,
	}
}

// getEligibleUsers returns all users who should receive arbitrage alerts with Redis caching
func (ns *NotificationService) getEligibleUsers(ctx context.Context) ([]userModels.User, error) {
	cacheKey := "eligible_users:arbitrage"

	// Try to get from Redis cache first
	if ns.redis != nil {
		cachedData, err := ns.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			var users []userModels.User
			if err := json.Unmarshal([]byte(cachedData), &users); err == nil {
				log.Printf("Retrieved %d eligible users from Redis cache", len(users))
				return users, nil
			}
			log.Printf("Failed to unmarshal cached users: %v", err)
		}
	}

	// Cache miss or Redis unavailable, query database
	query := `
		SELECT id, email, telegram_chat_id, subscription_tier, created_at, updated_at
		FROM users 
		WHERE telegram_chat_id IS NOT NULL 
		  AND telegram_chat_id != ''
		  AND id NOT IN (
			  SELECT DISTINCT user_id 
			  FROM user_alerts 
			  WHERE alert_type = 'arbitrage' 
			    AND is_active = false
			    AND conditions->>'notifications_enabled' = 'false'
		  )
	`

	rows, err := ns.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query eligible users: %w", err)
	}
	defer rows.Close()

	var users []userModels.User
	for rows.Next() {
		var user userModels.User
		if err := rows.Scan(&user.ID, &user.Email, &user.TelegramChatID, &user.SubscriptionTier, &user.CreatedAt, &user.UpdatedAt); err != nil {
			log.Printf("Failed to scan user row: %v", err)
			continue
		}
		users = append(users, user)
	}

	// Cache the result in Redis with 5-minute TTL
	if ns.redis != nil && len(users) > 0 {
		usersJSON, err := json.Marshal(users)
		if err == nil {
			if err := ns.redis.Set(ctx, cacheKey, string(usersJSON), 5*time.Minute); err != nil {
				log.Printf("Failed to cache eligible users: %v", err)
			} else {
				log.Printf("Cached %d eligible users in Redis for 5 minutes", len(users))
			}
		} else {
			log.Printf("Failed to marshal users for caching: %v", err)
		}
	}

	return users, nil
}

// checkRateLimit checks if a user has exceeded the notification rate limit (5 notifications per minute)
func (ns *NotificationService) checkRateLimit(ctx context.Context, userID string) (bool, error) {
	if ns.redis == nil {
		return true, nil // Allow if Redis is not available
	}

	// Use sliding window with Redis sorted set
	rateKey := fmt.Sprintf("rate_limit:notifications:%s", userID)
	now := time.Now().Unix()
	oneMinuteAgo := now - 60

	// Remove old entries (older than 1 minute)
	if err := ns.redis.Client.ZRemRangeByScore(ctx, rateKey, "0", fmt.Sprintf("%d", oneMinuteAgo)).Err(); err != nil {
		log.Printf("Failed to clean old rate limit entries for user %s: %v", userID, err)
	}

	// Count current notifications in the last minute
	count, err := ns.redis.Client.ZCard(ctx, rateKey).Result()
	if err != nil {
		log.Printf("Failed to get rate limit count for user %s: %v", userID, err)
		return true, nil // Allow if Redis operation fails
	}

	// Check if user has exceeded the limit (5 notifications per minute)
	if count >= 5 {
		return false, nil
	}

	// Add current notification to the sliding window
	if err := ns.redis.Client.ZAdd(ctx, rateKey, redis.Z{
		Score:  float64(now),
		Member: fmt.Sprintf("%d", now),
	}).Err(); err != nil {
		log.Printf("Failed to add rate limit entry for user %s: %v", userID, err)
	}

	// Set expiration for the key (2 minutes to be safe)
	if err := ns.redis.Client.Expire(ctx, rateKey, 2*time.Minute).Err(); err != nil {
		log.Printf("Failed to set expiration for rate limit key %s: %v", rateKey, err)
	}

	return true, nil
}

// generateOpportunityHash creates a hash for opportunities to use as cache key
func (ns *NotificationService) generateOpportunityHash(opportunities []ArbitrageOpportunity) string {
	// Create a consistent string representation of opportunities
	var hashData strings.Builder
	for _, opp := range opportunities {
		hashData.WriteString(fmt.Sprintf("%s:%s:%s:%.4f:%.4f:%.2f",
			opp.Symbol, opp.BuyExchange, opp.SellExchange,
			opp.BuyPrice, opp.SellPrice, opp.ProfitPercent))
	}

	hash := md5.Sum([]byte(hashData.String()))
	return hex.EncodeToString(hash[:])
}

// generateTechnicalSignalsHash creates a consistent hash for technical signals
func (ns *NotificationService) generateTechnicalSignalsHash(signals []TechnicalSignalNotification) string {
	var hashData strings.Builder
	for _, signal := range signals {
		hashData.WriteString(fmt.Sprintf("%s:%s:%s:%.4f:%.2f",
			signal.Symbol, signal.SignalType, signal.Action, signal.CurrentPrice, signal.Confidence))
	}

	hash := md5.Sum([]byte(hashData.String()))
	return hex.EncodeToString(hash[:])
}

// getCachedMessage retrieves a cached message from Redis
func (ns *NotificationService) getCachedMessage(ctx context.Context, msgType, hash string) (string, bool) {
	if ns.redis == nil {
		return "", false
	}

	cacheKey := fmt.Sprintf("msg_cache:%s:%s", msgType, hash)
	message, err := ns.redis.Get(ctx, cacheKey)
	if err != nil {
		return "", false
	}
	return message, true
}

// setCachedMessage stores a formatted message in Redis with TTL
func (ns *NotificationService) setCachedMessage(ctx context.Context, msgType, hash, message string) {
	if ns.redis == nil {
		return
	}

	cacheKey := fmt.Sprintf("msg_cache:%s:%s", msgType, hash)
	if err := ns.redis.Set(ctx, cacheKey, message, 5*time.Minute); err != nil {
		log.Printf("Failed to cache message for key %s: %v", cacheKey, err)
	}
}

// sendArbitrageAlert sends a formatted arbitrage alert to a specific user
func (ns *NotificationService) sendArbitrageAlert(ctx context.Context, user userModels.User, opportunities []ArbitrageOpportunity) error {
	if ns.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	// Check rate limit before sending
	allowed, err := ns.checkRateLimit(ctx, user.ID)
	if err != nil {
		log.Printf("Rate limit check failed for user %s: %v", user.ID, err)
	}
	if !allowed {
		log.Printf("Rate limit exceeded for user %s, skipping notification", user.ID)
		return fmt.Errorf("rate limit exceeded for user %s", user.ID)
	}

	chatID, err := strconv.ParseInt(*user.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Generate hash for opportunities to check cache
	oppHash := ns.generateOpportunityHash(opportunities)

	// Try to get cached message first
	var message string
	if cachedMsg, found := ns.getCachedMessage(ctx, "arbitrage", oppHash); found {
		message = cachedMsg
		log.Printf("Using cached arbitrage message for hash %s", oppHash[:8])
	} else {
		// Format the alert message and cache it
		message = ns.formatArbitrageMessage(opportunities)
		ns.setCachedMessage(ctx, "arbitrage", oppHash, message)
		log.Printf("Formatted and cached new arbitrage message for hash %s", oppHash[:8])
	}

	// Send the message
	_, err = ns.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	// Log the notification
	if err := ns.logNotification(ctx, user.ID, "telegram", "arbitrage_alert"); err != nil {
		log.Printf("Failed to log notification for user %s: %v", user.ID, err)
	}

	return nil
}

// sendEnhancedArbitrageAlert sends a formatted enhanced arbitrage alert to a specific user
func (ns *NotificationService) sendEnhancedArbitrageAlert(ctx context.Context, user userModels.User, signal *AggregatedSignal) error {
	if ns.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	// Check rate limit before sending
	allowed, err := ns.checkRateLimit(ctx, user.ID)
	if err != nil {
		log.Printf("Rate limit check failed for user %s: %v", user.ID, err)
	}
	if !allowed {
		log.Printf("Rate limit exceeded for user %s, skipping enhanced arbitrage notification", user.ID)
		return fmt.Errorf("rate limit exceeded for user %s", user.ID)
	}

	chatID, err := strconv.ParseInt(*user.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Generate hash for signal to check cache
	signalHash := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%.4f", signal.Symbol, signal.SignalType, signal.Confidence.InexactFloat64()))))

	// Try to get cached message first
	var message string
	if cachedMsg, found := ns.getCachedMessage(ctx, "enhanced_arbitrage", signalHash); found {
		message = cachedMsg
		log.Printf("Using cached enhanced arbitrage message for hash %s", signalHash[:8])
	} else {
		// Format the enhanced alert message and cache it
		message = ns.formatEnhancedArbitrageMessage(signal)
		ns.setCachedMessage(ctx, "enhanced_arbitrage", signalHash, message)
		log.Printf("Formatted and cached new enhanced arbitrage message for hash %s", signalHash[:8])
	}

	// Send the message
	_, err = ns.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})
	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	// Log the notification
	if err := ns.logNotification(ctx, user.ID, "telegram", "enhanced_arbitrage_alert"); err != nil {
		log.Printf("Failed to log notification for user %s: %v", user.ID, err)
	}

	return nil
}

// NotifyEnhancedArbitrageSignals sends notifications about enhanced arbitrage signals to eligible users
func (ns *NotificationService) NotifyEnhancedArbitrageSignals(ctx context.Context, signals []*AggregatedSignal) error {
	// Get eligible users (those with Telegram chat IDs and arbitrage alerts enabled)
	users, err := ns.getEligibleUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(users) == 0 {
		log.Printf("No eligible users found for enhanced arbitrage notifications")
		return nil
	}

	// Filter arbitrage signals
	arbitrageSignals := make([]*AggregatedSignal, 0)
	for _, signal := range signals {
		if signal.SignalType == SignalTypeArbitrage {
			arbitrageSignals = append(arbitrageSignals, signal)
		}
	}

	if len(arbitrageSignals) == 0 {
		log.Printf("No arbitrage signals found to notify")
		return nil
	}

	// Send notifications to each user
	for _, user := range users {
		for _, signal := range arbitrageSignals {
			if err := ns.sendEnhancedArbitrageAlert(ctx, user, signal); err != nil {
				log.Printf("Failed to send enhanced arbitrage alert to user %s: %v", user.ID, err)
			} else {
				log.Printf("Sent enhanced arbitrage alert to user %s for %s", user.ID, signal.Symbol)
			}
		}
	}

	log.Printf("Sent enhanced arbitrage notifications to %d users: %d signals", len(users), len(arbitrageSignals))
	return nil
}

// formatArbitrageMessage creates a formatted message for arbitrage opportunities
func (ns *NotificationService) formatArbitrageMessage(opportunities []ArbitrageOpportunity) string {
	if len(opportunities) == 0 {
		return "No arbitrage opportunities found."
	}

	// Take top 3 opportunities for the alert
	topOpportunities := opportunities
	if len(opportunities) > 3 {
		topOpportunities = opportunities[:3]
	}

	// Determine message header based on opportunity type
	header := "🚨 *Arbitrage Alert!*\n\n"
	if len(opportunities) > 0 {
		switch opportunities[0].OpportunityType {
		case "arbitrage":
			header = "🚀 *True Arbitrage Opportunities*\n\n"
		case "technical":
			header = "📊 *Technical Analysis Signals*\n\n"
		case "ai_generated":
			header = "🤖 *AI-Generated Opportunities*\n\n"
		}
	}

	message := header
	message += fmt.Sprintf("Found %d profitable opportunities:\n\n", len(opportunities))

	for i, opp := range topOpportunities {
		message += fmt.Sprintf("*%d. %s*\n", i+1, opp.Symbol)
		message += fmt.Sprintf("💰 Profit: *%.2f%%*\n", opp.ProfitPercent)
		message += fmt.Sprintf("📈 Buy: %s @ $%.4f\n", opp.BuyExchange, opp.BuyPrice)
		message += fmt.Sprintf("📉 Sell: %s @ $%.4f\n", opp.SellExchange, opp.SellPrice)
		message += "\n"
	}

	if len(opportunities) > 3 {
		message += fmt.Sprintf("...and %d more opportunities\n\n", len(opportunities)-3)
	}

	message += "⚡ *Act fast!* These opportunities may disappear quickly.\n\n"
	message += "Use /opportunities to see all current opportunities\n"
	message += "Use /stop to pause these alerts"

	return message
}

// formatEnhancedArbitrageMessage creates a formatted message for enhanced arbitrage signals with price ranges
func (ns *NotificationService) formatEnhancedArbitrageMessage(signal *AggregatedSignal) string {
	if signal == nil || signal.SignalType != SignalTypeArbitrage {
		return "No arbitrage signal found."
	}

	// Extract metadata
	metadata := signal.Metadata
	buyPriceRange, _ := metadata["buy_price_range"].(map[string]interface{})
	sellPriceRange, _ := metadata["sell_price_range"].(map[string]interface{})
	profitRange, _ := metadata["profit_range"].(map[string]interface{})
	buyExchanges, _ := metadata["buy_exchanges"].([]string)
	sellExchanges, _ := metadata["sell_exchanges"].([]string)
	opportunityCount, _ := metadata["opportunity_count"].(int)
	minVolume, _ := metadata["min_volume"].(decimal.Decimal)
	validityMinutes, _ := metadata["validity_minutes"].(int)

	// Build the message
	message := fmt.Sprintf("🔄 *ARBITRAGE ALERT: %s*\n\n", signal.Symbol)

	// Profit range
	if profitRange != nil {
		minPercent, _ := profitRange["min_percent"].(decimal.Decimal)
		maxPercent, _ := profitRange["max_percent"].(decimal.Decimal)
		minDollar, _ := profitRange["min_dollar"].(decimal.Decimal)
		maxDollar, _ := profitRange["max_dollar"].(decimal.Decimal)
		baseAmount, _ := profitRange["base_amount"].(decimal.Decimal)

		if minPercent.Equal(maxPercent) {
			message += fmt.Sprintf("💰 Profit: *%.2f%%* ($%.0f on $%.0f)\n",
				minPercent.InexactFloat64(), minDollar.InexactFloat64(), baseAmount.InexactFloat64())
		} else {
			message += fmt.Sprintf("💰 Profit: *%.2f%% - %.2f%%* ($%.0f - $%.0f on $%.0f)\n",
				minPercent.InexactFloat64(), maxPercent.InexactFloat64(),
				minDollar.InexactFloat64(), maxDollar.InexactFloat64(), baseAmount.InexactFloat64())
		}
	}

	// Buy price range
	if buyPriceRange != nil && len(buyExchanges) > 0 {
		buyMin, _ := buyPriceRange["min"].(decimal.Decimal)
		buyMax, _ := buyPriceRange["max"].(decimal.Decimal)

		exchangeList := strings.Join(buyExchanges, ", ")
		if buyMin.Equal(buyMax) {
			message += fmt.Sprintf("📈 BUY: $%.4f (%s)\n", buyMin.InexactFloat64(), exchangeList)
		} else {
			message += fmt.Sprintf("📈 BUY: $%.4f - $%.4f (%s)\n",
				buyMin.InexactFloat64(), buyMax.InexactFloat64(), exchangeList)
		}
	}

	// Sell price range
	if sellPriceRange != nil && len(sellExchanges) > 0 {
		sellMin, _ := sellPriceRange["min"].(decimal.Decimal)
		sellMax, _ := sellPriceRange["max"].(decimal.Decimal)

		exchangeList := strings.Join(sellExchanges, ", ")
		if sellMin.Equal(sellMax) {
			message += fmt.Sprintf("📉 SELL: $%.4f (%s)\n", sellMax.InexactFloat64(), exchangeList)
		} else {
			message += fmt.Sprintf("📉 SELL: $%.4f - $%.4f (%s)\n",
				sellMin.InexactFloat64(), sellMax.InexactFloat64(), exchangeList)
		}
	}

	// Validity and volume info
	if validityMinutes > 0 {
		message += fmt.Sprintf("⏰ Valid for: *%d minutes*\n", validityMinutes)
	}

	if !minVolume.IsZero() {
		message += fmt.Sprintf("🎯 Min Volume: *$%.0f*\n", minVolume.InexactFloat64())
	}

	// Additional info
	if opportunityCount > 1 {
		message += fmt.Sprintf("📊 Opportunities: *%d*\n", opportunityCount)
	}

	message += fmt.Sprintf("🎯 Confidence: *%.1f%%*\n", signal.Confidence.Mul(decimal.NewFromFloat(100)).InexactFloat64())

	message += "\n⚡ *Act fast!* Arbitrage opportunities disappear quickly.\n"
	message += "💡 *Min Volume* helps filter out low-liquidity fake signals."

	return message
}

// logNotification records the notification in the database
func (ns *NotificationService) logNotification(ctx context.Context, userID, notificationType, content string) error {
	query := `
		INSERT INTO alert_notifications (user_id, notification_type, content, sent_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	now := time.Now()
	_, err := ns.db.Pool.Exec(ctx, query, userID, notificationType, content, now, now)
	if err != nil {
		return fmt.Errorf("failed to log notification: %w", err)
	}

	return nil
}

// CheckUserNotificationPreferences checks if a user wants to receive arbitrage notifications with Redis caching
func (ns *NotificationService) CheckUserNotificationPreferences(ctx context.Context, userID string) (bool, error) {
	cacheKey := fmt.Sprintf("user_preferences:%s:arbitrage", userID)

	// Try to get from Redis cache first
	if ns.redis != nil {
		cachedData, err := ns.redis.Get(ctx, cacheKey)
		if err == nil && cachedData != "" {
			switch cachedData {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
		}
	}

	// Cache miss or Redis unavailable, query database
	query := `
		SELECT COUNT(*) 
		FROM user_alerts 
		WHERE user_id = $1 
		  AND alert_type = 'arbitrage' 
		  AND is_active = false
		  AND conditions->>'notifications_enabled' = 'false'
	`

	var count int
	err := ns.db.Pool.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return true, fmt.Errorf("failed to check user preferences: %w", err) // Default to enabled on error
	}

	result := count == 0 // Return true if no disabled alerts found

	// Cache the result in Redis with 5-minute TTL
	if ns.redis != nil {
		cacheValue := "false"
		if result {
			cacheValue = "true"
		}
		if err := ns.redis.Set(ctx, cacheKey, cacheValue, 5*time.Minute); err != nil {
			log.Printf("Failed to cache user preferences for user %s: %v", userID, err)
		} else {
			log.Printf("Cached user preferences for user %s: %v", userID, result)
		}
	}

	return result, nil
}

// NotifyTechnicalSignals sends notifications about technical analysis signals to eligible users
func (ns *NotificationService) NotifyTechnicalSignals(ctx context.Context, signals []TechnicalSignalNotification) error {
	// Get eligible users (those with Telegram chat IDs and technical alerts enabled)
	users, err := ns.getEligibleUsers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get eligible users: %w", err)
	}

	if len(users) == 0 {
		log.Printf("No eligible users found for technical signal notifications")
		return nil
	}

	// Send notifications to each user
	for _, user := range users {
		if err := ns.sendTechnicalAlert(ctx, user, signals); err != nil {
			log.Printf("Failed to send technical alert to user %s: %v", user.ID, err)
		} else {
			log.Printf("Sent technical alert to user %s", user.ID)
		}
	}

	log.Printf("Sent technical signal notifications to %d users: %d signals", len(users), len(signals))
	return nil
}

// sendTechnicalAlert sends a formatted technical analysis alert to a specific user
func (ns *NotificationService) sendTechnicalAlert(ctx context.Context, user userModels.User, signals []TechnicalSignalNotification) error {
	if ns.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	// Check rate limit before sending
	allowed, err := ns.checkRateLimit(ctx, user.ID)
	if err != nil {
		log.Printf("Rate limit check failed for user %s: %v", user.ID, err)
	}
	if !allowed {
		log.Printf("Rate limit exceeded for user %s, skipping technical alert", user.ID)
		return fmt.Errorf("rate limit exceeded for user %s", user.ID)
	}

	chatID, err := strconv.ParseInt(*user.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Generate hash for signals to check cache
	signalsHash := ns.generateTechnicalSignalsHash(signals)

	// Try to get cached message first
	var message string
	if cachedMsg, found := ns.getCachedMessage(ctx, "technical", signalsHash); found {
		message = cachedMsg
		log.Printf("Using cached technical message for hash %s", signalsHash[:8])
	} else {
		// Format the technical alert message and cache it
		message = ns.formatTechnicalSignalMessage(signals)
		ns.setCachedMessage(ctx, "technical", signalsHash, message)
		log.Printf("Formatted and cached new technical message for hash %s", signalsHash[:8])
	}

	// Send the message
	_, err = ns.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      message,
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		return fmt.Errorf("failed to send telegram message: %w", err)
	}

	// Log the notification
	if err := ns.logNotification(ctx, user.ID, "telegram", "technical_alert"); err != nil {
		log.Printf("Failed to log notification for user %s: %v", user.ID, err)
	}

	return nil
}
