package bot

import (
	"context"
	"fmt"
	"log"
	"strings"

	tgbot "github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"

	"rules-telegram-bot/internal/config"
	"rules-telegram-bot/internal/i18n"
	"rules-telegram-bot/internal/rules"
)

// Admin commands (message text, case-insensitive, trimmed).
const (
	adminPanelText  = "/rules_admin"
	globalToggleTxt = "/rules_toggle"
	adminTitleCmd   = "/admin_title"
	rulesWord       = "правила"
	rulesWordEn     = "rules"
)

// Handler owns bot dependencies and routing.
type Handler struct {
	bot      *tgbot.Bot
	cfg      *config.Config
	store    *rules.Store
	sessions *Sessions
}

// New builds the Telegram bot and registers all handlers.
//
// Routing uses the library's RegisterHandler functions: the first matching
// handler wins, the default handler fires only when nothing matches.
// Registration order therefore matters: the pending-input matcher goes first
// so the next message from an editing admin is consumed as new item text.
func New(cfg *config.Config, store *rules.Store) (*tgbot.Bot, *Handler, error) {
	h := &Handler{cfg: cfg, store: store, sessions: NewSessions()}
	b, err := tgbot.New(cfg.BotToken,
		tgbot.WithDefaultHandler(h.onDefault),
		tgbot.WithMiddlewares(h.chatGate),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("create bot: %w", err)
	}
	h.bot = b
	h.registerHandlers(b)
	return b, h, nil
}

// Run starts long polling until ctx is cancelled.
func Run(ctx context.Context, b *tgbot.Bot) {
	b.Start(ctx)
}

func (h *Handler) registerHandlers(b *tgbot.Bot) {
	// Message text handlers (first match wins — order matters).
	b.RegisterHandlerMatchFunc(h.matchPendingInput, h.handlePendingInput)
	b.RegisterHandlerMatchFunc(h.matchAdminPanel, h.handleAdminPanel, h.requireAdmin)
	b.RegisterHandlerMatchFunc(h.matchGlobalToggle, h.handleGlobalToggle, h.requireAdmin)
	b.RegisterHandlerMatchFunc(h.matchAdminTitle, h.handleAdminTitle, h.requireAdmin)
	b.RegisterHandlerMatchFunc(h.matchRules, h.handleRules)

	// User callbacks.
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, "noop", tgbot.MatchTypeExact, h.handleNoop)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPUserChapter, tgbot.MatchTypePrefix, h.handleUserChapter)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPUserPage, tgbot.MatchTypePrefix, h.handleUserPage)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPUserBack, tgbot.MatchTypePrefix, h.handleUserBack)

	// Admin callbacks. Prefixes are disjoint by construction
	// ("a:del:" never prefixes "a:delok:", etc. — see TestCallbackPrefixesDisjoint).
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminBack, tgbot.MatchTypePrefix, h.handleAdminBack, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminOpen, tgbot.MatchTypePrefix, h.handleAdminOpenChapter, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminPage, tgbot.MatchTypePrefix, h.handleAdminPage, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminVis, tgbot.MatchTypePrefix, h.handleAdminToggleChapter, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminDel, tgbot.MatchTypePrefix, h.handleAdminAskDeleteChapter, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminDelOK, tgbot.MatchTypePrefix, h.handleAdminConfirmDeleteChapter, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminItems, tgbot.MatchTypePrefix, h.handleAdminItemsPage, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminEditItem, tgbot.MatchTypePrefix, h.handleAdminEditItem, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminVisItem, tgbot.MatchTypePrefix, h.handleAdminToggleItem, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminDelItem, tgbot.MatchTypePrefix, h.handleAdminAskDeleteItem, h.requireAdmin)
	b.RegisterHandler(tgbot.HandlerTypeCallbackQueryData, i18n.CbPAdminDelItemOK, tgbot.MatchTypePrefix, h.handleAdminConfirmDeleteItem, h.requireAdmin)
}

// --- middlewares ---

// chatGate is a global middleware: only the target chat is served.
func (h *Handler) chatGate(next tgbot.HandlerFunc) tgbot.HandlerFunc {
	return func(ctx context.Context, b *tgbot.Bot, u *models.Update) {
		if chat := h.updateChatID(u); chat == 0 || chat != h.cfg.TargetChatID {
			return
		}
		next(ctx, b, u)
	}
}

// requireAdmin is a per-handler middleware for admin-only handlers.
func (h *Handler) requireAdmin(next tgbot.HandlerFunc) tgbot.HandlerFunc {
	return func(ctx context.Context, b *tgbot.Bot, u *models.Update) {
		if !h.cfg.IsAdmin(h.updateUserID(u)) {
			if u.CallbackQuery != nil {
				h.answerCb(ctx, u.CallbackQuery.ID, i18n.AccessDeniedAck())
			} else if chat := h.updateChatID(u); chat != 0 {
				h.sendText(ctx, chat, i18n.AccessDenied().Text)
			}
			return
		}
		next(ctx, b, u)
	}
}

// --- update field helpers ---

func (h *Handler) updateChatID(u *models.Update) int64 {
	if u.Message != nil {
		return u.Message.Chat.ID
	}
	if u.CallbackQuery != nil && u.CallbackQuery.Message.Message != nil {
		return u.CallbackQuery.Message.Message.Chat.ID
	}
	return 0
}

func (h *Handler) updateUserID(u *models.Update) int64 {
	if u.CallbackQuery != nil {
		return u.CallbackQuery.From.ID
	}
	if u.Message != nil && u.Message.From != nil {
		return u.Message.From.ID
	}
	return 0
}

// ownerOf returns the message author who will own the opened menu.
// Menus without an attributable author are not opened at all.
func ownerOf(u *models.Update) (int64, bool) {
	if u.Message == nil || u.Message.From == nil || u.Message.From.ID == 0 {
		return 0, false
	}
	return u.Message.From.ID, true
}

// messageText returns trimmed message text, if any.
func messageText(u *models.Update) (string, bool) {
	if u.Message == nil {
		return "", false
	}
	t := strings.TrimSpace(u.Message.Text)
	if t == "" {
		return "", false
	}
	return t, true
}

// --- message match funcs ---

func (h *Handler) matchPendingInput(u *models.Update) bool {
	if _, ok := messageText(u); !ok {
		return false
	}
	id, ok := ownerOf(u)
	return ok && h.sessions.Has(id)
}

func (h *Handler) matchAdminPanel(u *models.Update) bool {
	if _, ok := ownerOf(u); !ok {
		return false
	}
	t, ok := messageText(u)
	return ok && strings.ToLower(t) == adminPanelText
}

func (h *Handler) matchGlobalToggle(u *models.Update) bool {
	if _, ok := ownerOf(u); !ok {
		return false
	}
	t, ok := messageText(u)
	return ok && strings.ToLower(t) == globalToggleTxt
}

// matchAdminTitle matches "/admin_title" and "/admin_title <new text>".
func (h *Handler) matchAdminTitle(u *models.Update) bool {
	if _, ok := ownerOf(u); !ok {
		return false
	}
	t, ok := messageText(u)
	if !ok {
		return false
	}
	lower := strings.ToLower(t)
	return lower == adminTitleCmd || strings.HasPrefix(lower, adminTitleCmd+" ")
}

func (h *Handler) matchRules(u *models.Update) bool {
	if _, ok := ownerOf(u); !ok {
		return false
	}
	t, ok := messageText(u)
	if !ok {
		return false
	}
	lower := strings.ToLower(t)
	switch lower {
	case rulesWord, rulesWordEn, "/rules", "/start":
		return true
	}
	return strings.HasPrefix(lower, "/rules@") ||
		strings.HasPrefix(lower, "/start@") ||
		strings.HasPrefix(lower, "/rules ") ||
		strings.HasPrefix(lower, "/start ")
}

// --- message handlers ---

// handlePendingInput consumes the next text message as new item content.
// Registered first so it wins over command handlers while an edit is pending.
func (h *Handler) handlePendingInput(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	msg := u.Message
	chatID := msg.Chat.ID
	userID, ok := ownerOf(u)
	if !ok {
		return
	}
	text := strings.TrimSpace(msg.Text)

	pend, ok := h.sessions.Take(userID)
	if !ok {
		return
	}
	if !h.cfg.IsAdmin(userID) {
		h.sendText(ctx, chatID, i18n.AccessDenied().Text)
		return
	}
	if lower := strings.ToLower(text); lower == "отмена" || lower == "/cancel" {
		h.sendText(ctx, chatID, i18n.EditCancelled().Text)
		return
	}
	if pend.Kind == PendingTitle {
		if err := h.store.SetTitle(text); err != nil {
			h.sendText(ctx, chatID, i18n.SaveFailed(err).Text)
			return
		}
		h.sendText(ctx, chatID, i18n.TitleUpdated().Text)
		h.renderAdminChapters(ctx, chatID, 0, 0, true, userID)
		return
	}
	if err := h.store.SetItemText(pend.Chapter, pend.Item, text); err != nil {
		h.sendText(ctx, chatID, i18n.SaveFailed(err).Text)
		return
	}
	h.sendText(ctx, chatID, i18n.ItemUpdated(pend.Chapter+1, pend.Item+1).Text)
	h.renderAdminItems(ctx, chatID, msg.ID, pend.Chapter, 0, true, userID)
}

func (h *Handler) handleAdminPanel(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	owner, ok := ownerOf(u)
	if !ok {
		return
	}
	h.renderAdminChapters(ctx, u.Message.Chat.ID, 0, 0, true, owner)
}

func (h *Handler) handleGlobalToggle(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID := u.Message.Chat.ID
	now, err := h.store.ToggleGlobalVisible()
	if err != nil {
		h.sendText(ctx, chatID, i18n.ToggleFailed(err).Text)
		return
	}
	h.sendText(ctx, chatID, i18n.GlobalVisibility(now).Text)
}

// handleAdminTitle sets the top-level title: "/admin_title <text>" applies
// immediately, bare "/admin_title" asks for the text via pending input.
func (h *Handler) handleAdminTitle(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	chatID := u.Message.Chat.ID
	userID, ok := ownerOf(u)
	if !ok {
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	rest := ""
	if len(text) > len(adminTitleCmd) {
		rest = strings.TrimSpace(text[len(adminTitleCmd):])
	}
	if rest == "" {
		h.sessions.Set(userID, Pending{Kind: PendingTitle, ChatID: chatID})
		m := i18n.EditTitlePrompt(h.store.Snapshot().Title)
		h.sendText(ctx, chatID, m.Text)
		return
	}
	if err := h.store.SetTitle(rest); err != nil {
		h.sendText(ctx, chatID, i18n.SaveFailed(err).Text)
		return
	}
	h.sendText(ctx, chatID, i18n.TitleUpdated().Text)
}

func (h *Handler) handleRules(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	owner, ok := ownerOf(u)
	if !ok {
		return
	}
	h.renderUserChapters(ctx, u.Message.Chat.ID, 0, true, owner)
}

// onDefault fires only when no handler matches: dismiss spinner, ignore chatter.
func (h *Handler) onDefault(ctx context.Context, b *tgbot.Bot, u *models.Update) {
	if u.CallbackQuery != nil {
		h.answerCb(ctx, u.CallbackQuery.ID, "")
	}
}

// --- shared send helpers ---

func (h *Handler) sendText(ctx context.Context, chatID int64, text string) {
	_, err := h.bot.SendMessage(ctx, &tgbot.SendMessageParams{ChatID: chatID, Text: text})
	if err != nil {
		log.Printf("send message: %v", err)
	}
}

func (h *Handler) answerCb(ctx context.Context, cbID, text string) {
	_, err := h.bot.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{
		CallbackQueryID: cbID,
		Text:            text,
	})
	if err != nil {
		log.Printf("answer callback: %v", err)
	}
}
