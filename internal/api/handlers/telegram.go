package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/gin-gonic/gin"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	userModels "github.com/irfandi/celebrum-ai-go/internal/models"
	"github.com/irfandi/celebrum-ai-go/internal/services"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

// TelegramHandler handles Telegram webhook requests and bot interactions.
type TelegramHandler struct {
	db               *database.PostgresDB
	config           *config.TelegramConfig
	bot              *bot.Bot
	arbitrageHandler *ArbitrageHandler
	signalAggregator *services.SignalAggregator
	redis            *redis.Client
	pollingActive    bool
	pollingCancel    context.CancelFunc
}

// NewTelegramHandler creates a new Telegram handler.
// It initializes the Telegram bot and sets up either webhook or polling mode based on configuration.
//
// Parameters:
//
//	db: Database connection.
//	cfg: Telegram configuration.
//	arbitrageHandler: Handler for arbitrage operations.
//	signalAggregator: Service for signal aggregation.
//	redisClient: Redis client.
//
// Returns:
//
//	*TelegramHandler: Initialized handler.
func NewTelegramHandler(db *database.PostgresDB, cfg *config.TelegramConfig, arbitrageHandler *ArbitrageHandler, signalAggregator *services.SignalAggregator, redisClient *redis.Client) *TelegramHandler {
	log.Printf("[TELEGRAM] NewTelegramHandler called")
	// Return handler with nil bot if config is not provided
	if cfg == nil {
		log.Printf("[TELEGRAM] WARNING: Telegram config is nil, returning handler with nil bot")
		return &TelegramHandler{
			db:               db,
			config:           nil,
			bot:              nil,
			arbitrageHandler: arbitrageHandler,
			signalAggregator: signalAggregator,
			redis:            redisClient,
		}
	}

	botTokenDisplay := "<empty>"
	if len(cfg.BotToken) > 10 {
		botTokenDisplay = cfg.BotToken[:10] + "..."
	} else if len(cfg.BotToken) > 0 {
		botTokenDisplay = cfg.BotToken + "..."
	}
	log.Printf("[TELEGRAM] Config: BotToken=%s, UsePolling=%t, WebhookURL=%s", botTokenDisplay, cfg.UsePolling, cfg.WebhookURL)

	// Create handler first
	handler := &TelegramHandler{
		db:               db,
		config:           cfg,
		bot:              nil, // Will be set after bot creation
		arbitrageHandler: arbitrageHandler,
		signalAggregator: signalAggregator,
		redis:            redisClient,
		pollingActive:    false,
		pollingCancel:    nil,
	}

	// Initialize the bot with handler reference
	log.Printf("[TELEGRAM] Creating bot instance...")
	b, err := bot.New(cfg.BotToken, bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
		// Process updates in polling mode
		log.Printf("[TELEGRAM] Received update in default handler")
		if err := handler.processUpdate(ctx, update); err != nil {
			log.Printf("[TELEGRAM] ERROR: Failed to process update: %v", err)
		}
	}))
	if err != nil {
		log.Printf("[TELEGRAM] ERROR: Failed to create Telegram bot: %v", err)
		log.Printf("[TELEGRAM] WARNING: Telegram bot functionality will be disabled")
		// Return handler with nil bot - webhook will handle gracefully
		return handler
	}

	log.Printf("[TELEGRAM] Bot instance created successfully")

	// Set the bot in the handler
	handler.bot = b

	// Start polling mode if configured
	if cfg.UsePolling {
		log.Printf("[TELEGRAM] Starting Telegram bot in polling mode")
		go handler.StartPolling()
	} else {
		log.Printf("[TELEGRAM] Telegram bot configured for webhook mode")
		log.Printf("[TELEGRAM] Webhook should be set to: %s", cfg.WebhookURL)
	}

	return handler
}

// Using models from go-telegram/bot package instead of custom structs

// HandleWebhook processes incoming Telegram webhook requests.
// It parses the update and delegates processing to the bot framework.
//
// Parameters:
//
//	c: Gin context.
func (h *TelegramHandler) HandleWebhook(c *gin.Context) {
	log.Printf("[TELEGRAM] HandleWebhook called from %s", c.ClientIP())

	if h.bot == nil {
		log.Printf("[TELEGRAM] ERROR: Telegram bot is not initialized, ignoring webhook")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telegram bot not available"})
		return
	}

	// Parse the update
	var update models.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		log.Printf("[TELEGRAM] ERROR: Failed to parse Telegram update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	log.Printf("[TELEGRAM] Webhook update received: update_id=%d", update.ID)

	// Process the update using the bot framework
	go func() {
		ctx := context.Background()
		if err := h.processUpdate(ctx, &update); err != nil {
			log.Printf("[TELEGRAM] ERROR: Failed to process Telegram update: %v", err)
		}
	}()

	// Always return 200 OK to acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"ok": true})
	log.Printf("[TELEGRAM] Webhook acknowledged")
}

// processUpdate processes a Telegram update
func (h *TelegramHandler) processUpdate(ctx context.Context, update *models.Update) error {
	log.Printf("[TELEGRAM] processUpdate called")

	if update.Message == nil {
		log.Printf("[TELEGRAM] Update has no message, ignoring")
		return nil // Ignore non-message updates for now
	}

	message := update.Message

	if message.From == nil || message.Chat.ID == 0 {
		log.Printf("[TELEGRAM] ERROR: Invalid message: missing from or chat")
		return fmt.Errorf("invalid message: missing from or chat")
	}

	// Security: Log message metadata without exposing user content
	log.Printf("[TELEGRAM] Message received from user %d in chat %d (length: %d)", message.From.ID, message.Chat.ID, len(message.Text))

	// Handle different commands
	text := strings.TrimSpace(message.Text)
	if strings.HasPrefix(text, "/") {
		log.Printf("[TELEGRAM] Processing command: %s", text)
		return h.handleCommand(ctx, message, text)
	}

	// Handle regular text messages
	log.Printf("[TELEGRAM] Processing text message from user %d (length: %d)", message.From.ID, len(text))
	return h.handleTextMessage(ctx, message, text)
}

// handleCommand processes bot commands
func (h *TelegramHandler) handleCommand(ctx context.Context, message *models.Message, command string) error {
	chatID := message.Chat.ID
	userID := message.From.ID

	switch {
	case strings.HasPrefix(command, "/start"):
		return h.handleStartCommand(ctx, chatID, userID, message.From)
	case strings.HasPrefix(command, "/opportunities"):
		return h.handleOpportunitiesCommand(ctx, chatID, userID)
	case strings.HasPrefix(command, "/settings"):
		return h.handleSettingsCommand(ctx, chatID, userID)
	case strings.HasPrefix(command, "/help"):
		return h.handleHelpCommand(ctx, chatID)
	case strings.HasPrefix(command, "/upgrade"):
		return h.handleUpgradeCommand(ctx, chatID, userID)
	case strings.HasPrefix(command, "/status"):
		return h.handleStatusCommand(ctx, chatID, userID)
	case strings.HasPrefix(command, "/stop"):
		return h.handleStopCommand(ctx, chatID, userID)
	case strings.HasPrefix(command, "/resume"):
		return h.handleResumeCommand(ctx, chatID, userID)
	default:
		return h.sendMessage(ctx, chatID, "Unknown command. Use /help to see available commands.")
	}
}

// handleStartCommand handles the /start command
func (h *TelegramHandler) handleStartCommand(ctx context.Context, chatID, userID int64, from *models.User) error {
	log.Printf("[TELEGRAM] handleStartCommand called for user %d, chat %d", userID, chatID)

	// Check if user already exists
	chatIDStr := strconv.FormatInt(chatID, 10)
	var existingUser userModels.User

	log.Printf("[TELEGRAM] Checking if user exists with telegram_chat_id=%s", chatIDStr)
	err := h.db.Pool.QueryRow(ctx, "SELECT id, email, telegram_chat_id, subscription_tier, created_at, updated_at FROM users WHERE telegram_chat_id = $1", chatIDStr).Scan(
		&existingUser.ID, &existingUser.Email, &existingUser.TelegramChatID, &existingUser.SubscriptionTier, &existingUser.CreatedAt, &existingUser.UpdatedAt)
	if err == nil {
		// User already exists, send welcome back message
		log.Printf("[TELEGRAM] User already exists: %s", existingUser.ID)
		return h.sendMessage(ctx, chatID, "Welcome back! 🎉\n\nYou're already registered and receiving alerts.\n\nUse /opportunities to see current arbitrage opportunities.")
	}

	if err != pgx.ErrNoRows {
		log.Printf("[TELEGRAM] ERROR: Database error checking user: %v", err)
		return fmt.Errorf("database error: %w", err)
	}

	log.Printf("[TELEGRAM] User not found, creating new user")

	// Create new user - let database generate UUID automatically
	email := fmt.Sprintf("telegram_%d@celebrum.ai", userID)
	telegramChatID := &chatIDStr
	subscriptionTier := "free"
	now := time.Now()

	log.Printf("[TELEGRAM] Inserting new user with email=%s, telegram_chat_id=%s", email, chatIDStr)
	_, err = h.db.Pool.Exec(ctx, `
		INSERT INTO users (email, password_hash, telegram_chat_id, subscription_tier, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5, $6)`,
		email, "telegram_user", telegramChatID, subscriptionTier, now, now)
	if err != nil {
		log.Printf("[TELEGRAM] ERROR: Failed to create user: %v", err)
		return fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("[TELEGRAM] User created successfully, sending welcome message")

	// Send welcome message
	welcomeMsg := `🚀 Welcome to Celebrum AI!

✅ You're now registered and ready to receive arbitrage alerts!

🔍 What you get:
• Real-time arbitrage opportunities
• Profit calculations across exchanges
• Instant notifications when opportunities arise

📊 Current opportunities: Loading...
💰 Best opportunity: Scanning markets...

Use /opportunities to see all current opportunities
Use /help to see all available commands

🎯 Want more features? /upgrade for premium access!`

	return h.sendMessage(ctx, chatID, welcomeMsg)
}

// handleOpportunitiesCommand handles the /opportunities command
func (h *TelegramHandler) handleOpportunitiesCommand(ctx context.Context, chatID, userID int64) error {
	// Check for cached aggregated signals first
	cachedSignals, err := h.getCachedAggregatedSignals(ctx)
	if err == nil && len(cachedSignals) > 0 {
		log.Printf("Using cached aggregated signals for Telegram user %d", userID)
		return h.sendAggregatedSignalsMessage(ctx, chatID, cachedSignals)
	}

	// If no cached signals, fetch new ones from SignalAggregator
	if h.signalAggregator == nil {
		log.Printf("SignalAggregator not available")
		return h.sendMessage(ctx, chatID, "❌ Signal aggregation service is not available. Please try again later.")
	}

	// Get active aggregated signals (limit to 10 for display)
	signals, err := h.signalAggregator.GetActiveAggregatedSignals(ctx, 10)
	if err != nil {
		log.Printf("Failed to get aggregated signals: %v", err)
		return h.sendMessage(ctx, chatID, "❌ Failed to fetch trading signals. Please try again later.")
	}

	// Cache the signals
	h.cacheAggregatedSignals(ctx, signals)

	// Send the aggregated signals
	return h.sendAggregatedSignalsMessage(ctx, chatID, signals)
}

// handleStopCommand handles the /stop command
func (h *TelegramHandler) handleStopCommand(ctx context.Context, chatID, userID int64) error {
	if h.redis == nil {
		return h.sendMessage(ctx, chatID, "❌ Service temporarily unavailable. Please try again later.")
	}

	// Set notification preference to false in Redis
	// Key: telegram:user:{userID}:notifications_enabled
	key := fmt.Sprintf("telegram:user:%d:notifications_enabled", userID)
	if err := h.redis.Set(ctx, key, "false", 0).Err(); err != nil {
		log.Printf("Failed to disable notifications for user %d: %v", userID, err)
		return h.sendMessage(ctx, chatID, "❌ Failed to update settings. Please try again.")
	}

	msg := `⏸️ Notifications Paused

You will no longer receive arbitrage alerts.

Use /resume to start receiving alerts again.`

	return h.sendMessage(ctx, chatID, msg)
}

// handleResumeCommand handles the /resume command
func (h *TelegramHandler) handleResumeCommand(ctx context.Context, chatID, userID int64) error {
	if h.redis == nil {
		return h.sendMessage(ctx, chatID, "❌ Service temporarily unavailable. Please try again later.")
	}

	// Set notification preference to true in Redis
	key := fmt.Sprintf("telegram:user:%d:notifications_enabled", userID)
	if err := h.redis.Set(ctx, key, "true", 0).Err(); err != nil {
		log.Printf("Failed to enable notifications for user %d: %v", userID, err)
		return h.sendMessage(ctx, chatID, "❌ Failed to update settings. Please try again.")
	}

	msg := `▶️ Notifications Resumed

You will now receive arbitrage alerts again.

Use /opportunities to see current opportunities.`

	return h.sendMessage(ctx, chatID, msg)
}

// handleSettingsCommand handles the /settings command
func (h *TelegramHandler) handleSettingsCommand(ctx context.Context, chatID, userID int64) error {
	// Check current notification status
	notificationsEnabled := true
	if h.redis != nil {
		key := fmt.Sprintf("telegram:user:%d:notifications_enabled", userID)
		val, err := h.redis.Get(ctx, key).Result()
		if err == nil && val == "false" {
			notificationsEnabled = false
		}
	}

	statusIcon := "✅"
	statusText := "ON"
	if !notificationsEnabled {
		statusIcon = "❌"
		statusText = "OFF"
	}

	msg := fmt.Sprintf(`⚙️ Alert Settings:

🔔 Notifications: %s %s
📊 Min Profit Threshold: 1.5%%
⏰ Alert Frequency: Every 5 minutes
💰 Subscription: Free Tier

To change settings:
/stop - Pause notifications
/resume - Resume notifications
/upgrade - Upgrade to premium for more options`, statusIcon, statusText)

	return h.sendMessage(ctx, chatID, msg)
}

// handleUpgradeCommand handles the /upgrade command
func (h *TelegramHandler) handleUpgradeCommand(ctx context.Context, chatID, userID int64) error {
	msg := `🎯 Upgrade to Premium

✨ Premium Benefits:
• Unlimited alerts
• Instant notifications
• Custom profit thresholds
• Website dashboard access
• Priority support

💰 Price: $29/month

💳 Payment Options:
1️⃣ Telegram Stars (⭐ 290)
2️⃣ Credit Card
3️⃣ Crypto (USDT)

To upgrade, please contact our support team or visit our website (coming soon).`

	return h.sendMessage(ctx, chatID, msg)
}

// StartPolling starts the Telegram bot in polling mode.
// It launches a goroutine to continuously fetch updates.
func (h *TelegramHandler) StartPolling() {
	if h.bot == nil {
		log.Printf("[TELEGRAM] ERROR: Cannot start polling: Telegram bot is not initialized")
		return
	}

	if h.pollingActive {
		log.Printf("[TELEGRAM] Polling is already active, skipping")
		return
	}

	log.Printf("[TELEGRAM] Starting Telegram bot polling...")
	h.pollingActive = true

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	h.pollingCancel = cancel

	// Start the bot with polling - this will handle updates automatically
	go func() {
		defer func() {
			h.pollingActive = false
			log.Printf("[TELEGRAM] Polling stopped")
		}()

		log.Printf("[TELEGRAM] Bot polling started, listening for updates...")
		// The bot.Start method handles polling internally
		h.bot.Start(ctx)
	}()

	log.Printf("[TELEGRAM] Polling goroutine launched")
}

// StopPolling stops the Telegram bot polling.
// It cancels the context used by the polling loop.
func (h *TelegramHandler) StopPolling() {
	if !h.pollingActive {
		log.Printf("Polling is not active")
		return
	}

	log.Printf("Stopping Telegram bot polling...")
	if h.pollingCancel != nil {
		h.pollingCancel()
		h.pollingCancel = nil
	}
	h.pollingActive = false
}

// handleHelpCommand handles the /help command
func (h *TelegramHandler) handleHelpCommand(ctx context.Context, chatID int64) error {
	msg := `🤖 Celebrum AI Bot Commands:

/start - Register and get started
/opportunities - View current arbitrage opportunities
/settings - Configure your alert preferences
/upgrade - Upgrade to premium subscription
/status - Check your account status
/stop - Pause all notifications
/resume - Resume notifications
/help - Show this help message

💡 Tip: You'll receive automatic alerts when profitable arbitrage opportunities are detected!`

	return h.sendMessage(ctx, chatID, msg)
}

// handleStatusCommand handles the /status command
func (h *TelegramHandler) handleStatusCommand(ctx context.Context, chatID, userID int64) error {
	// Get user from database
	chatIDStr := strconv.FormatInt(chatID, 10)
	var user userModels.User
	err := h.db.Pool.QueryRow(ctx, "SELECT id, email, telegram_chat_id, subscription_tier, created_at, updated_at FROM users WHERE telegram_chat_id = $1", chatIDStr).Scan(
		&user.ID, &user.Email, &user.TelegramChatID, &user.SubscriptionTier, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return h.sendMessage(ctx, chatID, "User not found. Please use /start to register.")
	}

	userIDDisplay := user.ID
	if len(userIDDisplay) > 8 {
		userIDDisplay = userIDDisplay[:8] + "..."
	}

	msg := fmt.Sprintf(`📊 Account Status:

👤 User ID: %s
💰 Subscription: %s
📅 Member since: %s
🔔 Notifications: Active

📈 Today's alerts: 0
💎 Opportunities found: 0

Use /upgrade to unlock premium features!`,
		userIDDisplay,
		cases.Title(language.English).String(user.SubscriptionTier),
		user.CreatedAt.Format("Jan 2, 2006"))

	return h.sendMessage(ctx, chatID, msg)
}

// handleTextMessage processes regular text messages
func (h *TelegramHandler) handleTextMessage(ctx context.Context, message *models.Message, text string) error {
	chatID := message.Chat.ID

	// For now, just acknowledge the message and suggest using commands
	msg := "Thanks for your message! 👋\n\nI work best with commands. Try /help to see what I can do!"
	return h.sendMessage(ctx, chatID, msg)
}

// sendMessage sends a message to a Telegram chat using the bot framework
func (h *TelegramHandler) sendMessage(ctx context.Context, chatID int64, text string) error {
	log.Printf("[TELEGRAM] sendMessage called for chat %d", chatID)

	if h.bot == nil {
		log.Printf("[TELEGRAM] ERROR: Telegram bot is not initialized, cannot send message to chat %d", chatID)
		return fmt.Errorf("telegram bot not available")
	}

	// Truncate message preview for logging
	messagePreview := text
	if len(text) > 50 {
		messagePreview = text[:50] + "..."
	}
	log.Printf("[TELEGRAM] Sending message to chat %d: %s", chatID, messagePreview)

	_, err := h.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		log.Printf("[TELEGRAM] ERROR: Failed to send message to chat %d: %v", chatID, err)
		return err
	}

	log.Printf("[TELEGRAM] Message sent successfully to chat %d", chatID)
	return nil
}

// Removed unused cacheTelegramOpportunities function

// cacheAggregatedSignals caches aggregated signals in Redis
func (h *TelegramHandler) cacheAggregatedSignals(ctx context.Context, signals []*services.AggregatedSignal) {
	if h.redis == nil {
		return
	}

	data, err := json.Marshal(signals)
	if err != nil {
		log.Printf("Error marshaling aggregated signals for cache: %v", err)
		return
	}

	cacheKey := "telegram:aggregated_signals"
	err = h.redis.Set(ctx, cacheKey, data, 3*time.Minute).Err() // Shorter cache time for signals
	if err != nil {
		log.Printf("Error caching aggregated signals: %v", err)
	}
}

// getCachedAggregatedSignals retrieves cached aggregated signals from Redis
func (h *TelegramHandler) getCachedAggregatedSignals(ctx context.Context) ([]*services.AggregatedSignal, error) {
	if h.redis == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	cacheKey := "telegram:aggregated_signals"
	data, err := h.redis.Get(ctx, cacheKey).Result()
	if err != nil {
		return nil, err
	}

	var signals []*services.AggregatedSignal
	err = json.Unmarshal([]byte(data), &signals)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling cached aggregated signals: %w", err)
	}

	return signals, nil
}

// Removed unused getString function

// Removed unused getFloat64 function

// getCachedTelegramOpportunities retrieves cached opportunities from Redis
func (h *TelegramHandler) getCachedTelegramOpportunities(ctx context.Context) ([]ArbitrageOpportunity, error) {
	if h.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	cacheKey := "telegram:opportunities:latest"
	cachedData, err := h.redis.Get(ctx, cacheKey).Result()
	if err != nil || cachedData == "" {
		return nil, fmt.Errorf("no cached Telegram opportunities found")
	}

	var opportunities []ArbitrageOpportunity
	if err := json.Unmarshal([]byte(cachedData), &opportunities); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached Telegram opportunities: %w", err)
	}

	log.Printf("Retrieved cached Telegram opportunities from Redis")
	return opportunities, nil
}

// Removed unused sendOpportunitiesMessage function

// sendAggregatedSignalsMessage sends aggregated trading signals to Telegram
func (h *TelegramHandler) sendAggregatedSignalsMessage(ctx context.Context, chatID int64, signals []*services.AggregatedSignal) error {
	if len(signals) == 0 {
		return h.sendMessage(ctx, chatID, "📊 No trading signals found at the moment.")
	}

	// Limit to top 5 signals for better readability
	if len(signals) > 5 {
		signals = signals[:5]
	}

	message := "🎯 *Top Trading Signals*\n\n"

	for i, signal := range signals {
		// Signal header with type icon
		var typeIcon string
		switch signal.SignalType {
		case "arbitrage":
			typeIcon = "⚡"
		case "technical":
			typeIcon = "📈"
		default:
			typeIcon = "🔍"
		}

		message += fmt.Sprintf("*%d. %s %s*\n", i+1, typeIcon, signal.Symbol)
		message += fmt.Sprintf("🎯 Action: *%s*\n", strings.ToUpper(signal.Action))
		message += fmt.Sprintf("💪 Strength: *%s*\n", strings.ToUpper(string(signal.Strength)))
		confidenceFloat, _ := signal.Confidence.Float64()
		profitPotentialFloat, _ := signal.ProfitPotential.Float64()
		message += fmt.Sprintf("🎲 Confidence: *%.1f%%*\n", confidenceFloat*100)
		message += fmt.Sprintf("💰 Profit Potential: *%.2f%%*\n", profitPotentialFloat*100)

		// Add signal-specific details
		if signal.SignalType == "arbitrage" && len(signal.Exchanges) > 0 {
			message += fmt.Sprintf("🏪 Exchanges: %s\n", strings.Join(signal.Exchanges, ", "))
		}

		if signal.SignalType == "technical" && len(signal.Indicators) > 0 {
			indicatorNames := make([]string, 0, len(signal.Indicators))
			indicatorNames = append(indicatorNames, signal.Indicators...)
			message += fmt.Sprintf("📊 Indicators: %s\n", strings.Join(indicatorNames, ", "))
		}

		// Risk level
		riskLevelFloat, _ := signal.RiskLevel.Float64()
		var riskIcon string
		switch {
		case riskLevelFloat <= 0.3:
			riskIcon = "🟢"
		case riskLevelFloat <= 0.6:
			riskIcon = "🟡"
		default:
			riskIcon = "🔴"
		}
		message += fmt.Sprintf("⚠️ Risk: %s *%.1f%%*\n", riskIcon, riskLevelFloat*100)

		// Time info
		timeAgo := time.Since(signal.CreatedAt)
		var timeStr string
		if timeAgo < time.Minute {
			timeStr = "just now"
		} else if timeAgo < time.Hour {
			timeStr = fmt.Sprintf("%dm ago", int(timeAgo.Minutes()))
		} else {
			timeStr = fmt.Sprintf("%dh ago", int(timeAgo.Hours()))
		}
		message += fmt.Sprintf("⏰ Generated: %s\n", timeStr)

		if i < len(signals)-1 {
			message += "\n"
		}
	}

	message += "\n🔄 _Auto-updated every 3 minutes_"

	return h.sendMessage(ctx, chatID, message)
}
