package main

import (
        "log"
        "os"

        "school-diary-bot/internal/bot"
        tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func main() {
        // Получаем токен бота из переменных окружения
        botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
        if botToken == "" {
                log.Fatal("TELEGRAM_BOT_TOKEN не установлен")
        }

        // Создаем экземпляр бота
        diaryBot, err := bot.NewBot(botToken)
        if err != nil {
                log.Fatal("Ошибка создания бота:", err)
        }

        log.Printf("Бот запущен: %s", diaryBot.API.Self.UserName)

        // Настройка получения обновлений
        u := tgbotapi.NewUpdate(0)
        u.Timeout = 60

        updates := diaryBot.API.GetUpdatesChan(u)

        // Обработка сообщений
        for update := range updates {
                if update.Message != nil {
                        if err := diaryBot.HandleMessage(update.Message); err != nil {
                                log.Printf("Ошибка обработки сообщения: %v", err)
                        }
                } else if update.CallbackQuery != nil {
                        if err := diaryBot.HandleCallback(update.CallbackQuery); err != nil {
                                log.Printf("Ошибка обработки callback: %v", err)
                        }
                }
        }
}

