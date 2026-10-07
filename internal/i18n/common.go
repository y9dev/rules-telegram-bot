package i18n

import "fmt"

// AccessDenied is the reply for non-admins hitting admin handlers.
func AccessDenied() Message {
	return Message{Text: "⛔ Нет доступа."}
}

// AccessDeniedAck is the callback answer for non-admins.
func AccessDeniedAck() string {
	return "⛔ Нет доступа"
}

// EditCancelled confirms the pending item edit was dropped.
func EditCancelled() Message {
	return Message{Text: "Редактирование отменено."}
}

// SaveFailed reports a failed item-text update.
func SaveFailed(err error) Message {
	return Message{Text: "Не удалось сохранить: " + err.Error()}
}

// ToggleFailed reports a failed global-visibility toggle.
func ToggleFailed(err error) Message {
	return Message{Text: "Не удалось переключить: " + err.Error()}
}

// ItemUpdated confirms the new item text was stored.
func ItemUpdated(chapterNum, itemNum int) Message {
	return Message{Text: fmt.Sprintf("✅ Пункт %d.%d обновлён.", chapterNum, itemNum)}
}

// GlobalVisibility confirms the new state of the global toggle.
func GlobalVisibility(now bool) Message {
	if now {
		return Message{Text: "✅ Правила теперь ВИДИМЫ для всех."}
	}
	return Message{Text: "🚫 Правила теперь СКРЫТЫ для всех."}
}

// VisibilityAck is the callback answer after a visibility toggle.
func VisibilityAck(visible bool) string {
	if visible {
		return "✅ Видимо"
	}
	return "🚫 Скрыто"
}

// NotYourMenuAck answers presses on someone else's menu.
func NotYourMenuAck() string {
	return "⛔ Это меню вызвал другой пользователь"
}
