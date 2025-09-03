package main

import (
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
	// Получаем токен бота из переменных окружения
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
	}

	// Создаем экземпляр бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Ошибка создания бота:", err)
	}

	bot.Debug = false
	log.Printf("Бот запущен: %s", bot.Self.UserName)

	// Настройка получения обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Обработка сообщений
	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}

// handleMessage обрабатывает текстовые сообщения
func handleMessage(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID
	text := message.Text

	switch text {
	case "/start":
		handleStart(bot, chatID)
	case "/help":
		handleHelp(bot, chatID)
	case "/login":
		handleLogin(bot, chatID)
	case "/diary":
		handleDiary(bot, chatID)
	case "/periods":
		handlePeriods(bot, chatID)
	default:
		handleDefault(bot, chatID, text)
	}
}

// handleStart обрабатывает команду /start
func handleStart(bot *tgbotapi.BotAPI, chatID int64) {
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 Дневник", "diary"),
			tgbotapi.NewInlineKeyboardButtonData("📅 Периоды", "periods"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔐 Войти", "login"),
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "help"),
		),
	)

	msg := tgbotapi.NewMessage(chatID, "👋 Добро пожаловать в школьный электронный дневник!\n\nВыберите действие:")
	msg.ReplyMarkup = keyboard
	bot.Send(msg)
}

// handleHelp обрабатывает команду /help
func handleHelp(bot *tgbotapi.BotAPI, chatID int64) {
	helpText := `🤖 *Школьный электронный дневник*

*Доступные команды:*
/start - Главное меню
/login - Авторизация в системе
/diary - Просмотр дневника
/periods - Учебные периоды
/help - Эта справка

*Как пользоваться:*
1. Авторизуйтесь с помощью /login
2. Используйте команды для просмотра информации
3. Выбирайте недели и периоды для просмотра оценок`

	msg := tgbotapi.NewMessage(chatID, helpText)
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// handleLogin обрабатывает авторизацию
func handleLogin(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🔐 Для авторизации отправьте данные в формате:\n`логин пароль`\n\nПример: `Daniil_Melnik mypassword`")
	msg.ParseMode = "Markdown"
	bot.Send(msg)
}

// handleDiary обрабатывает просмотр дневника
func handleDiary(bot *tgbotapi.BotAPI, chatID int64) {
	// TODO: Проверить авторизацию пользователя
	// TODO: Показать выбор недели
	msg := tgbotapi.NewMessage(chatID, "📚 Дневник\n\n⚠️ Сначала необходимо авторизоваться через /login")
	bot.Send(msg)
}

// handlePeriods обрабатывает просмотр периодов
func handlePeriods(bot *tgbotapi.BotAPI, chatID int64) {
	// TODO: Проверить авторизацию пользователя
	// TODO: Показать периоды обучения
	msg := tgbotapi.NewMessage(chatID, "📅 Учебные периоды\n\n⚠️ Сначала необходимо авторизоваться через /login")
	bot.Send(msg)
}

// handleDefault обрабатывает остальные сообщения
func handleDefault(bot *tgbotapi.BotAPI, chatID int64, text string) {
	// TODO: Обработка данных авторизации
	msg := tgbotapi.NewMessage(chatID, "❓ Неизвестная команда. Используйте /help для получения справки.")
	bot.Send(msg)
}

// handleCallbackQuery обрабатывает нажатия на кнопки
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID
	data := query.Data

	// Отвечаем на callback query
	callback := tgbotapi.NewCallback(query.ID, "")
	bot.Request(callback)

	switch data {
	case "diary":
		handleDiary(bot, chatID)
	case "periods":
		handlePeriods(bot, chatID)
	case "login":
		handleLogin(bot, chatID)
	case "help":
		handleHelp(bot, chatID)
	default:
		// TODO: Обработка выбора недель и периодов
		msg := tgbotapi.NewMessage(chatID, "🔄 Обрабатываем запрос...")
		bot.Send(msg)
	}
}