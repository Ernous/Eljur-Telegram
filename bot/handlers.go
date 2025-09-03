package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"school-diary-bot/bot/eljur"
	"school-diary-bot/internal/gemini"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// splitMessage умно разбивает текст на части, стараясь не разрывать предложения
func splitMessage(text string, maxLength int) []string {
	if len(text) <= maxLength {
		return []string{text}
	}

	var parts []string
	remaining := text

	for len(remaining) > 0 {
		if len(remaining) <= maxLength {
			parts = append(parts, remaining)
			break
		}

		// Ищем лучшее место для разрыва (конец предложения, параграфа или слова)
		cutIndex := maxLength

		// Ищем ближайший перенос строки назад от максимальной длины
		if idx := strings.LastIndex(remaining[:maxLength], "\n\n"); idx > maxLength/2 {
			cutIndex = idx + 2
		} else if idx := strings.LastIndex(remaining[:maxLength], "\n"); idx > maxLength/2 {
			cutIndex = idx + 1
		} else if idx := strings.LastIndex(remaining[:maxLength], ". "); idx > maxLength/2 {
			cutIndex = idx + 2
		} else if idx := strings.LastIndex(remaining[:maxLength], " "); idx > maxLength/2 {
			cutIndex = idx + 1
		}

		parts = append(parts, strings.TrimSpace(remaining[:cutIndex]))
		remaining = strings.TrimSpace(remaining[cutIndex:])
	}

	return parts
}

// formatDateRu преобразует дату из формата YYYYMMDD в русский формат
func formatDateRu(dateStr string) string {
	if len(dateStr) != 8 {
		return dateStr
	}

	year := dateStr[:4]
	month := dateStr[4:6]
	day := dateStr[6:8]

	monthNames := map[string]string{
		"01": "января", "02": "февраля", "03": "марта", "04": "апреля",
		"05": "мая", "06": "июня", "07": "июля", "08": "августа",
		"09": "сентября", "10": "октября", "11": "ноября", "12": "декабря",
	}

	monthName := monthNames[month]
	if monthName == "" {
		monthName = month
	}

	// Убираем ведущий ноль из дня
	dayInt, _ := strconv.Atoi(day)
	return fmt.Sprintf("%d %s %s", dayInt, monthName, year)
}

// HandleMessage обрабатывает текстовые сообщения
func (b *Bot) HandleMessage(message *tgbotapi.Message) error {
	user := b.GetUserState(message.Chat.ID)
	text := message.Text

	switch user.State {
	case "auth_waiting":
		return b.handleAuthInput(user, text)
	case "message_compose_subject":
		return b.handleMessageSubject(user, text)
	case "message_compose_text":
		return b.handleMessageText(user, text)
	case "gemini_api_setup":
		// Удаляем сообщение с API ключом для безопасности
		deleteMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, message.MessageID)
		b.API.Send(deleteMsg)
		return b.handleGeminiAPISetup(user, text)
	case "gemini_chat":
		return b.handleGeminiChat(user, text)
	default:
		return b.handleCommands(user, text)
	}
}

// handleCommands обрабатывает команды бота
func (b *Bot) handleCommands(user *UserState, text string) error {
	// Проверяем команды с параметрами
	if strings.HasPrefix(text, "/login ") {
		return b.handleLoginWithParams(user, text)
	}
	if strings.HasPrefix(text, "/messages send ") {
		return b.handleMessageSendWithParams(user, text)
	}
	if strings.HasPrefix(text, "/gemini ") {
		return b.handleGeminiWithParams(user, text)
	}

	switch text {
	case "/start":
		return b.handleStart(user)
	case "/help":
		return b.handleHelp(user)
	case "/login":
		return b.handleLogin(user)
	case "/logout":
		return b.handleLogout(user)
	case "/diary":
		return b.handleDiary(user)
	case "/periods":
		return b.handlePeriods(user)
	case "/messages":
		return b.handleMessages(user)
	case "/schedule":
		return b.handleSchedule(user)
	case "/marks":
		return b.handleMarks(user)
	case "/gemini":
		return b.handleGemini(user)
	default:
		return b.SendMessage(user.ChatID, "❓ Неизвестная команда. Используйте /help для получения справки.", nil)
	}
}

// handleStart обрабатывает команду /start
func (b *Bot) handleStart(user *UserState) error {
	log.Printf("[START] User %d - IsAuthenticated: %v, Login: %s, Token length: %d", user.ChatID, user.Client.IsAuthenticated(), user.Client.GetLogin(), len(user.Client.GetToken()))
	
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📚 Дневник", "diary"),
			tgbotapi.NewInlineKeyboardButtonData("📅 Периоды", "periods"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Сообщения", "messages"),
			tgbotapi.NewInlineKeyboardButtonData("📋 Расписание", "schedule"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Оценки", "marks"),
			tgbotapi.NewInlineKeyboardButtonData("🔐 Войти", "login"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("ℹ️ Помощь", "help"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Gemini AI", "gemini"),
		),
	)

	welcomeText := "👋 <b>Добро пожаловать в школьный электронный дневник!</b>\n\n"
	if user.Client.IsAuthenticated() {
		welcomeText += "✅ Вы авторизованы\n\n"
	} else {
		welcomeText += "⚠️ Для доступа ко всем функциям необходимо авторизоваться\n\n"
	}
	welcomeText += "Выберите действие:"

	return b.SendMessage(user.ChatID, welcomeText, keyboard)
}

// handleHelp обрабатывает команду /help
func (b *Bot) handleHelp(user *UserState) error {
	helpText := "🤖 <b>Школьный электронный дневник</b>\n\n" +
		"<b>Основные команды:</b>\n" +
		"/start - Главное меню\n" +
		"/login - Авторизация в системе\n" +
		"/logout - Выход из системы\n" +
		"/diary - Просмотр дневника\n" +
		"/periods - Учебные периоды\n" +
		"/messages - Сообщения\n" +
		"/schedule - Расписание занятий\n" +
		"/marks - Оценки по предметам\n" +
		"/gemini - Gemini AI Ассистент\n" +
		"/help - Эта справка\n\n" +
		"<b>Быстрые команды:</b>\n" +
		"/login логин пароль - быстрая авторизация\n" +
		"/messages send ID \"\u0442\u0435\u043c\u0430\" \"\u0442\u0435\u043a\u0441\u0442\" - быстрая отправка сообщения\n" +
		"/gemini вопрос - быстрый запрос к AI\n\n" +
		"<b>Примеры использования:</b>\n" +
		"<code>/login Ivanov password123</code>\n" +
		"<code>/messages send 123 \"Вопрос\" \"Привет, как дела?\"</code>\n" +
		"<code>/gemini Объясни закон Ньютона</code>\n\n" +
		"<b>Как пользоваться:</b>\n" +
		"1. Авторизуйтесь любым способом\n" +
		"2. Используйте команды или кнопки меню\n" +
		"3. Выбирайте недели и периоды для просмотра данных"

	return b.SendMessage(user.ChatID, helpText, nil)
}

// handleLogin обрабатывает авторизацию
func (b *Bot) handleLogin(user *UserState) error {
	if user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "✅ Вы уже авторизованы! Используйте /logout для выхода.", nil)
	}

	user.State = "auth_waiting"
	user.AuthStep = 1

	return b.SendMessage(user.ChatID, "🔐 <b>Авторизация</b>\n\nВведите ваш логин и пароль:\n\n<i>Пример: /login Ivanov passwd123</i>", nil)
}

// handleLoginWithParams обрабатывает авторизацию с параметрами /login username password
func (b *Bot) handleLoginWithParams(user *UserState, text string) error {
	if user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "✅ Вы уже авторизованы! Используйте /logout для выхода.", nil)
	}

	// Разбираем команду на части
	parts := strings.Fields(text)
	if len(parts) != 3 {
		return b.SendMessage(user.ChatID, "❌ Неверный формат команды.\n\n<b>Используйте:</b>\n/login логин пароль\n\n<b>Пример:</b>\n/login Ivanov password123\n\nИли используйте /login для пошаговой авторизации.", nil)
	}

	username := strings.TrimSpace(parts[1])
	password := strings.TrimSpace(parts[2])

	if username == "" || password == "" {
		return b.SendMessage(user.ChatID, "❌ Логин и пароль не могут быть пустыми.", nil)
	}

	// Отправляем сообщение о процессе авторизации
	b.SendMessage(user.ChatID, "🔄 Проверяем данные авторизации...", nil)

	// Выполняем авторизацию
	err := user.Client.Authenticate(username, password)

	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка авторизации: %v\n\nПроверьте правильность логина и пароля.", err), nil)
	}

	// Сохраняем состояние после успешной авторизации
	b.SaveUserStateIfNeeded(user)

	// После успешной авторизации показываем главное меню
	_ = b.SendMessage(user.ChatID, "✅ Авторизация успешна! Теперь вам доступны все функции дневника.", nil)
	return b.handleStart(user)
}

// handleMessageSendWithParams обрабатывает отправку сообщения с параметрами
func (b *Bot) handleMessageSendWithParams(user *UserState, text string) error {
	if !user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо авторизоваться через /login", nil)
	}

	// Убираем "/messages send " из начала команды
	params := strings.TrimPrefix(text, "/messages send ")

	// Парсим параметры: recipientID "subject" "message text"
	// Простой парсинг по пробелам и кавычкам
	parts := []string{}
	current := ""
	inQuotes := false

	for i, r := range params {
		if r == '"' {
			inQuotes = !inQuotes
		} else if r == ' ' && !inQuotes {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(r)
		}

		// Добавляем последнюю часть
		if i == len(params)-1 && current != "" {
			parts = append(parts, current)
		}
	}

	if len(parts) < 3 {
		return b.SendMessage(user.ChatID, "❌ Неверный формат команды.\n\n<b>Используйте:</b>\n/messages send получатель_ID \"\u0442\u0435\u043c\u0430\" \"\u0442\u0435\u043a\u0441\u0442 \u0441\u043e\u043e\u0431\u0449\u0435\u043d\u0438\u044f\"\n\n<b>Пример:</b>\n/messages send 123 \"Вопрос по уроку\" \"Привет! Можно ли получить домашнее задание?\"", nil)
	}

	recipientID := strings.TrimSpace(parts[0])
	subject := strings.TrimSpace(parts[1])
	messageText := strings.TrimSpace(parts[2])

	if recipientID == "" || subject == "" || messageText == "" {
		return b.SendMessage(user.ChatID, "❌ Все параметры обязательны.", nil)
	}

	// Отправляем сообщение
	b.SendMessage(user.ChatID, "📤 Отправляем сообщение...", nil)

	recipients := []string{recipientID}
	_, err := user.Client.SendMessage(recipients, subject, messageText)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка отправки сообщения: %v", err), nil)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✉️ Написать еще", "msg_compose"),
			tgbotapi.NewInlineKeyboardButtonData("📥 К сообщениям", "messages"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, fmt.Sprintf("✅ <b>Сообщение отправлено!</b>\n\n👤 Получатель: %s\n📝 Тема: %s", recipientID, subject), keyboard)
}

// handleGeminiWithParams обрабатывает запрос к Gemini AI с параметрами
func (b *Bot) handleGeminiWithParams(user *UserState, text string) error {
	// Проверяем, настроен ли Gemini
	if user.GeminiAPIKey == "" {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо настроить Gemini AI через /gemini", nil)
	}

	// Убираем "/gemini " из начала команды
	prompt := strings.TrimPrefix(text, "/gemini ")
	prompt = strings.TrimSpace(prompt)

	if prompt == "" {
		return b.SendMessage(user.ChatID, "❌ Пустой запрос.\n\n<b>Используйте:</b>\n/gemini ваш вопрос\n\n<b>Пример:</b>\n/gemini Объясни мне закон Ньютона", nil)
	}

	// Отправляем сообщение о обработке
	processingMsg := tgbotapi.NewMessage(user.ChatID, "🤖 Обрабатываем ваш запрос...")
	sentMsg, _ := b.API.Send(processingMsg)

	// Создаем клиент Gemini
	client := gemini.NewClient(user.GeminiAPIKey, user.GeminiModel)

	// Отправляем запрос к Gemini
	response, err := client.SendMessage(prompt, user.GeminiContext)

	// Удаляем сообщение "думает"
	if sentMsg.MessageID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(user.ChatID, sentMsg.MessageID)
		b.API.Send(deleteMsg)
	}
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка Gemini AI: %v", err), nil)
	}

	// Ограничиваем длину ответа
	maxLength := 3900
	if len(response) > maxLength {
		response = response[:maxLength] + "...\n\n[Ответ сокращен]"
	}

	// Очищаем ответ от потенциально проблемных символов
	response = strings.ReplaceAll(response, "\u0000", "")
	response = strings.ReplaceAll(response, "`", "'")   // Заменяем обратные кавычки
	response = strings.ReplaceAll(response, "*", "\\*") // Экранируем звездочки
	response = strings.ReplaceAll(response, "_", "\\_") // Экранируем подчеркивания
	response = strings.ReplaceAll(response, "[", "\\[") // Экранируем квадратные скобки
	response = strings.ReplaceAll(response, "]", "\\]")
	response = strings.TrimSpace(response)

	if response == "" {
		return b.SendMessage(user.ChatID, "❌ Gemini вернул пустой ответ. Попробуйте переформулировать вопрос.", nil)
	}

	// Сохраняем состояние
	b.SaveUserStateIfNeeded(user)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("💬 Продолжить чат", "gemini_chat"),
			tgbotapi.NewInlineKeyboardButtonData("🤖 Gemini меню", "gemini"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	// Отправляем ответ
	return b.SendMessage(user.ChatID, fmt.Sprintf("🤖 <b>Gemini AI:</b>\n\n%s", response), keyboard)
}

// handleLogout обрабатывает выход из системы
func (b *Bot) handleLogout(user *UserState) error {
	user.Client = eljur.NewClient()
	user.State = "idle"
	user.AuthStep = 0
	user.TempLogin = ""
	user.TempPassword = ""

	return b.SendMessage(user.ChatID, "👋 Вы вышли из системы.", nil)
}

// handleAuthInput обрабатывает ввод данных авторизации (оптимизировано для webhook)
func (b *Bot) handleAuthInput(user *UserState, text string) error {
	switch user.AuthStep {
	case 1: // Логин
		user.TempLogin = strings.TrimSpace(text)
		user.AuthStep = 2
		// Сохраняем состояние после обновления
		b.SaveUserStateIfNeeded(user)
		return b.SendMessage(user.ChatID, "🔑 Теперь введите ваш пароль:\n\n<i>Пример: password123</i>", nil)

	case 2: // Пароль
		user.TempPassword = strings.TrimSpace(text)

		// Отправляем сообщение о процессе авторизации
		b.SendMessage(user.ChatID, "🔄 Проверяем данные авторизации...", nil)

		// Выполняем авторизацию
		err := user.Client.Authenticate(user.TempLogin, user.TempPassword)

		// Очищаем временные данные
		user.TempLogin = ""
		user.TempPassword = ""
		user.State = "idle"
		user.AuthStep = 0

		if err != nil {
			b.SaveUserStateIfNeeded(user)
			return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка авторизации: %v\n\nПопробуйте еще раз с помощью /login", err), nil)
		}

		// Сохраняем состояние после успешной авторизации
		b.SaveUserStateIfNeeded(user)

		// После успешной авторизации показываем главное меню
		_ = b.SendMessage(user.ChatID, "✅ Авторизация успешна! Теперь вам доступны все функции дневника.", nil)
		return b.handleStart(user)
	}

	return nil
}

// HandleCallback обрабатывает нажатия на кнопки
func (b *Bot) HandleCallback(query *tgbotapi.CallbackQuery) error {
	user := b.GetUserState(query.Message.Chat.ID)
	data := query.Data

	// Отвечаем на callback query
	b.AnswerCallback(query.ID, "")

	switch {
	case data == "start":
		return b.handleStart(user)
	case data == "diary":
		return b.handleDiary(user)
	case data == "periods":
		return b.handlePeriods(user)
	case data == "messages":
		return b.handleMessages(user)
	case data == "schedule":
		return b.handleSchedule(user)
	case data == "marks":
		return b.handleMarks(user)
	case data == "login":
		return b.handleLogin(user)
	case data == "help":
		return b.handleHelp(user)
	case data == "clear_chat":
		return b.handleClearChat(user)
	case data == "gemini":
		return b.handleGemini(user)
	case data == "gemini_setup":
		return b.handleGeminiSetup(user)
	case data == "gemini_help":
		return b.handleGeminiHelp(user)
	case data == "gemini_model_select":
		return b.handleGeminiModelSelect(user, data) // Показать список моделей
	case strings.HasPrefix(data, "gemini_model_"):
		return b.handleGeminiModelSelect(user, data) // Выбор конкретной модели
	case data == "gemini_change_key":
		return b.handleGeminiSetup(user) // Показываем форму ввода ключа
	case data == "gemini_reset":
		return b.handleGeminiReset(user)
	case data == "gemini_chat":
		return b.handleGeminiChatStart(user)
	case strings.HasPrefix(data, "gemini_context_"):
		return b.handleGeminiContextSelect(user, data)
	case strings.HasPrefix(data, "week_"):
		return b.handleWeekSelect(user, data)
	case strings.HasPrefix(data, "period_"):
		return b.handlePeriodSelect(user, data)
	case strings.HasPrefix(data, "msg_read_"):
		return b.handleReadMessage(user, data)
	case strings.HasPrefix(data, "compose_to_"):
		return b.handleSelectRecipient(user, data)
	case strings.HasPrefix(data, "msg_"):
		return b.handleMessageAction(user, data)
	default:
		return b.SendMessage(user.ChatID, "🔄 Обрабатываем запрос...", nil)
	}
}

// handleDiary обрабатывает просмотр дневника
func (b *Bot) handleDiary(user *UserState) error {
	if !user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо авторизоваться через /login", nil)
	}

	// Получаем периоды для выбора недель
	periods, err := user.Client.GetPeriods(true, false)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения периодов: %v", err), nil)
	}

	if len(periods.Response.Result.Students) == 0 {
		return b.SendMessage(user.ChatID, "❌ Не найдены данные о студенте", nil)
	}

	student := periods.Response.Result.Students[0]
	if len(student.Periods) == 0 {
		return b.SendMessage(user.ChatID, "❌ Не найдены учебные периоды", nil)
	}

	// Показываем выбор недель из текущего периода
	return b.showWeekSelection(user, student.Periods[len(student.Periods)-1]) // Последний период (текущий)
}

// showWeekSelection показывает выбор недель
func (b *Bot) showWeekSelection(user *UserState, period eljur.Period) error {
	var keyboard [][]tgbotapi.InlineKeyboardButton

	text := fmt.Sprintf("📅 <b>Выберите неделю из %s:</b>\n\n", period.FullName)

	for i, week := range period.Weeks {
		if i%2 == 0 {
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{})
		}

		// Преобразуем даты в читабьый формат
		startFormatted := formatDateRu(week.Start)
		endFormatted := formatDateRu(week.End)
		weekTitle := fmt.Sprintf("%s - %s", startFormatted, endFormatted)

		weekData := fmt.Sprintf("week_%s_%s_%s", period.Name, week.Start, week.End)
		button := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("📅 %s", weekTitle),
			weekData,
		)

		keyboard[len(keyboard)-1] = append(keyboard[len(keyboard)-1], button)
	}

	// Добавляем кнопку "Назад"
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "start"),
	})

	return b.SendMessage(user.ChatID, text, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
}

// handleWeekSelect обрабатывает выбор недели
func (b *Bot) handleWeekSelect(user *UserState, data string) error {
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return b.SendMessage(user.ChatID, "❌ Ошибка выбора недели", nil)
	}

	startDate := parts[2]
	endDate := parts[3]

	days := fmt.Sprintf("%s-%s", startDate, endDate)
	user.CurrentWeek = days

	// Получаем дневник за выбранную неделю
	diary, err := user.Client.GetDiary(days)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения дневника: %v", err), nil)
	}

	return b.formatDiary(user, diary)
}

// formatDiary форматирует и отправляет дневник
func (b *Bot) formatDiary(user *UserState, diary *eljur.DiaryResponse) error {
	var diaryText strings.Builder
	diaryText.WriteString("📚 <b>Дневник за выбранную неделю:</b>\n\n")

	result := diary.Response.Result
	hasLessons := false

	// Ищем ключ "students" в результате
	studentsData, hasStudents := result["students"]
	if !hasStudents {
		diaryText.WriteString("📝 Данные о дневнике не найдены")
	} else {
		// studentsData должно быть объектом, где ключ - это ID студента
		if studentsMap, ok := studentsData.(map[string]interface{}); ok {
			// Проходим по каждому студенту
			for _, studentInfo := range studentsMap {
				// studentInfo должно содержать данные студента
				if studentData, ok := studentInfo.(map[string]interface{}); ok {

					// Ищем поле "days" в данных студента
					daysData, hasDays := studentData["days"]
					if !hasDays {
						diaryText.WriteString("📝 Данные о днях не найдены")
						continue
					}

					// days должно быть объектом с датами как ключами
					if daysMap, ok := daysData.(map[string]interface{}); ok {
						// Собираем все даты и сортируем их
						var dates []string
						for dateKey := range daysMap {
							if len(dateKey) == 8 && isDate(dateKey) {
								dates = append(dates, dateKey)
							}
						}

						// Сортируем даты
						for i := 0; i < len(dates); i++ {
							for j := i + 1; j < len(dates); j++ {
								if dates[i] > dates[j] {
									dates[i], dates[j] = dates[j], dates[i]
								}
							}
						}

						// Отображаем информацию по дням
						for _, dateKey := range dates {
							if dayInfo, exists := daysMap[dateKey]; exists {
								if dayData, ok := dayInfo.(map[string]interface{}); ok {
									title, _ := dayData["title"].(string)
									if title == "" {
										title = formatDateRu(dateKey)
									}

									diaryText.WriteString(fmt.Sprintf("📅 <b>%s</b>\n", title))

									// Проверяем есть ли праздник
									if alert, hasAlert := dayData["alert"]; hasAlert {
										if alert == "holiday" {
											if holidayName, ok := dayData["holiday_name"].(string); ok {
												diaryText.WriteString(fmt.Sprintf("   🎉 %s\n", holidayName))
											}
										} else if alert == "today" {
											diaryText.WriteString("   📍 Сегодня\n")
										}
									}

									// Ищем уроки в items
									itemsData, hasItems := dayData["items"]
									if !hasItems {
										diaryText.WriteString("   Уроков нет\n\n")
										continue
									}

									items, ok := itemsData.(map[string]interface{})
									if !ok || len(items) == 0 {
										diaryText.WriteString("   Уроков нет\n\n")
										continue
									}

									hasLessons = true

									// Сортируем уроки по номеру
									var lessonNumbers []string
									for lessonNum := range items {
										lessonNumbers = append(lessonNumbers, lessonNum)
									}

									// Простая сортировка номеров уроков
									for i := 0; i < len(lessonNumbers); i++ {
										for j := i + 1; j < len(lessonNumbers); j++ {
											num1, _ := strconv.Atoi(lessonNumbers[i])
											num2, _ := strconv.Atoi(lessonNumbers[j])
											if num1 > num2 {
												lessonNumbers[i], lessonNumbers[j] = lessonNumbers[j], lessonNumbers[i]
											}
										}
									}

									// Отображаем уроки
									for _, lessonNum := range lessonNumbers {
										if lessonData, exists := items[lessonNum]; exists {
											if lesson, ok := lessonData.(map[string]interface{}); ok {
												name, _ := lesson["name"].(string)
												teacher, _ := lesson["teacher"].(string)
												room, _ := lesson["room"].(string)
												starttime, _ := lesson["starttime"].(string)
												endtime, _ := lesson["endtime"].(string)

												diaryText.WriteString(fmt.Sprintf("   %s. %s", lessonNum, name))

												if teacher != "" {
													diaryText.WriteString(fmt.Sprintf("\n      👨‍🏫 %s", teacher))
												}

												if room != "" && room != " " {
													diaryText.WriteString(fmt.Sprintf("\n      🏫 Кабинет %s", room))
												}

												if starttime != "" && endtime != "" {
													diaryText.WriteString(fmt.Sprintf("\n      ⏰ %s - %s", starttime, endtime))
												}

												// Проверяем домашнее задание
												if homeworkData, ok := lesson["homework"]; ok {
													if homework, ok := homeworkData.(map[string]interface{}); ok && len(homework) > 0 {
														diaryText.WriteString("\n      📝 ДЗ:")
														for _, hwData := range homework {
															if hw, ok := hwData.(map[string]interface{}); ok {
																if value, ok := hw["value"].(string); ok && value != "" {
																	diaryText.WriteString(fmt.Sprintf(" %s", value))
																}
															}
														}
													}
												}

												diaryText.WriteString("\n")
											}
										}
									}
									diaryText.WriteString("\n")
								}
							}
						}
					}
					if !ok {
						diaryText.WriteString("📝 Ошибка обработки данных студентов")
						return nil
					}
				}
			}
		}

		if !hasLessons {
			diaryText.WriteString("📝 Уроков на этой неделе нет")
			return nil
		}

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать другую неделю", "diary"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
			),
		)

		if err := b.SendMessage(user.ChatID, diaryText.String(), keyboard); err != nil {
			return err
		}
		return nil
	}
	return nil
}

// isDate проверяет, является ли строка датой в формате YYYYMMDD
func isDate(s string) bool {
	if len(s) != 8 {
		return false
	}

	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	// Простая проверка на валидную дату
	year, _ := strconv.Atoi(s[:4])
	month, _ := strconv.Atoi(s[4:6])
	day, _ := strconv.Atoi(s[6:8])

	return year >= 2020 && year <= 2030 && month >= 1 && month <= 12 && day >= 1 && day <= 31
}

// handlePeriods обрабатывает просмотр периодов
func (b *Bot) handlePeriods(user *UserState) error {
	if !user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо авторизоваться через /login", nil)
	}

	periods, err := user.Client.GetPeriods(true, false)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения периодов: %v", err), nil)
	}

	if len(periods.Response.Result.Students) == 0 {
		return b.SendMessage(user.ChatID, "❌ Не найдены данные о студенте", nil)
	}

	student := periods.Response.Result.Students[0]
	text := "📅 <b>Учебные периоды:</b>\n\n"

	for _, period := range student.Periods {
		status := "✅"
		if period.Disabled {
			status = "⏸"
		}

		text += fmt.Sprintf("%s <b>%s</b>\n", status, period.FullName)
		startFormatted := formatDateRu(period.Start)
		endFormatted := formatDateRu(period.End)
		text += fmt.Sprintf("   📅 %s - %s\n", startFormatted, endFormatted)
		text += fmt.Sprintf("   📊 Недель: %d\n\n", len(period.Weeks))
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleMessages обрабатывает просмотр сообщений
func (b *Bot) handleMessages(user *UserState) error {
	if !user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо авторизоваться через /login", nil)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📥 Входящие", "msg_inbox"),
			tgbotapi.NewInlineKeyboardButtonData("📤 Отправленные", "msg_sent"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✍️ Написать сообщение", "msg_compose"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, "💬 <b>Сообщения</b>\n\nВыберите действие:", keyboard)
}

// handleMessageAction обрабатывает действия с сообщениями
func (b *Bot) handleMessageAction(user *UserState, action string) error {
	switch action {
	case "msg_inbox":
		return b.showMessages(user, "inbox")
	case "msg_sent":
		return b.showMessages(user, "sent")
	case "msg_compose":
		return b.startComposeMessage(user)
	default:
		return b.SendMessage(user.ChatID, "❌ Неизвестное действие", nil)
	}
}

// showMessages показывает список сообщений как интерактивные кнопки
func (b *Bot) showMessages(user *UserState, folder string) error {
	messages, err := user.Client.GetMessages(folder)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения сообщений: %v", err), nil)
	}

	folderName := "📥 Входящие"
	if folder == "sent" {
		folderName = "📤 Отправленные"
	}

	text := fmt.Sprintf("💬 <b>%s сообщения:</b>\n\nНажмите на сообщение для просмотра:", folderName)
	var keyboard [][]tgbotapi.InlineKeyboardButton

	if len(messages.Response.Result.Messages) == 0 {
		text += "\n\n<i>Сообщений нет</i>"
	} else {
		for i, msg := range messages.Response.Result.Messages {
			if i >= 15 { // Показываем только первые 15 сообщений
				break
			}

			subject := msg.Subject
			if len(subject) > 35 {
				subject = subject[:35] + "..."
			}

			// Определяем статус прочтения и отправителя
			readStatus := "📖"
			if !msg.Read {
				readStatus = "📩"
			}

			// Формируем имя отправителя из новой структуры
			sender := ""
			if msg.UserFrom.FirstName != "" || msg.UserFrom.LastName != "" {
				sender = fmt.Sprintf("%s %s", msg.UserFrom.LastName, msg.UserFrom.FirstName)
				sender = strings.TrimSpace(sender)
			}
			if sender == "" {
				sender = "Неизвестный"
			}
			if len(sender) > 20 {
				sender = sender[:20] + "..."
			}

			// Создаем кнопку для каждого сообщения
			buttonText := fmt.Sprintf("%s %s\n👤 %s", readStatus, subject, sender)
			callbackData := fmt.Sprintf("msg_read_%s_%s", folder, msg.ID)

			button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
			keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
		}
	}

	// Добавляем кнопки управления
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", fmt.Sprintf("msg_%s", folder)),
		tgbotapi.NewInlineKeyboardButtonData("🗑 Очистить чат", "clear_chat"),
	})
	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "messages"),
	})

	return b.SendMessage(user.ChatID, text, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
}

// handleClearChat очищает чат
func (b *Bot) handleClearChat(user *UserState) error {
	// Отправляем множество пустых сообщений чтобы "очистить" чат
	for i := 0; i < 20; i++ {
		_ = b.SendMessage(user.ChatID, ".", nil)
	}

	return b.SendMessage(user.ChatID, "🗑 <b>Чат очищен</b>\n\nВыберите действие:",
		tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
			),
		))
}

// handleReadMessage показывает содержимое сообщения
func (b *Bot) handleReadMessage(user *UserState, data string) error {
	parts := strings.Split(data, "_")
	if len(parts) < 4 {
		return b.SendMessage(user.ChatID, "❌ Ошибка открытия сообщения", nil)
	}

	folder := parts[2]
	messageID := parts[3]

	// Получаем детали сообщения
	msgDetails, err := user.Client.GetMessageDetails(messageID)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения сообщения: %v", err), nil)
	}

	if msgDetails.Response.State != 200 {
		return b.SendMessage(user.ChatID, "❌ Сообщение не найдено", nil)
	}

	message := msgDetails.Response.Result.Message

	// Формируем имя отправителя
	from := ""
	if message.UserFrom.FirstName != "" || message.UserFrom.LastName != "" {
		from = fmt.Sprintf("%s %s %s", message.UserFrom.LastName, message.UserFrom.FirstName, message.UserFrom.MiddleName)
		from = strings.TrimSpace(from)
	}

	// Формируем список получателей
	to := ""
	if len(message.UserTo) > 0 {
		var recipients []string
		for _, user := range message.UserTo {
			recipient := fmt.Sprintf("%s %s", user.LastName, user.FirstName)
			recipients = append(recipients, strings.TrimSpace(recipient))
		}
		to = strings.Join(recipients, ", ")
	}

	subject := message.Subject
	text := message.Text
	date := message.Date

	if from == "" && to != "" {
		from = "Вы → " + to
	} else if from == "" {
		from = "Неизвестный отправитель"
	}
	if subject == "" {
		subject = "Без темы"
	}
	// Очищаем HTML-теги из текста
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br>", "\n")

	if text == "" {
		text = "<i>Текст сообщения отсутствует</i>"
	}
	if date == "" {
		date = "<i>Дата не указана</i>"
	}

	messageText := fmt.Sprintf("📨 <b>Детали сообщения:</b>\n\n"+
		"👤 От: %s\n"+
		"📋 Тема: %s\n"+
		"📅 Дата: %s\n\n"+
		"📝 Сообщение:\n%s",
		from, subject, date, text)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 К сообщениям", fmt.Sprintf("msg_%s", folder)),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, messageText, keyboard)
}

// handleSelectRecipient обрабатывает выбор получателя для нового сообщения
func (b *Bot) handleSelectRecipient(user *UserState, data string) error {
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return b.SendMessage(user.ChatID, "❌ Ошибка выбора получателя", nil)
	}

	recipientID := parts[2]
	user.TempRecipient = recipientID
	user.State = "message_compose_subject"

	return b.SendMessage(user.ChatID, "✍️ <b>Новое сообщение</b>\n\n📝 Введите тему сообщения:", nil)
}

// startComposeMessage начинает создание сообщения с выбором получателя
func (b *Bot) startComposeMessage(user *UserState) error {
	// Получаем список получателей
	receivers, err := user.Client.GetMessageReceivers()
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения получателей: %v", err), nil)
	}

	text := "✍️ <b>Написать сообщение</b>\n\nВыберите получателя:"
	var keyboard [][]tgbotapi.InlineKeyboardButton
	receiversFound := false

	// Проверяем различные варианты структуры ответа
	result := receivers.Response.Result

	// Вариант 1: receivers в корне result
	if receiversData, ok := result["receivers"]; ok {
		if receiversArray, ok := receiversData.([]interface{}); ok {
			for i, receiverData := range receiversArray {
				if i >= 20 { // Показываем максимум 20 получателей
					break
				}

				if receiver, ok := receiverData.(map[string]interface{}); ok {
					id := fmt.Sprintf("%v", receiver["id"])
					name := fmt.Sprintf("%v", receiver["name"])

					buttonText := fmt.Sprintf("👤 %s", name)
					callbackData := fmt.Sprintf("compose_to_%s", id)

					button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
					keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
					receiversFound = true
				}
			}
		}
	}

	// Вариант 2: получатели могут быть в другом формате
	if !receiversFound {
		// Пробуем найти получателей в других полях result
		for _, value := range result {
			if array, ok := value.([]interface{}); ok && len(array) > 0 {
				// Проверяем первый элемент массива
				if first, ok := array[0].(map[string]interface{}); ok {
					if _, hasID := first["id"]; hasID {
						if _, hasName := first["name"]; hasName {
							// Это похоже на список получателей
							for i, receiverData := range array {
								if i >= 20 {
									break
								}
								if receiver, ok := receiverData.(map[string]interface{}); ok {
									id := fmt.Sprintf("%v", receiver["id"])
									name := fmt.Sprintf("%v", receiver["name"])

									buttonText := fmt.Sprintf("👤 %s", name)
									callbackData := fmt.Sprintf("compose_to_%s", id)

									button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
									keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
									receiversFound = true
								}
							}
							break
						}
					}
				}
			}
		}
	}

	if !receiversFound {
		return b.SendMessage(user.ChatID, "❌ Нет доступных получателей", nil)
	}

	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "messages"),
	})

	return b.SendMessage(user.ChatID, text, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
}

// handleMessageSubject обрабатывает ввод темы сообщения
func (b *Bot) handleMessageSubject(user *UserState, subject string) error {
	user.TempLogin = subject // Временно используем для хранения темы
	user.State = "message_compose_text"
	return b.SendMessage(user.ChatID, "📝 Теперь введите текст сообщения:", nil)
}

// handleMessageText обрабатывает ввод текста сообщения
func (b *Bot) handleMessageText(user *UserState, text string) error {
	subject := user.TempLogin
	recipientID := user.TempRecipient

	// Очищаем временные данные
	user.TempLogin = ""
	user.TempRecipient = ""
	user.State = "idle"

	if recipientID == "" {
		return b.SendMessage(user.ChatID, "❌ Получатель не выбран", nil)
	}

	// Отправляем сообщение выбранному получателю
	recipients := []string{recipientID}

	_, err := user.Client.SendMessage(recipients, subject, text)
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка отправки сообщения: %v", err), nil)
	}

	// Получаем информацию о получателе для отображения
	receivers, err := user.Client.GetMessageReceivers()
	recipientName := recipientID
	if err == nil {
		result := receivers.Response.Result

		// Ищем получателя в списке
		if receiversData, ok := result["receivers"]; ok {
			if receiversArray, ok := receiversData.([]interface{}); ok {
				for _, receiverData := range receiversArray {
					if receiver, ok := receiverData.(map[string]interface{}); ok {
						id := fmt.Sprintf("%v", receiver["id"])
						if id == recipientID {
							recipientName = fmt.Sprintf("%v", receiver["name"])
							break
						}
					}
				}
			}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✍️ Написать еще", "msg_compose"),
			tgbotapi.NewInlineKeyboardButtonData("📥 К сообщениям", "messages"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, fmt.Sprintf("✅ <b>Сообщение отправлено!</b>\n\n👤 Получатель: %s\n📝 Тема: %s", recipientName, subject), keyboard)
}

// handleSchedule обрабатывает просмотр расписания
func (b *Bot) handleSchedule(user *UserState) error {
	if !user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо авторизоваться через /login", nil)
	}

	schedule, err := user.Client.GetSchedule("", "")
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения расписания: %v", err), nil)
	}

	text := "📋 <b>Расписание занятий:</b>\n\n"

	if len(schedule.Response.Result.Students) == 0 {
		text += "<i>Расписание не найдено</i>"
	} else {
		student := schedule.Response.Result.Students[0]
		for _, day := range student.Days {
			// Преобразуем дату в читабьый формат
			dayFormatted := formatDateRu(day.Date)
			text += fmt.Sprintf("📅 <b>%s</b>\n", dayFormatted)

			if len(day.Lessons) == 0 {
				text += "   <i>Занятий нет</i>\n\n"
				continue
			}

			for _, lesson := range day.Lessons {
				text += fmt.Sprintf("   %d. <b>%s</b>\n", lesson.Number, lesson.Name)
				if lesson.Teacher != "" {
					text += fmt.Sprintf("      👨‍🏫 %s\n", lesson.Teacher)
				}
				if lesson.Room != "" {
					text += fmt.Sprintf("      🏫 Кабинет %s\n", lesson.Room)
				}
				if lesson.Time != "" {
					text += fmt.Sprintf("      ⏰ %s\n", lesson.Time)
				}
			}
			text += "\n"
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleMarks обрабатывает просмотр оценок
func (b *Bot) handleMarks(user *UserState) error {
	if !user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "⚠️ Сначала необходимо авторизоваться через /login", nil)
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1️⃣ I четверть", "period_1"),
			tgbotapi.NewInlineKeyboardButtonData("2️⃣ II четверть", "period_2"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("3️⃣ III четверть", "period_3"),
			tgbotapi.NewInlineKeyboardButtonData("4️⃣ IV четверть", "period_4"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 За весь год", "period_year"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, "📊 *Выберите период для просмотра оценок:*", keyboard)
}

// handlePeriodSelect обрабатывает выбор периода для оценок
func (b *Bot) handlePeriodSelect(user *UserState, data string) error {
	var period int
	var periodName string

	switch data {
	case "period_1":
		period = 1
		periodName = "I четверть"
	case "period_2":
		period = 2
		periodName = "II четверть"
	case "period_3":
		period = 3
		periodName = "III четверть"
	case "period_4":
		period = 4
		periodName = "IV четверть"
	case "period_year":
		period = 0 // За весь год
		periodName = "Весь учебный год"
	default:
		return b.SendMessage(user.ChatID, "❌ Неизвестный период", nil)
	}

	marks, err := user.Client.GetMarks(period, "", "")
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения оценок: %v", err), nil)
	}

	return b.formatMarks(user, marks, periodName)
}

// formatMarks форматирует и отправляет оценки
func (b *Bot) formatMarks(user *UserState, marks *eljur.MarksResponse, periodName string) error {
	text := fmt.Sprintf("📊 <b>Оценки - %s:</b>\n\n", periodName)

	if len(marks.Response.Result.Students) == 0 {
		text += "<i>Оценки не найдены</i>"
	} else {
		student := marks.Response.Result.Students[0]

		if len(student.Subjects) == 0 {
			text += "<i>Оценки отсутствуют за выбранный период</i>"
		} else {
			for _, subject := range student.Subjects {
				text += fmt.Sprintf("📚 <b>%s</b>\n", subject.Name)

				if len(subject.Marks) == 0 {
					text += "   <i>Оценок нет</i>\n\n"
				} else {
					text += "   "
					for _, mark := range subject.Marks {
						text += fmt.Sprintf("[%s] ", mark.Value)
					}

					// Вычисляем средний балл (упрощенно)
					if len(subject.Marks) > 0 {
						var sum, count float64
						for _, mark := range subject.Marks {
							if val, err := strconv.ParseFloat(mark.Value, 64); err == nil {
								sum += val
								count++
							}
						}
						if count > 0 {
							avg := sum / count
							text += fmt.Sprintf("\n   📈 Средний балл: %.2f", avg)
						}
					}
					text += "\n\n"
				}
			}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать период", "marks"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGemini обрабатывает главное меню Gemini
func (b *Bot) handleGemini(user *UserState) error {
	var text string
	var keyboard tgbotapi.InlineKeyboardMarkup

	if user.GeminiAPIKey == "" {
		text = "🤖 *Gemini AI Ассистент*\n\n" +
			"⚠️ API ключ не настроен!\n\n" +
			"🔧 Для использования Gemini AI необходимо:\n" +
			"1. Получить API ключ в Google AI Studio\n" +
			"2. Настроить ключ в боте\n" +
			"3. Выбрать модель для работы\n\n" +
			"📱 Затем вы сможете:\n" +
			"• Задавать вопросы по домашнему заданию\n" +
			"• Получать объяснения по темам\n" +
			"• Искать материалы для изучения\n" +
			"• Анализировать учебную информацию"

		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔧 Настроить API", "gemini_setup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📖 Инструкция", "gemini_help"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "start"),
			),
		)
	} else {
		modelName := user.GeminiModel
		if modelName == "" {
			modelName = "gemini-1.5-flash"
		}

		text = "🤖 <b>Gemini AI Ассистент</b>\n\n" +
			fmt.Sprintf("✅ API ключ настроен\n🧠 Модель: %s\n\n", modelName) +
			"Выберите действие:"

		keyboard = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Задать вопрос", "gemini_chat"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("📚 Помощь с ДЗ", "gemini_context_homework"),
				tgbotapi.NewInlineKeyboardButtonData("📖 Объяснить тему", "gemini_context_explain"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔧 Сменить модель", "gemini_model_select"),
				tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "gemini_setup"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Главное меню", "start"),
			),
		)
	}

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGeminiSetup обрабатывает настройку Gemini
func (b *Bot) handleGeminiSetup(user *UserState) error {
	if user.GeminiAPIKey != "" {
		// Если ключ уже есть, показываем меню настроек
		text := "⚙️ <b>Настройки Gemini AI</b>\n\n" +
			fmt.Sprintf("🔑 API ключ: настроен (%s...)\n", user.GeminiAPIKey[:min(8, len(user.GeminiAPIKey))]) +
			fmt.Sprintf("🧠 Модель: %s\n\n", user.GeminiModel) +
			"Выберите действие:"

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔄 Сменить API ключ", "gemini_change_key"),
				tgbotapi.NewInlineKeyboardButtonData("🧠 Сменить модель", "gemini_model_select"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("❌ Удалить настройки", "gemini_reset"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "gemini"),
			),
		)

		return b.SendMessage(user.ChatID, text, keyboard)
	}

	// Показываем инструкцию по получению API ключа
	text := "🔧 <b>Настройка Gemini AI</b>\n\n" +
		"📋 <b>Инструкция по получению API ключа:</b>\n\n" +
		"1️⃣ Перейдите на <a href=\"https://aistudio.google.com/\">Google AI Studio</a>\n" +
		"2️⃣ Войдите в свой Google аккаунт\n" +
		"3️⃣ Нажмите «Get API key» или «Получить API ключ»\n" +
		"4️⃣ Создайте новый API ключ\n" +
		"5️⃣ Скопируйте ключ и вставьте здесь\n\n" +
		"⚠️ <b>Важно:</b> Никому не передавайте свой API ключ!\n\n" +
		"🔑 Введите ваш API ключ:"

	user.State = "gemini_api_setup"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Отмена", "gemini"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGeminiAPISetup обрабатывает ввод API ключа
func (b *Bot) handleGeminiAPISetup(user *UserState, apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)

	if len(apiKey) < 10 {
		return b.SendMessage(user.ChatID, "❌ API ключ слишком короткий. Попробуйте еще раз:", nil)
	}

	// Проверяем валидность ключа
	testClient := gemini.NewClient(apiKey, "gemini-1.5-flash")
	if err := testClient.ValidateAPIKey(); err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Неверный API ключ: %v\n\nПопробуйте еще раз:", err), nil)
	}

	user.GeminiAPIKey = apiKey
	user.GeminiModel = "gemini-1.5-flash" // Модель по умолчанию
	user.State = "idle"

	text := "✅ <b>API ключ успешно сохранен!</b>\n\n" +
		"🧠 Выбрана модель: gemini-1.5-flash\n\n" +
		"Теперь вы можете использовать Gemini AI для помощи с учебой!"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🤖 Использовать Gemini", "gemini"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGeminiModelSelect показывает выбор модели
func (b *Bot) handleGeminiModelSelect(user *UserState, data string) error {
	if strings.HasPrefix(data, "gemini_model_") && data != "gemini_model_select" {
		// Выбор конкретной модели
		model := strings.TrimPrefix(data, "gemini_model_")
		user.GeminiModel = model

		description := gemini.GetModelDescription(model)
		text := fmt.Sprintf("✅ <b>Модель изменена!</b>\n\n🧠 Выбрана: %s\n%s", model, description)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Попробовать", "gemini_chat"),
				tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "gemini"),
			),
		)

		return b.SendMessage(user.ChatID, text, keyboard)
	}

	// Показ списка моделей
	text := "🧠 <b>Выберите модель Gemini:</b>\n\n"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for _, model := range gemini.GetAvailableModels() {
		description := gemini.GetModelDescription(model)
		current := ""
		if model == user.GeminiModel {
			current = " ✅"
		}

		buttonText := fmt.Sprintf("%s%s", model, current)
		callbackData := fmt.Sprintf("gemini_model_%s", model)

		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})

		text += fmt.Sprintf("%s\n\n", description)
	}

	keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "gemini"),
	})

	return b.SendMessage(user.ChatID, text, tgbotapi.NewInlineKeyboardMarkup(keyboard...))
}

// handleGeminiContextSelect обрабатывает выбор контекста
func (b *Bot) handleGeminiContextSelect(user *UserState, data string) error {
	context := ""
	contextName := ""

	switch data {
	case "gemini_context_homework":
		context = "Ты помощник по домашнему заданию. Помоги найти информацию, объясни сложные темы, предложи ресурсы для изучения."
		contextName = "Помощь с домашним заданием"
	case "gemini_context_explain":
		context = "Ты учитель-объяснитель. Объясни тему простым языком, приведи примеры, дай ссылки на полезные видео и материалы."
		contextName = "Объяснение темы"
	default:
		context = "Ты помощник ученика. Отвечай на вопросы, помогай с учебой."
		contextName = "Общий чат"
	}

	user.GeminiContext = context
	user.State = "gemini_chat"

	text := fmt.Sprintf("🤖 <b>%s</b>\n\n💭 Введите ваш вопрос:", contextName)

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "gemini"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGeminiChatStart начинает чат с Gemini
func (b *Bot) handleGeminiChatStart(user *UserState) error {
	if user.GeminiAPIKey == "" {
		return b.SendMessage(user.ChatID, "❌ Сначала настройте API ключ через /gemini_setup", nil)
	}

	user.State = "gemini_chat"
	user.GeminiContext = "Ты помощник ученика. Отвечай на вопросы, помогай с учебой."

	text := "🤖 <b>Чат с Gemini AI</b>\n\n💭 Задайте ваш вопрос:\n\n" +
		"<i>Примеры:</i>\n" +
		"• Объясни что такое квадратные уравнения\n" +
		"• Найди информацию о Великой Отечественной войне\n" +
		"• Помоги решить задачу по физике\n" +
		"• Дай ссылки на видео по алгебре"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "gemini"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGeminiChat обрабатывает сообщения в чате с Gemini
func (b *Bot) handleGeminiChat(user *UserState, message string) error {
	if user.GeminiAPIKey == "" {
		return b.SendMessage(user.ChatID, "❌ API ключ не настроен. Используйте /gemini_setup", nil)
	}

	// Проверяем валидность сообщения
	if len(strings.TrimSpace(message)) == 0 {
		return b.SendMessage(user.ChatID, "❌ Пожалуйста, введите вопрос", nil)
	}

	// Отправляем сообщение о том, что обрабатываем запрос
	processingMsg := tgbotapi.NewMessage(user.ChatID, "🤔 Gemini думает...")
	sentMsg, _ := b.API.Send(processingMsg)

	// Создаем клиент Gemini
	client := gemini.NewClient(user.GeminiAPIKey, user.GeminiModel)

	// Отправляем сообщение в Gemini
	response, err := client.SendMessage(message, user.GeminiContext)

	// Удаляем сообщение "думает"
	if sentMsg.MessageID != 0 {
		deleteMsg := tgbotapi.NewDeleteMessage(user.ChatID, sentMsg.MessageID)
		b.API.Send(deleteMsg)
	}

	if err != nil {
		user.State = "idle"
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка Gemini: %v\n\nПопробуйте еще раз или проверьте API ключ.", err), nil)
	}

	// Очищаем ответ от потенциально проблемных символов
	response = strings.ReplaceAll(response, "\u0000", "")
	response = strings.ReplaceAll(response, "`", "'")   // Заменяем обратные кавычки
	response = strings.ReplaceAll(response, "*", "\\*") // Экранируем звездочки
	response = strings.ReplaceAll(response, "_", "\\_") // Экранируем подчеркивания
	response = strings.ReplaceAll(response, "[", "\\[") // Экранируем квадратные скобки
	response = strings.ReplaceAll(response, "]", "\\]")
	response = strings.TrimSpace(response)

	if response == "" {
		return b.SendMessage(user.ChatID, "❌ Gemini вернул пустой ответ. Попробуйте переформулировать вопрос.", nil)
	}

	// Проверяем длину ответа и разбиваем на части если нужно
	maxLength := 3900 // Оставляем место для заголовка и кнопок

	if len(response) <= maxLength {
		// Ответ помещается в одно сообщение
		text := fmt.Sprintf("🤖 <b>Gemini AI:</b>\n\n%s", response)

		keyboard := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("💬 Продолжить чат", "gemini_chat"),
			),
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("🔙 Меню Gemini", "gemini"),
				tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
			),
		)

		return b.SendMessage(user.ChatID, text, keyboard)
	} else {
		// Разбиваем ответ на части
		parts := splitMessage(response, maxLength-100) // Оставляем место для заголовка части

		for i, part := range parts {
			var text string
			var keyboard tgbotapi.InlineKeyboardMarkup

			if i == 0 {
				text = fmt.Sprintf("🤖 <b>Gemini AI</b> (часть %d/%d):\n\n%s", i+1, len(parts), part)
			} else {
				text = fmt.Sprintf("🤖 <b>Продолжение</b> (часть %d/%d):\n\n%s", i+1, len(parts), part)
			}

			// Добавляем кнопки только к последней части
			if i == len(parts)-1 {
				keyboard = tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("💬 Продолжить чат", "gemini_chat"),
					),
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("🔙 Меню Gemini", "gemini"),
						tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
					),
				)
			}

			if err := b.SendMessage(user.ChatID, text, keyboard); err != nil {
				return err
			}

			// Небольшая задержка между сообщениями
			time.Sleep(500 * time.Millisecond)
		}

		return nil
	}
}

// handleGeminiHelp показывает инструкцию по использованию Gemini
func (b *Bot) handleGeminiHelp(user *UserState) error {
	text := "📖 <b>Инструкция по использованию Gemini AI</b>\n\n" +
		"🔧 <b>Настройка:</b>\n" +
		"1. Перейдите на <a href=\"https://aistudio.google.com/\">Google AI Studio</a>\n" +
		"2. Войдите в Google аккаунт\n" +
		"3. Нажмите «Get API key»\n" +
		"4. Создайте новый проект или выберите существующий\n" +
		"5. Создайте API ключ\n" +
		"6. Скопируйте ключ и вставьте в бота\n\n" +
		"🤖 <b>Возможности:</b>\n" +
		"• Помощь с домашним заданием\n" +
		"• Объяснение сложных тем\n" +
		"• Поиск учебных материалов\n" +
		"• Ссылки на обучающие видео\n" +
		"• Решение задач и примеров\n\n" +
		"💡 <b>Примеры вопросов:</b>\n" +
		"• «Объясни теорему Пифагора»\n" +
		"• «Найди видео про квадратные уравнения»\n" +
		"• «Помоги с задачей по химии»\n" +
		"• «Что такое митоз в биологии?»"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔧 Настроить API", "gemini_setup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "gemini"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// handleGeminiReset сбрасывает настройки Gemini
func (b *Bot) handleGeminiReset(user *UserState) error {
	user.GeminiAPIKey = ""
	user.GeminiModel = ""
	user.GeminiContext = ""
	user.State = "idle"

	text := "🗑 <b>Настройки Gemini сброшены</b>\n\n" +
		"Все данные удалены. Для повторного использования необходимо заново настроить API ключ."

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔧 Настроить заново", "gemini_setup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, text, keyboard)
}

// min возвращает минимальное из двух чисел
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
