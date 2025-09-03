package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"school-diary-bot/bot"
	"school-diary-bot/bot/eljur"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// validateWebhookSignature проверяет подпись webhook от Telegram
func validateWebhookSignature(body []byte, signature string, secretToken string) bool {
	if secretToken == "" {
		// Если секретный токен не настроен, пропускаем валидацию
		return true
	}

	if signature == "" {
		return false
	}

	// Убираем префикс "sha256="
	if strings.HasPrefix(signature, "sha256=") {
		signature = signature[7:]
	}

	// Вычисляем HMAC
	h := hmac.New(sha256.New, []byte(secretToken))
	h.Write(body)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	// Сравниваем подписи
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// validateEnvironment проверяет наличие всех необходимых переменных окружения
func validateEnvironment() error {
	if os.Getenv("TELEGRAM_BOT_TOKEN") == "" {
		return fmt.Errorf("TELEGRAM_BOT_TOKEN environment variable is required")
	}
	return eljur.ValidateConfig()
}

// Handler обрабатывает входящие webhook от Telegram (с улучшенной безопасностью)
func Handler(w http.ResponseWriter, r *http.Request) {
	log.Printf("[WEBHOOK] Received %s request from %s", r.Method, r.RemoteAddr)
	
	// Устанавливаем CORS заголовки для безопасности
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Проверяем метод запроса
	if r.Method != "POST" {
		log.Printf("[SECURITY] Invalid method: %s from %s", r.Method, r.RemoteAddr)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем наличие необходимых переменных окружения
	if err := validateEnvironment(); err != nil {
		log.Printf("[SECURITY] Environment validation failed: %v", err)
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

	// Создаем экземпляр бота (оптимизировано для serverless)
	diaryBot, err := bot.NewBot(botToken)
	if err != nil {
		log.Printf("Ошибка создания бота: %v", err)
		http.Error(w, "Failed to create bot", http.StatusInternalServerError)
		return
	}

	// Читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[SECURITY] Failed to read request body from %s: %v", r.RemoteAddr, err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	// Проверяем подпись webhook (если настроен секретный токен)
	webhookSecret := os.Getenv("WEBHOOK_SECRET")
	signature := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
	if !validateWebhookSignature(body, signature, webhookSecret) {
		log.Printf("[SECURITY] Invalid webhook signature from %s", r.RemoteAddr)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
		return
	}

	log.Printf("[WEBHOOK] Webhook signature validated successfully")

	// Парсим JSON
	var update tgbotapi.Update
	if err := json.Unmarshal(body, &update); err != nil {
		log.Printf("Ошибка парсинга JSON: %v", err)
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Обрабатываем обновление (с управлением состоянием для serverless)
	var processingError error
	if update.Message != nil {
		log.Printf("[WEBHOOK] Processing message from user %d: %s", update.Message.From.ID, update.Message.Text)
		userState := diaryBot.GetUserStateServerless(update.Message.Chat.ID)
		if err := diaryBot.HandleMessage(update.Message); err != nil {
			log.Printf("Ошибка обработки сообщения от пользователя %d: %v", update.Message.From.ID, err)
			processingError = err
		} else {
			log.Printf("[WEBHOOK] Successfully processed message from user %d", update.Message.From.ID)
		}
		// Сохраняем состояние пользователя
		diaryBot.SaveUserStateServerless(userState)
	} else if update.CallbackQuery != nil {
		log.Printf("[WEBHOOK] Processing callback query from user %d: %s", update.CallbackQuery.From.ID, update.CallbackQuery.Data)
		userState := diaryBot.GetUserStateServerless(update.CallbackQuery.Message.Chat.ID)
		if err := diaryBot.HandleCallback(update.CallbackQuery); err != nil {
			log.Printf("Ошибка обработки callback от пользователя %d: %v", update.CallbackQuery.From.ID, err)
			processingError = err
		} else {
			log.Printf("[WEBHOOK] Successfully processed callback from user %d", update.CallbackQuery.From.ID)
		}
		// Сохраняем состояние пользователя
		diaryBot.SaveUserStateServerless(userState)
	} else {
		log.Printf("[WEBHOOK] Received unsupported update type: %+v", update)
	}

	// Отвечаем успехом (с учетом ошибок обработки)
	w.WriteHeader(http.StatusOK)
	if processingError != nil {
		log.Printf("[WEBHOOK] Responding with processing error: %v", processingError)
		response := map[string]string{
			"status": "OK",
			"message": "Request processed with errors",
		}
		if jsonResp, err := json.Marshal(response); err == nil {
			w.Write(jsonResp)
		} else {
			w.Write([]byte(`{"status": "OK", "message": "Request processed with errors"}`))
		}
	} else {
		log.Printf("[WEBHOOK] Request processed successfully")
		response := map[string]string{"status": "OK"}
		if jsonResp, err := json.Marshal(response); err == nil {
			w.Write(jsonResp)
		} else {
			w.Write([]byte(`{"status": "OK"}`))
		}
	}
}