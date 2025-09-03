package handler

import (
        "encoding/json"
        "io"
        "log"
        "net/http"
        "os"

        "school-diary-bot/internal/bot"
        "school-diary-bot/internal/eljur"
        tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Webhook обрабатывает входящие webhook от Telegram
func Webhook(w http.ResponseWriter, r *http.Request) {
        // Проверяем метод запроса
        if r.Method != "POST" {
                http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
                return
        }

        // Проверяем наличие необходимых переменных окружения
        if err := eljur.ValidateConfig(); err != nil {
                log.Printf("Ошибка конфигурации: %v", err)
                http.Error(w, "Configuration error", http.StatusInternalServerError)
                return
        }

        // Получаем токен бота из переменных окружения
        botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
        diaryBot, err := bot.NewBot(botToken)
                log.Println("TELEGRAM_BOT_TOKEN не установлен")
                http.Error(w, "Bot token not configured", http.StatusInternalServerError)
                return
        }

        // Создаем экземпляр бота
        diaryBot, err := bot.NewBot(botToken)
        if err != nil {
                log.Printf("Ошибка создания бота: %v", err)
                http.Error(w, "Failed to create bot", http.StatusInternalServerError)
                return
        }

        // Читаем тело запроса
        body, err := io.ReadAll(r.Body)
        if err != nil {
                log.Printf("Ошибка чтения тела запроса: %v", err)
                http.Error(w, "Failed to read request body", http.StatusBadRequest)
                return
        }

        // Парсим JSON
        var update tgbotapi.Update
        if err := json.Unmarshal(body, &update); err != nil {
                log.Printf("Ошибка парсинга JSON: %v", err)
                http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
                return
        }

        // Обрабатываем обновление
        if update.Message != nil {
                if err := diaryBot.HandleMessage(update.Message); err != nil {
                        log.Printf("Ошибка обработки сообщения: %v", err)
                }
        } else if update.CallbackQuery != nil {
                if err := diaryBot.HandleCallback(update.CallbackQuery); err != nil {
                        log.Printf("Ошибка обработки callback: %v", err)
                }
        }

        // Отвечаем успехом
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
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
        if _, err := bot.Send(msg); err != nil {
                log.Printf("Ошибка отправки сообщения: %v", err)
        }
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
        if _, err := bot.Send(msg); err != nil {
                log.Printf("Ошибка отправки сообщения: %v", err)
        }
}

// handleLogin обрабатывает авторизацию
func handleLogin(bot *tgbotapi.BotAPI, chatID int64) {
        msg := tgbotapi.NewMessage(chatID, "🔐 Для авторизации отправьте данные в формате:\n`логин пароль`\n\nПример: `Daniil_Melnik mypassword`")
        msg.ParseMode = "Markdown"
        if _, err := bot.Send(msg); err != nil {
                log.Printf("Ошибка отправки сообщения: %v", err)
        }
}

// handleDiary обрабатывает просмотр дневника
func handleDiary(bot *tgbotapi.BotAPI, chatID int64) {
        // TODO: Проверить авторизацию пользователя
        // TODO: Показать выбор недели
        msg := tgbotapi.NewMessage(chatID, "📚 Дневник\n\n⚠️ Сначала необходимо авторизоваться через /login")
        if _, err := bot.Send(msg); err != nil {
                log.Printf("Ошибка отправки сообщения: %v", err)
        }
}

// handlePeriods обрабатывает просмотр периодов
func handlePeriods(bot *tgbotapi.BotAPI, chatID int64) {
        // TODO: Проверить авторизацию пользователя
        // TODO: Показать периоды обучения
        msg := tgbotapi.NewMessage(chatID, "📅 Учебные периоды\n\n⚠️ Сначала необходимо авторизоваться через /login")
        if _, err := bot.Send(msg); err != nil {
                log.Printf("Ошибка отправки сообщения: %v", err)
        }
}

// handleDefault обрабатывает остальные сообщения
func handleDefault(bot *tgbotapi.BotAPI, chatID int64, text string) {
        // TODO: Обработка данных авторизации
        msg := tgbotapi.NewMessage(chatID, "❓ Неизвестная команда. Используйте /help для получения справки.")
        if _, err := bot.Send(msg); err != nil {
                log.Printf("Ошибка отправки сообщения: %v", err)
        }
}

// handleCallbackQuery обрабатывает нажатия на кнопки
func handleCallbackQuery(bot *tgbotapi.BotAPI, query *tgbotapi.CallbackQuery) {
        chatID := query.Message.Chat.ID
        data := query.Data

        // Отвечаем на callback query
        callback := tgbotapi.NewCallback(query.ID, "")
        if _, err := bot.Request(callback); err != nil {
                log.Printf("Ошибка ответа на callback: %v", err)
        }

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
                if _, err := bot.Send(msg); err != nil {
                        log.Printf("Ошибка отправки сообщения: %v", err)
                }
        }
}