package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime settings. Secrets come from env only.
type Config struct {
	BotToken     string
	TargetChatID int64
	AdminIDs     map[int64]struct{}
	RulesPath    string
}

// IsAdmin reports whether userID is in ADMIN_IDS.
func (c *Config) IsAdmin(userID int64) bool {
	_, ok := c.AdminIDs[userID]
	return ok
}

// Load reads configuration from environment variables.
// Required: BOT_TOKEN, TARGET_CHAT_ID. Optional: ADMIN_IDS, RULES_PATH.
func Load() (*Config, error) {
	_ = godotenv.Load()

	token := strings.TrimSpace(os.Getenv("BOT_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("BOT_TOKEN is required")
	}
	chatRaw := strings.TrimSpace(os.Getenv("TARGET_CHAT_ID"))
	if chatRaw == "" {
		return nil, fmt.Errorf("TARGET_CHAT_ID is required")
	}
	chatID, err := strconv.ParseInt(chatRaw, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("TARGET_CHAT_ID must be integer: %w", err)
	}
	admins := make(map[int64]struct{})
	for _, part := range strings.Split(os.Getenv("ADMIN_IDS"), ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("ADMIN_IDS entry %q must be integer: %w", part, err)
		}
		admins[id] = struct{}{}
	}
	rulesPath := strings.TrimSpace(os.Getenv("RULES_PATH"))
	if rulesPath == "" {
		rulesPath = "rules.json"
	}
	return &Config{
		BotToken:     token,
		TargetChatID: chatID,
		AdminIDs:     admins,
		RulesPath:    rulesPath,
	}, nil
}
