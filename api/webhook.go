package handler

import (
        "encoding/json"
        "io"
        "log"
        "net/http"
        "os"

        "school-diary-bot/bot"
        "school-diary-bot/bot/eljur"
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
        if botToken == "" {
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