package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"rules-telegram-bot/internal/bot"
	"rules-telegram-bot/internal/config"
	"rules-telegram-bot/internal/rules"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	store, err := rules.Load(cfg.RulesPath)
	if err != nil {
		log.Fatalf("rules: %v", err)
	}
	b, _, err := bot.New(cfg, store)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}
	log.Printf("bot started for chat %d (rules: %s)", cfg.TargetChatID, cfg.RulesPath)
	bot.Run(ctx, b)
}
