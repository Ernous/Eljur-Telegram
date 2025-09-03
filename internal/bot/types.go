package bot

import (
	"school-diary-bot/internal/eljur"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// UserState представляет состояние пользователя
type UserState struct {
	ChatID       int64
	State        string // "idle", "auth_waiting", "message_compose", "week_select", "gemini_api_setup", "gemini_chat"
	AuthStep     int    // 0 - не авторизован, 1 - логин, 2 - пароль
	TempLogin    string
	TempPassword string
	TempRecipient string // ID выбранного получателя
	Client       *eljur.Client
	CurrentWeek  string
	CurrentPeriod string
	GeminiAPIKey string // API ключ для Gemini
	GeminiModel  string // Выбранная модель Gemini
	GeminiContext string // Контекст для Gemini (домашнее задание и т.д.)
}

// Bot представляет основную структуру бота
type Bot struct {
	API   *tgbotapi.BotAPI
	Users  map[int64]*UserState
}

// NewBot создает новый экземпляр бота
func NewBot(token string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	api.Debug = false

	return &Bot{
		API:   api,
		Users: make(map[int64]*UserState),
	}, nil
}

// GetUserState получает или создает состояние пользователя
func (b *Bot) GetUserState(chatID int64) *UserState {
	if user, exists := b.Users[chatID]; exists {
		return user
	}

	user := &UserState{
		ChatID: chatID,
		State:  "idle",
		Client: eljur.NewClient(),
	}
	b.Users[chatID] = user
	return user
}

// SendMessage отправляет сообщение пользователю
func (b *Bot) SendMessage(chatID int64, text string, keyboard interface{}) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML" // Используем HTML вместо Markdown для лучшей совместимости

	if keyboard != nil {
		if kb, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
			msg.ReplyMarkup = kb
		}
	}

	_, err := b.API.Send(msg)
	return err
}

// EditMessage редактирует сообщение
func (b *Bot) EditMessage(chatID int64, messageID int, text string, keyboard interface{}) error {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "HTML" // Используем HTML вместо Markdown для лучшей совместимости

	if keyboard != nil {
		if kb, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
			msg.ReplyMarkup = kb
		}
	}

	_, err := b.API.Send(msg)
	return err
}

// AnswerCallback отвечает на callback query
func (b *Bot) AnswerCallback(callbackID, text string) {
	callback := tgbotapi.NewCallback(callbackID, text)
	b.API.Request(callback)
}