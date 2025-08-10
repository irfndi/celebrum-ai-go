package services

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	userModels "github.com/irfndi/celebrum-ai-go/internal/models"
)

type NotificationService struct {
	db  *database.PostgresDB
	bot *bot.Bot
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

func NewNotificationService(db *database.PostgresDB, telegramBotToken string) *NotificationService {
	// Initialize Telegram bot if token is provided
	var telegramBot *bot.Bot
	if telegramBotToken != "" {
		telegramBot, _ = bot.New(telegramBotToken)
	}

	return &NotificationService{
		db:  db,
		bot: telegramBot,
	}
}

// NotifyArbitrageOpportunities sends notifications about arbitrage opportunities to eligible users
func (ns *NotificationService) NotifyArbitrageOpportunities(ctx context.Context, opportunities []ArbitrageOpportunity) error {
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

// getEligibleUsers returns all users who should receive arbitrage alerts
func (ns *NotificationService) getEligibleUsers(ctx context.Context) ([]userModels.User, error) {
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

	return users, nil
}

// sendArbitrageAlert sends a formatted arbitrage alert to a specific user
func (ns *NotificationService) sendArbitrageAlert(ctx context.Context, user userModels.User, opportunities []ArbitrageOpportunity) error {
	if ns.bot == nil {
		return fmt.Errorf("telegram bot not initialized")
	}

	chatID, err := strconv.ParseInt(*user.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Format the alert message
	message := ns.formatArbitrageMessage(opportunities)

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

// CheckUserNotificationPreferences checks if a user wants to receive arbitrage notifications
func (ns *NotificationService) CheckUserNotificationPreferences(ctx context.Context, userID string) (bool, error) {
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

	return count == 0, nil // Return true if no disabled alerts found
}
