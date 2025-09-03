package gemini

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client представляет клиент для работы с Gemini AI
type Client struct {
	httpClient *http.Client
	apiKey     string
	model      string
}

// NewClient создает новый клиент Gemini
func NewClient(apiKey, model string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
		model:  model,
	}
}

// GeminiRequest представляет запрос к Gemini
type GeminiRequest struct {
	Contents []Content `json:"contents"`
}

// Content представляет содержимое сообщения
type Content struct {
	Parts []Part `json:"parts"`
}

// Part представляет часть сообщения
type Part struct {
	Text string `json:"text"`
}

// GeminiResponse представляет ответ от Gemini
type GeminiResponse struct {
	Candidates []Candidate `json:"candidates"`
	Error      *GeminiError `json:"error,omitempty"`
}

// Candidate представляет кандидата ответа
type Candidate struct {
	Content Content `json:"content"`
}

// GeminiError представляет ошибку от Gemini
type GeminiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// SendMessage отправляет сообщение в Gemini и получает ответ
func (c *Client) SendMessage(message string, context string) (string, error) {
	// Формируем полный запрос с контекстом
	fullMessage := message
	if context != "" {
		fullMessage = fmt.Sprintf("Контекст: %s\n\nВопрос: %s", context, message)
	}

	// Подготавливаем запрос
	request := GeminiRequest{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: fullMessage},
				},
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("ошибка создания JSON: %w", err)
	}

	// Формируем URL
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Отправляем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	// Проверяем ошибки
	if geminiResp.Error != nil {
		return "", fmt.Errorf("ошибка Gemini API: %s", geminiResp.Error.Message)
	}

	// Извлекаем ответ
	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("получен пустой ответ от Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// ValidateAPIKey проверяет валидность API ключа
func (c *Client) ValidateAPIKey() error {
	// Проверяем валидность ключа
	testClient := NewClient(c.apiKey, "gemini-2.0-flash-exp")
	_, err := testClient.SendMessage("Привет! Это тестовое сообщение для проверки API ключа.", "")
	return err
}

// GetAvailableModels возвращает доступные модели Gemini
func GetAvailableModels() []string {
	return []string{
		"gemini-1.5-flash",
		"gemini-1.5-pro",
		"gemini-1.0-pro",
		"gemini-2.0-flash-exp",
		"gemini-2.5-pro",
		"gemini-2.5-flash",
	}
}

// GetModelDescription возвращает описание модели
func GetModelDescription(model string) string {
	descriptions := map[string]string{
		"gemini-1.5-flash": "🚀 Быстрая модель - оптимальна для простых запросов",
		"gemini-1.5-pro":   "🧠 Продвинутая модель - лучше для сложных задач",
		"gemini-1.0-pro":   "⚡ Стандартная модель - баланс скорости и качества",
		"gemini-2.0-flash-exp": "✨ Новейшая модель Gemini 2.0 Flash Experimental",
		"gemini-2.5-pro": "🚀 Продвинутая модель Gemini 2.5 Pro",
		"gemini-2.5-flash": "⚡ Быстрая модель Gemini 2.5 Flash",
	}

	if desc, exists := descriptions[model]; exists {
		return desc
	}
	return "📝 Стандартная модель Gemini"
}