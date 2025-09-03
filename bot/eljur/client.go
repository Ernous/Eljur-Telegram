package eljur

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// getBaseURL возвращает базовый URL API из переменных окружения
func getBaseURL() string {
	if url := os.Getenv("ELJUR_API_URL"); url != "" {
		return url
	}
	return "https://eljur.gospmr.org/apiv3/" // Значение по умолчанию
}

// getDevKey возвращает ключ разработчика из переменных окружения
func getDevKey() string {
	return os.Getenv("ELJUR_DEV_KEY")
}

// Client представляет клиент для работы с API Эльжур
type Client struct {
	httpClient   *http.Client
	authToken    string
	userLogin    string
	studentID    string
	studentClass string
	domain       string
	cookies      map[string]string
}

// NewClient создает новый клиент
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cookies: make(map[string]string),
	}
}

// ValidateConfig проверяет наличие необходимых переменных окружения
func ValidateConfig() error {
	if getDevKey() == "" {
		return fmt.Errorf("ELJUR_DEV_KEY environment variable is required")
	}
	if getBaseURL() == "" {
		return fmt.Errorf("ELJUR_API_URL environment variable is required")
	}
	return nil
}

// Response представляет базовую структуру ответа API
type Response struct {
	Response struct {
		State  int         `json:"state"`
		Error  string      `json:"error,omitempty"`
		Result interface{} `json:"result,omitempty"`
	} `json:"response"`
}

// AuthResponse представляет ответ на запрос авторизации
type AuthResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Token string `json:"token"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// RulesResponse представляет ответ на запрос правил пользователя
type RulesResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			ID        interface{} `json:"id"`
			Name      interface{} `json:"name"`
			Relations struct {
				Students map[string]struct {
					Class string `json:"class"`
				} `json:"students,omitempty"`
				Groups map[string]interface{} `json:"groups,omitempty"`
			} `json:"relations,omitempty"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// Period представляет учебный период
type Period struct {
	Name     string `json:"name"`
	FullName string `json:"fullname"`
	Disabled bool   `json:"disabled"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Weeks    []Week `json:"weeks,omitempty"`
}

// Week представляет учебную неделю
type Week struct {
	Start string `json:"start"`
	End   string `json:"end"`
	Title string `json:"title"`
}

// PeriodsResponse представляет ответ на запрос периодов
type PeriodsResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Students []struct {
				Name    interface{} `json:"name"`
				Title   string      `json:"title"`
				Periods []Period    `json:"periods"`
			} `json:"students"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// DiaryResponse представляет ответ на запрос дневника
type DiaryResponse struct {
	Response struct {
		State  int                    `json:"state"`
		Error  string                 `json:"error,omitempty"`
		Result map[string]interface{} `json:"result,omitempty"`
	} `json:"response"`
}

// DiaryStudent представляет студента в дневнике
type DiaryStudent struct {
	Name interface{} `json:"name"`
	Days []DiaryDay  `json:"days,omitempty"`
}

// DiaryDay представляет день в дневнике
type DiaryDay struct {
	Date    string        `json:"date"`
	Lessons []DiaryLesson `json:"lessons,omitempty"`
}

// DiaryLesson представляет урок в дневнике
type DiaryLesson struct {
	Name     string      `json:"name"`
	Number   int         `json:"number"`
	Marks    []DiaryMark `json:"marks,omitempty"`
	Homework string      `json:"homework,omitempty"`
}

// DiaryMark представляет оценку
type DiaryMark struct {
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Message представляет сообщение
type Message struct {
	ID        string `json:"id"`
	Subject   string `json:"subject"`
	Text      string `json:"text"`
	ShortText string `json:"short_text"`
	UserFrom  struct {
		Name       string `json:"name"`
		LastName   string `json:"lastname"`
		FirstName  string `json:"firstname"`
		MiddleName string `json:"middlename"`
	} `json:"user_from"`
	UserTo []struct {
		Name       string `json:"name"`
		LastName   string `json:"lastname"`
		FirstName  string `json:"firstname"`
		MiddleName string `json:"middlename"`
	} `json:"user_to"`
	Files []struct {
		FileName string `json:"filename"`
		Link     string `json:"link"`
	} `json:"files"`
	Date   string `json:"date"`
	Read   bool   `json:"read"`
	Folder string `json:"folder"`
}

// MessagesResponse представляет ответ на запрос сообщений
type MessagesResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Messages []Message `json:"messages"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// MessageDetailsResponse представляет ответ на запрос деталей сообщения
type MessageDetailsResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Message Message `json:"message"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// Receiver представляет получателя сообщения
type Receiver struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// ReceiversResponse представляет ответ на запрос получателей
type ReceiversResponse struct {
	Response struct {
		State  int                    `json:"state"`
		Error  string                 `json:"error,omitempty"`
		Result map[string]interface{} `json:"result,omitempty"`
	} `json:"response"`
}

// SendMessageResponse представляет ответ на отправку сообщения
type SendMessageResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// ScheduleResponse представляет ответ на запрос расписания
type ScheduleResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Students []struct {
				Name interface{} `json:"name"`
				Days []struct {
					Date    string `json:"date"`
					Lessons []struct {
						Name    string `json:"name"`
						Number  int    `json:"number"`
						Teacher string `json:"teacher"`
						Room    string `json:"room"`
						Time    string `json:"time"`
					} `json:"lessons,omitempty"`
				} `json:"days,omitempty"`
			} `json:"students"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// MarksResponse представляет ответ на запрос оценок
type MarksResponse struct {
	Response struct {
		State  int    `json:"state"`
		Error  string `json:"error,omitempty"`
		Result struct {
			Students []struct {
				Name     interface{} `json:"name"`
				Subjects []struct {
					Name  string `json:"name"`
					Marks []struct {
						Value string `json:"value"`
						Date  string `json:"date"`
						Type  string `json:"type"`
					} `json:"marks"`
				} `json:"subjects"`
			} `json:"students"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// IsAuthenticated проверяет, авторизован ли пользователь
func (c *Client) IsAuthenticated() bool {
	return c.authToken != "" && c.domain != ""
}

// makeRequest выполняет HTTP запрос
func (c *Client) makeRequest(method, endpoint string, params url.Values, data url.Values) (*http.Response, error) {
	var req *http.Request
	var err error

	fullURL := getBaseURL() + endpoint

	if method == "POST" {
		// Для POST запроса добавляем параметры к URL, как в Python коде
		if params != nil && len(params) > 0 {
			fullURL += "?" + params.Encode()
		}

		// Данные отправляем в теле запроса
		req, err = http.NewRequest("POST", fullURL, bytes.NewBufferString(data.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		if params != nil {
			fullURL += "?" + params.Encode()
		}
		req, err = http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return nil, err
		}
	}

	// Добавляем заголовки точно как в Python коде
	req.Header.Set("User-Agent", "")
	req.Header.Set("Accept-Encoding", "gzip")

	// Добавляем cookies
	if len(c.cookies) > 0 {
		var cookieStr string
		for k, v := range c.cookies {
			if cookieStr != "" {
				cookieStr += "; "
			}
			cookieStr += k + "=" + v
		}
		req.Header.Set("Cookie", cookieStr)
	}

	log.Printf("[REQUEST] Финальный URL: %s", req.URL.String())
	log.Printf("[REQUEST] Метод: %s", method)
	log.Printf("[REQUEST] Заголовки: %v", req.Header)
	if req.Body != nil && method == "POST" {
		log.Printf("[REQUEST] Тело запроса: %s", data.Encode())
	}

	return c.httpClient.Do(req)
}

// readResponseBody читает и декодирует тело ответа (поддержка gzip)
func (c *Client) readResponseBody(resp *http.Response) ([]byte, error) {
	var reader io.Reader = resp.Body

	// Проверяем, сжат ли ответ gzip
	if strings.Contains(resp.Header.Get("Content-Encoding"), "gzip") {
		log.Printf("[RESPONSE] Обнаружено gzip сжатие, декодируем...")
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("ошибка создания gzip ридера: %w", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	return io.ReadAll(reader)
}

// getCommonParams возвращает общие параметры для всех запросов
func (c *Client) getCommonParams() url.Values {
	params := url.Values{}
	params.Set("devkey", getDevKey())
	params.Set("out_format", "json")
	params.Set("auth_token", c.authToken)
	params.Set("vendor", "eljur")
	return params
}

// Authenticate выполняет авторизацию пользователя
func (c *Client) Authenticate(login, password string) error {
	log.Printf("[AUTH] Начинаем авторизацию пользователя: %s", login)

	params := url.Values{}
	params.Set("devkey", getDevKey())
	params.Set("out_format", "json")
	params.Set("auth_token", "")
	params.Set("vendor", "eljur")

	data := url.Values{}
	data.Set("login", login)
	data.Set("password", password)

	log.Printf("[AUTH] Отправляем запрос на: %s", getBaseURL()+"auth")
	log.Printf("[AUTH] Параметры: %s", params.Encode())
	log.Printf("[AUTH] Данные: login=%s, password=[HIDDEN]", login)

	resp, err := c.makeRequest("POST", "auth", params, data)
	if err != nil {
		log.Printf("[AUTH] Ошибка запроса: %v", err)
		return fmt.Errorf("ошибка запроса авторизации: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[AUTH] Получен ответ с кодом: %d", resp.StatusCode)
	log.Printf("[AUTH] Заголовки ответа: %v", resp.Header)

	if resp.StatusCode != 200 {
		body, _ := c.readResponseBody(resp)
		log.Printf("[AUTH] Тело ответа при ошибке: %s", string(body))
		return fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		log.Printf("[AUTH] Ошибка чтения тела ответа: %v", err)
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	log.Printf("[AUTH] Тело ответа: %s", string(body))

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		log.Printf("[AUTH] Ошибка парсинга JSON: %v", err)
		log.Printf("[AUTH] Сырой JSON: %s", string(body))
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	log.Printf("[AUTH] Разобранный ответ: %+v", authResp)

	if authResp.Response.State != 200 {
		log.Printf("[AUTH] Ошибка в ответе: State=%d, Error=%s", authResp.Response.State, authResp.Response.Error)
		return fmt.Errorf("ошибка API: %s", authResp.Response.Error)
	}

	if authResp.Response.Result.Token == "" {
		log.Printf("[AUTH] Токен пустой в ответе")
		return fmt.Errorf("получен пустой токен авторизации")
	}

	c.authToken = authResp.Response.Result.Token
	c.userLogin = login
	tokenPreview := c.authToken
	if len(tokenPreview) > 10 {
		tokenPreview = tokenPreview[:10]
	}
	log.Printf("[AUTH] Получен токен: %s...", tokenPreview)

	// Сохраняем cookies
	for _, cookie := range resp.Cookies() {
		log.Printf("[AUTH] Получен cookie: %s = %s", cookie.Name, cookie.Value)
		if cookie.Name == "school_domain" {
			c.domain = cookie.Value
			c.cookies[cookie.Name] = cookie.Value
			log.Printf("[AUTH] Сохранен домен: %s", c.domain)
		}
	}

	// Получаем информацию о пользователе
	log.Printf("[AUTH] Получаем информацию о пользователе...")
	return c.getRules()
}

// getRules получает информацию о пользователе
func (c *Client) getRules() error {
	log.Printf("[RULES] Запрашиваем информацию о пользователе...")
	params := c.getCommonParams()

	log.Printf("[RULES] Отправляем запрос на: %s", getBaseURL()+"getrules")
	log.Printf("[RULES] Параметры: %s", params.Encode())

	resp, err := c.makeRequest("GET", "getrules", params, nil)
	if err != nil {
		log.Printf("[RULES] Ошибка запроса: %v", err)
		return fmt.Errorf("ошибка запроса правил: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[RULES] Получен ответ с кодом: %d", resp.StatusCode)

	if resp.StatusCode != 200 {
		body, _ := c.readResponseBody(resp)
		log.Printf("[RULES] Тело ответа при ошибке: %s", string(body))
		return fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		log.Printf("[RULES] Ошибка чтения тела ответа: %v", err)
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	log.Printf("[RULES] Тело ответа: %s", string(body))

	var rulesResp RulesResponse
	if err := json.Unmarshal(body, &rulesResp); err != nil {
		log.Printf("[RULES] Ошибка парсинга JSON: %v", err)
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	log.Printf("[RULES] Разобранный ответ: %+v", rulesResp)

	if rulesResp.Response.State != 200 {
		log.Printf("[RULES] Ошибка в ответе: State=%d, Error=%s", rulesResp.Response.State, rulesResp.Response.Error)
		return fmt.Errorf("ошибка API: %s", rulesResp.Response.Error)
	}

	// Извлекаем ID студента
	if rulesResp.Response.Result.ID != nil {
		c.studentID = fmt.Sprintf("%v", rulesResp.Response.Result.ID)
	} else if rulesResp.Response.Result.Name != nil {
		c.studentID = fmt.Sprintf("%v", rulesResp.Response.Result.Name)
	}

	// Извлекаем класс студента
	for _, student := range rulesResp.Response.Result.Relations.Students {
		if student.Class != "" {
			c.studentClass = student.Class
			break
		}
	}

	return nil
}

// GetPeriods получает периоды обучения
func (c *Client) GetPeriods(weeks, showDisabled bool) (*PeriodsResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	params := c.getCommonParams()
	if weeks {
		params.Set("weeks", "true")
	} else {
		params.Set("weeks", "false")
	}
	if showDisabled {
		params.Set("show_disabled", "true")
	} else {
		params.Set("show_disabled", "false")
	}

	resp, err := c.makeRequest("GET", "getperiods", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса периодов: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var periodsResp PeriodsResponse
	if err := json.Unmarshal(body, &periodsResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if periodsResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", periodsResp.Response.Error)
	}

	return &periodsResp, nil
}

// GetDiary получает дневник за указанный период
func (c *Client) GetDiary(days string) (*DiaryResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	params := c.getCommonParams()
	params.Set("student", c.studentID)
	params.Set("days", days)
	params.Set("rings", "true")

	log.Printf("[DIARY] Запрашиваем дневник за период: %s", days)

	resp, err := c.makeRequest("GET", "getdiary", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса дневника: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	log.Printf("[DIARY] Тело ответа: %s", string(body))

	var diaryResp DiaryResponse
	if err := json.Unmarshal(body, &diaryResp); err != nil {
		log.Printf("[DIARY] Ошибка парсинга JSON: %v", err)
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if diaryResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", diaryResp.Response.Error)
	}

	return &diaryResp, nil
}

// GetStudentID возвращает ID студента
func (c *Client) GetStudentID() string {
	return c.studentID
}

// GetStudentClass возвращает класс студента
func (c *Client) GetStudentClass() string {
	return c.studentClass
}

// GetMessages получает сообщения (входящие или отправленные)
func (c *Client) GetMessages(folder string) (*MessagesResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	params := c.getCommonParams()
	params.Set("folder", folder)

	resp, err := c.makeRequest("GET", "getmessages", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса сообщений: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var messagesResp MessagesResponse
	if err := json.Unmarshal(body, &messagesResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if messagesResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", messagesResp.Response.Error)
	}

	return &messagesResp, nil
}

// GetMessageDetails получает детали конкретного сообщения
func (c *Client) GetMessageDetails(messageID string) (*MessageDetailsResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	params := c.getCommonParams()
	params.Set("id", messageID)

	resp, err := c.makeRequest("GET", "getmessageinfo", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса деталей сообщения: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	log.Printf("[MESSAGE_DETAILS] Тело ответа: %s", string(body))

	var detailsResp MessageDetailsResponse
	if err := json.Unmarshal(body, &detailsResp); err != nil {
		log.Printf("[MESSAGE_DETAILS] Ошибка парсинга JSON: %v", err)
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	log.Printf("[MESSAGE_DETAILS] Разобранный ответ: %+v", detailsResp)

	if detailsResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", detailsResp.Response.Error)
	}

	return &detailsResp, nil
}

// GetMessageReceivers получает список доступных получателей сообщений
func (c *Client) GetMessageReceivers() (*ReceiversResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	params := c.getCommonParams()

	resp, err := c.makeRequest("GET", "getmessagereceivers", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса получателей: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var receiversResp ReceiversResponse
	if err := json.Unmarshal(body, &receiversResp); err != nil {
		log.Printf("[RECEIVERS] Ошибка парсинга JSON: %v", err)
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	log.Printf("[RECEIVERS] Разобранный ответ: %+v", receiversResp)

	if receiversResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", receiversResp.Response.Error)
	}

	return &receiversResp, nil
}

// SendMessage отправляет сообщение
func (c *Client) SendMessage(recipients []string, subject, text string) (*SendMessageResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	params := c.getCommonParams()

	data := url.Values{}
	data.Set("users_to", fmt.Sprintf("%v", recipients))
	data.Set("subject", subject)
	data.Set("text", text)

	resp, err := c.makeRequest("POST", "sendmessage", params, data)
	if err != nil {
		return nil, fmt.Errorf("ошибка отправки сообщения: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var sendResp SendMessageResponse
	if err := json.Unmarshal(body, &sendResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if sendResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", sendResp.Response.Error)
	}

	return &sendResp, nil
}

// GetSchedule получает расписание занятий
func (c *Client) GetSchedule(days, classID string) (*ScheduleResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	// Если не указан период дней, используем текущую неделю
	if days == "" {
		days = "20250512-20250518" // Пример недели
	}

	// Если не указан класс, используем класс пользователя
	if classID == "" {
		classID = c.studentClass
	}

	params := c.getCommonParams()
	params.Set("student", c.studentID)
	params.Set("days", days)
	params.Set("class", classID)
	params.Set("rings", "true")

	resp, err := c.makeRequest("GET", "getschedule", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса расписания: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var scheduleResp ScheduleResponse
	if err := json.Unmarshal(body, &scheduleResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if scheduleResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", scheduleResp.Response.Error)
	}

	return &scheduleResp, nil
}

// GetMarks получает оценки за указанный период
func (c *Client) GetMarks(period int, startDate, endDate string) (*MarksResponse, error) {
	if !c.IsAuthenticated() {
		return nil, fmt.Errorf("пользователь не авторизован")
	}

	if c.studentID == "" {
		return nil, fmt.Errorf("ID студента не найден")
	}

	// Определяем даты периода на основе выбранной четверти
	if period > 0 && period <= 4 {
		quarters := map[int][2]string{
			1: {"20240903", "20241102"}, // Первая четверть
			2: {"20241111", "20241230"}, // Вторая четверть
			3: {"20250120", "20250322"}, // Третья четверть
			4: {"20250331", "20250524"}, // Четвертая четверть
		}
		startDate = quarters[period][0]
		endDate = quarters[period][1]
	} else if startDate == "" || endDate == "" {
		// Если не указаны ни четверть, ни даты - используем текущую четверть
		startDate = "20250331"
		endDate = "20250524"
	}

	days := fmt.Sprintf("%s-%s", startDate, endDate)

	params := c.getCommonParams()
	params.Set("student", c.studentID)
	params.Set("days", days)

	resp, err := c.makeRequest("GET", "getmarks", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса оценок: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := c.readResponseBody(resp)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var marksResp MarksResponse
	if err := json.Unmarshal(body, &marksResp); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if marksResp.Response.State != 200 {
		return nil, fmt.Errorf("ошибка API: %s", marksResp.Response.Error)
	}

	return &marksResp, nil
}

// GetLogin возвращает логин пользователя (для session management)
func (c *Client) GetLogin() string {
	return c.userLogin
}

// GetToken возвращает токен аутентификации (для session management)
func (c *Client) GetToken() string {
	return c.authToken
}

// RestoreSession восстанавливает сессию по логину и токену
func (c *Client) RestoreSession(login, token string) error {
	if token == "" {
		return fmt.Errorf("токен не может быть пустым")
	}

	c.authToken = token
	c.userLogin = login

	// Проверяем валидность токена запросом getrules
	return c.getRules()
}
