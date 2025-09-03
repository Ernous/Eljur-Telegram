package eljur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	BaseURL = "https://eljur.gospmr.org/apiv3/"
	DevKey  = "dd06cf484d85581e1976d93c639deee7"
)

// Client представляет клиент для работы с API Эльжур
type Client struct {
	httpClient  *http.Client
	authToken   string
	studentID   string
	studentClass string
	domain      string
	cookies     map[string]string
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

// Response представляет базовую структуру ответа API
type Response struct {
	Response struct {
		State int         `json:"state"`
		Error string      `json:"error,omitempty"`
		Result interface{} `json:"result,omitempty"`
	} `json:"response"`
}

// AuthResponse представляет ответ на запрос авторизации
type AuthResponse struct {
	Response struct {
		State int `json:"state"`
		Error string `json:"error,omitempty"`
		Result struct {
			Token string `json:"token"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// RulesResponse представляет ответ на запрос правил пользователя
type RulesResponse struct {
	Response struct {
		State int `json:"state"`
		Error string `json:"error,omitempty"`
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
	Name      string `json:"name"`
	FullName  string `json:"fullname"`
	Disabled  bool   `json:"disabled"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Weeks     []Week `json:"weeks,omitempty"`
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
		State int `json:"state"`
		Error string `json:"error,omitempty"`
		Result struct {
			Students []struct {
				Name    interface{} `json:"name"`
				Title   string     `json:"title"`
				Periods []Period   `json:"periods"`
			} `json:"students"`
		} `json:"result,omitempty"`
	} `json:"response"`
}

// DiaryResponse представляет ответ на запрос дневника
type DiaryResponse struct {
	Response struct {
		State int `json:"state"`
		Error string `json:"error,omitempty"`
		Result struct {
			Students []struct {
				Name interface{} `json:"name"`
				Days []struct {
					Date    string `json:"date"`
					Lessons []struct {
						Name   string `json:"name"`
						Number int    `json:"number"`
						Marks  []struct {
							Value string `json:"value"`
							Type  string `json:"type"`
						} `json:"marks,omitempty"`
						Homework string `json:"homework,omitempty"`
					} `json:"lessons,omitempty"`
				} `json:"days,omitempty"`
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

	fullURL := BaseURL + endpoint

	if method == "POST" {
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

	// Добавляем заголовки
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

	return c.httpClient.Do(req)
}

// getCommonParams возвращает общие параметры для всех запросов
func (c *Client) getCommonParams() url.Values {
	params := url.Values{}
	params.Set("devkey", DevKey)
	params.Set("out_format", "json")
	params.Set("auth_token", c.authToken)
	params.Set("vendor", "eljur")
	return params
}

// Authenticate выполняет авторизацию пользователя
func (c *Client) Authenticate(login, password string) error {
	params := url.Values{}
	params.Set("devkey", DevKey)
	params.Set("out_format", "json")
	params.Set("auth_token", "")
	params.Set("vendor", "eljur")

	data := url.Values{}
	data.Set("login", login)
	data.Set("password", password)

	resp, err := c.makeRequest("POST", "auth", params, data)
	if err != nil {
		return fmt.Errorf("ошибка запроса авторизации: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if authResp.Response.State != 200 {
		return fmt.Errorf("ошибка API: %s", authResp.Response.Error)
	}

	c.authToken = authResp.Response.Result.Token

	// Сохраняем cookies
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "school_domain" {
			c.domain = cookie.Value
			c.cookies[cookie.Name] = cookie.Value
		}
	}

	// Получаем информацию о пользователе
	return c.getRules()
}

// getRules получает информацию о пользователе
func (c *Client) getRules() error {
	params := c.getCommonParams()

	resp, err := c.makeRequest("GET", "getrules", params, nil)
	if err != nil {
		return fmt.Errorf("ошибка запроса правил: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var rulesResp RulesResponse
	if err := json.Unmarshal(body, &rulesResp); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	if rulesResp.Response.State != 200 {
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

	body, err := io.ReadAll(resp.Body)
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

	resp, err := c.makeRequest("GET", "getdiary", params, nil)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса дневника: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP ошибка: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var diaryResp DiaryResponse
	if err := json.Unmarshal(body, &diaryResp); err != nil {
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