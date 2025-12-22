package main

import (
	"context"
	"fmt"
	"time"

	"github.com/irfandi/celebrum-ai-go/internal/config"
	"github.com/irfandi/celebrum-ai-go/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func runSeeder() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := database.NewPostgresConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Seed test user
	email := "test@example.com"
	username := "testuser"
	password := "password123"

	// Check if user exists
	var exists bool
	err = db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)", email).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check existing user: %w", err)
	}

	if !exists {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		_, err = db.Pool.Exec(ctx, `
            INSERT INTO users (email, username, password_hash, role, created_at, updated_at)
            VALUES ($1, $2, $3, 'user', $4, $4)
        `, email, username, string(hashedPassword), time.Now())
		if err != nil {
			return fmt.Errorf("failed to insert user: %w", err)
		}
		fmt.Println("✅ Seeded test user: test@example.com / password123")
	} else {
		fmt.Println("ℹ️  Test user already exists")
	}

	return nil
}
