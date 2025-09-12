package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-telegram/bot"
	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🔧 Validating Telegram Bot Configuration...")

	// Load .env file
	if err := godotenv.Load(); err != nil {
		fmt.Printf("⚠️  Warning: Could not load .env file: %v\n", err)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Check if Telegram bot token is configured
	if cfg.Telegram.BotToken == "" {
		fmt.Println("❌ TELEGRAM_BOT_TOKEN is not configured")
		os.Exit(1)
	}

	fmt.Printf("✅ TELEGRAM_BOT_TOKEN is configured (length: %d)\n", len(cfg.Telegram.BotToken))

	// Try to create bot instance
	b, err := bot.New(cfg.Telegram.BotToken)
	if err != nil {
		fmt.Printf("❌ Failed to create Telegram bot: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Telegram bot created successfully")

	// Check webhook URL
	if cfg.Telegram.WebhookURL == "" {
		fmt.Println("⚠️  TELEGRAM_WEBHOOK_URL is not configured")
	} else {
		fmt.Printf("✅ TELEGRAM_WEBHOOK_URL is configured: %s\n", cfg.Telegram.WebhookURL)
	}

	// Try to get bot info (this makes an actual API call)
	fmt.Println("🔍 Testing bot API connection...")
	ctx := context.Background()
	botInfo, err := b.GetMe(ctx)
	if err != nil {
		fmt.Printf("❌ Failed to get bot info: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Bot API connection successful!\n")
	fmt.Printf("   Bot Name: %s\n", botInfo.FirstName)
	fmt.Printf("   Bot Username: @%s\n", botInfo.Username)
	fmt.Printf("   Bot ID: %d\n", botInfo.ID)

	fmt.Println("\n🎉 All Telegram bot configuration checks passed!")
}
