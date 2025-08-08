package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/irfndi/celebrum-ai-go/internal/database"
	"github.com/irfndi/celebrum-ai-go/internal/models"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	db *database.PostgresDB
}

type RegisterRequest struct {
	Email          string  `json:"email" binding:"required,email"`
	Password       string  `json:"password" binding:"required,min=8"`
	TelegramChatID *string `json:"telegram_chat_id,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID               string    `json:"id"`
	Email            string    `json:"email"`
	TelegramChatID   *string   `json:"telegram_chat_id,omitempty"`
	SubscriptionTier string    `json:"subscription_tier"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type LoginResponse struct {
	User  UserResponse `json:"user"`
	Token string       `json:"token"`
}

type UpdateProfileRequest struct {
	TelegramChatID *string `json:"telegram_chat_id,omitempty"`
}

func NewUserHandler(db *database.PostgresDB) *UserHandler {
	return &UserHandler{
		db: db,
	}
}

// RegisterUser handles user registration
func (h *UserHandler) RegisterUser(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if user already exists
	exists, err := h.userExists(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user existence"})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "User with this email already exists"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Create user
	userID := uuid.New().String()
	query := `
		INSERT INTO users (id, email, password_hash, telegram_chat_id, subscription_tier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	_, err = h.db.Pool.Exec(c.Request.Context(), query,
		userID, req.Email, string(hashedPassword), req.TelegramChatID, "free", now, now)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Return user response (without password)
	userResponse := UserResponse{
		ID:               userID,
		Email:            req.Email,
		TelegramChatID:   req.TelegramChatID,
		SubscriptionTier: "free",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	c.JSON(http.StatusCreated, gin.H{"user": userResponse})
}

// LoginUser handles user authentication
func (h *UserHandler) LoginUser(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user from database
	user, err := h.getUserByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Generate JWT token (simplified - in production use proper JWT library)
	token := h.generateSimpleToken(user.ID)

	userResponse := UserResponse{
		ID:               user.ID,
		Email:            user.Email,
		TelegramChatID:   user.TelegramChatID,
		SubscriptionTier: user.SubscriptionTier,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
	}

	response := LoginResponse{
		User:  userResponse,
		Token: token,
	}

	c.JSON(http.StatusOK, response)
}

// GetUserProfile returns the current user's profile
func (h *UserHandler) GetUserProfile(c *gin.Context) {
	// In a real implementation, you would extract user ID from JWT token
	// For now, we'll use a query parameter or header
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	user, err := h.getUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	userResponse := UserResponse{
		ID:               user.ID,
		Email:            user.Email,
		TelegramChatID:   user.TelegramChatID,
		SubscriptionTier: user.SubscriptionTier,
		CreatedAt:        user.CreatedAt,
		UpdatedAt:        user.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"user": userResponse})
}

// UpdateUserProfile updates user profile information
func (h *UserHandler) UpdateUserProfile(c *gin.Context) {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = c.Query("user_id")
	}

	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID required"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update user profile
	query := `
		UPDATE users 
		SET telegram_chat_id = $2, updated_at = $3
		WHERE id = $1
	`

	_, err := h.db.Pool.Exec(c.Request.Context(), query,
		userID, req.TelegramChatID, time.Now())

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Return updated user
	updatedUser, err := h.getUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated user"})
		return
	}

	userResponse := UserResponse{
		ID:               updatedUser.ID,
		Email:            updatedUser.Email,
		TelegramChatID:   updatedUser.TelegramChatID,
		SubscriptionTier: updatedUser.SubscriptionTier,
		CreatedAt:        updatedUser.CreatedAt,
		UpdatedAt:        updatedUser.UpdatedAt,
	}

	c.JSON(http.StatusOK, gin.H{"user": userResponse})
}

// Helper functions

func (h *UserHandler) userExists(ctx context.Context, email string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM users WHERE email = $1"
	err := h.db.Pool.QueryRow(ctx, query, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (h *UserHandler) getUserByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, telegram_chat_id, 
		       subscription_tier, created_at, updated_at
		FROM users WHERE email = $1
	`

	err := h.db.Pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.TelegramChatID,
		&user.SubscriptionTier, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (h *UserHandler) getUserByID(ctx context.Context, userID string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, telegram_chat_id, 
		       subscription_tier, created_at, updated_at
		FROM users WHERE id = $1
	`

	err := h.db.Pool.QueryRow(ctx, query, userID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.TelegramChatID,
		&user.SubscriptionTier, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// generateSimpleToken creates a simple token (in production, use proper JWT)
func (h *UserHandler) generateSimpleToken(userID string) string {
	// This is a simplified token generation
	// In production, use a proper JWT library like github.com/golang-jwt/jwt
	return fmt.Sprintf("token_%s_%d", userID, time.Now().Unix())
}

// GetUserByTelegramChatID retrieves user by Telegram chat ID (for Telegram bot)
func (h *UserHandler) GetUserByTelegramChatID(ctx context.Context, chatID string) (*models.User, error) {
	var user models.User
	query := `
		SELECT id, email, password_hash, telegram_chat_id, 
		       subscription_tier, created_at, updated_at
		FROM users WHERE telegram_chat_id = $1
	`

	err := h.db.Pool.QueryRow(ctx, query, chatID).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.TelegramChatID,
		&user.SubscriptionTier, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

// CreateTelegramUser creates a new user from Telegram registration
func (h *UserHandler) CreateTelegramUser(ctx context.Context, chatID string, username string) (*models.User, error) {
	userID := uuid.New().String()
	now := time.Now()

	// Create a temporary email based on Telegram username or chat ID
	email := fmt.Sprintf("telegram_%s@celebrum.ai", chatID)
	if username != "" {
		email = fmt.Sprintf("%s@telegram.celebrum.ai", username)
	}

	query := `
		INSERT INTO users (id, email, telegram_chat_id, subscription_tier, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, telegram_chat_id, subscription_tier, created_at, updated_at
	`

	var user models.User
	err := h.db.Pool.QueryRow(ctx, query,
		userID, email, chatID, "free", now, now).Scan(
		&user.ID, &user.Email, &user.TelegramChatID,
		&user.SubscriptionTier, &user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}
