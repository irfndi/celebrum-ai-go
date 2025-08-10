package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/irfndi/celebrum-ai-go/internal/config"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	userModels "github.com/irfndi/celebrum-ai-go/internal/models"
	"github.com/jackc/pgx/v5"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TelegramHandler handles Telegram webhook requests
type TelegramHandler struct {
	db               *database.PostgresDB
	config           *config.TelegramConfig
	bot              *bot.Bot
	arbitrageHandler *ArbitrageHandler
}

// NewTelegramHandler creates a new Telegram handler
func NewTelegramHandler(db *database.PostgresDB, cfg *config.TelegramConfig, arbitrageHandler *ArbitrageHandler) *TelegramHandler {
	// Return handler with nil bot if config is not provided
	if cfg == nil {
		return &TelegramHandler{
			db:               db,
			config:           nil,
			bot:              nil,
			arbitrageHandler: arbitrageHandler,
		}
	}

	// Initialize the bot
	b, err := bot.New(cfg.BotToken, bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
		// This will be handled by our custom webhook handler
	}))
	if err != nil {
		log.Printf("Failed to create Telegram bot: %v", err)
		log.Printf("Telegram bot functionality will be disabled")
		// Return handler with nil bot - webhook will handle gracefully
		return &TelegramHandler{
			db:               db,
			config:           cfg,
			bot:              nil,
			arbitrageHandler: arbitrageHandler,
		}
	}

	return &TelegramHandler{
		db:               db,
		config:           cfg,
		bot:              b,
		arbitrageHandler: arbitrageHandler,
	}
}

// Using models from go-telegram/bot package instead of custom structs

// HandleWebhook processes incoming Telegram webhook requests
func (h *TelegramHandler) HandleWebhook(c *gin.Context) {
	if h.bot == nil {
		log.Printf("Telegram bot is not initialized, ignoring webhook")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Telegram bot not available"})
		return
	}

	// Parse the update
	var update models.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		log.Printf("Failed to parse Telegram update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Process the update using the bot framework
	go func() {
		ctx := context.Background()
		if err := h.processUpdate(ctx, &update); err != nil {
			log.Printf("Failed to process Telegram update: %v", err)
		}
	}()

	// Always return 200 OK to acknowledge receipt
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// processUpdate processes a Telegram update
func (h *TelegramHandler) processUpdate(ctx context.Context, update *models.Update) error {
	if update.Message == nil {
		return nil // Ignore non-message updates for now
	}

	message := update.Message
	if message.From == nil || message.Chat.ID == 0 {
		return fmt.Errorf("invalid message: missing from or chat")
	}

	// Handle different commands
	text := strings.TrimSpace(message.Text)
	if strings.HasPrefix(text, "/") {
		return h.handleCommand(ctx, message, text)
	}

	// Handle regular text messages
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
	// Check if user already exists
	chatIDStr := strconv.FormatInt(chatID, 10)
	var existingUser userModels.User
	err := h.db.Pool.QueryRow(ctx, "SELECT id, email, telegram_chat_id, subscription_tier, created_at, updated_at FROM users WHERE telegram_chat_id = $1", chatIDStr).Scan(
		&existingUser.ID, &existingUser.Email, &existingUser.TelegramChatID, &existingUser.SubscriptionTier, &existingUser.CreatedAt, &existingUser.UpdatedAt)
	if err == nil {
		// User already exists, send welcome back message
		return h.sendMessage(ctx, chatID, "Welcome back! 🎉\n\nYou're already registered and receiving alerts.\n\nUse /opportunities to see current arbitrage opportunities.")
	}

	if err != pgx.ErrNoRows {
		return fmt.Errorf("database error: %w", err)
	}

	// Create new user
	userID_str := fmt.Sprintf("usr_%d_%d", userID, time.Now().Unix())
	email := fmt.Sprintf("telegram_%d@celebrum.ai", userID)
	telegramChatID := &chatIDStr
	subscriptionTier := "free"
	now := time.Now()

	_, err = h.db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, telegram_chat_id, subscription_tier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		userID_str, email, telegramChatID, subscriptionTier, now, now)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

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
	// Call the arbitrage handler's underlying function directly
	minProfit := 1.0   // Minimum 1% profit
	limit := 5         // Limit to top 5 opportunities for Telegram
	symbolFilter := "" // No symbol filter

	opportunities, err := h.arbitrageHandler.FindArbitrageOpportunities(ctx, minProfit, limit, symbolFilter)
	if err != nil {
		log.Printf("Error fetching arbitrage opportunities: %v", err)
		return h.sendMessage(ctx, chatID, "❌ Error fetching arbitrage opportunities. Please try again later.")
	}

	// Format the message
	if len(opportunities) == 0 {
		msg := `📈 Current Arbitrage Opportunities:

🔍 No profitable opportunities found at the moment.

💡 Opportunities appear when there are price differences ≥1% between exchanges.

⚙️ Configure alerts: /settings
🎯 Upgrade for more features: /upgrade`
		return h.sendMessage(ctx, chatID, msg)
	}

	// Build opportunities message
	msg := "📈 Current Arbitrage Opportunities:\n\n"
	for i, opp := range opportunities {
		if i >= 5 { // Limit to top 5 for readability
			break
		}
		msg += fmt.Sprintf("💰 %s\n", opp.Symbol)
		msg += fmt.Sprintf("📊 Profit: %.2f%% (%.4f)\n", opp.ProfitPercent, opp.ProfitAmount)
		msg += fmt.Sprintf("🔻 Buy: %s @ %.6f\n", opp.BuyExchange, opp.BuyPrice)
		msg += fmt.Sprintf("🔺 Sell: %s @ %.6f\n", opp.SellExchange, opp.SellPrice)
		msg += "\n"
	}

	msg += "⚙️ Configure alerts: /settings\n"
	msg += "🎯 Upgrade for more features: /upgrade"

	return h.sendMessage(ctx, chatID, msg)
}

// handleSettingsCommand handles the /settings command
func (h *TelegramHandler) handleSettingsCommand(ctx context.Context, chatID, userID int64) error {
	msg := `⚙️ Alert Settings:

🔔 Notifications: ON
📊 Min Profit Threshold: 1.5%
⏰ Alert Frequency: Every 5 minutes
💰 Subscription: Free Tier

Settings management is being implemented.

For now, use:
/stop - Pause notifications
/resume - Resume notifications
/upgrade - Upgrade to premium`

	return h.sendMessage(ctx, chatID, msg)
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

Payment integration is being implemented.
Contact support for early access!`

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

// handleStopCommand handles the /stop command
func (h *TelegramHandler) handleStopCommand(ctx context.Context, chatID, userID int64) error {
	// TODO: Implement notification pause logic
	msg := `⏸️ Notifications Paused

You will no longer receive arbitrage alerts.

Use /resume to start receiving alerts again.`

	return h.sendMessage(ctx, chatID, msg)
}

// handleResumeCommand handles the /resume command
func (h *TelegramHandler) handleResumeCommand(ctx context.Context, chatID, userID int64) error {
	// TODO: Implement notification resume logic
	msg := `▶️ Notifications Resumed

You will now receive arbitrage alerts again.

Use /opportunities to see current opportunities.`

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
	if h.bot == nil {
		log.Printf("Telegram bot is not initialized, cannot send message to chat %d", chatID)
		return fmt.Errorf("telegram bot not available")
	}

	_, err := h.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   text,
	})
	if err != nil {
		log.Printf("Failed to send message to chat %d: %v", chatID, err)
		return err
	}
	return nil
}
