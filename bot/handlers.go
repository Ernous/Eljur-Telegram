package bot

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"school-diary-bot/bot/eljur"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"school-diary-bot/internal/gemini"
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
	helpText := "🤖 *Школьный электронный дневник*\n\n" +
		"*Доступные команды:*\n" +
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
		"*Как пользоваться:*\n" +
		"1. Авторизуйтесь с помощью /login\n" +
		"2. Используйте команды для просмотра информации\n" +
		"3. Выбирайте недели и периоды для просмотра данных\n\n" +
		"*Пример авторизации:*\n" +
		"Логин: \\`Ivanov\\`\n" +
		"Пароль: \\`password123\\`"

	return b.SendMessage(user.ChatID, helpText, nil)
}

// handleLogin обрабатывает авторизацию
func (b *Bot) handleLogin(user *UserState) error {
	if user.Client.IsAuthenticated() {
		return b.SendMessage(user.ChatID, "✅ Вы уже авторизованы! Используйте /logout для выхода.", nil)
	}

	user.State = "auth_waiting"
	user.AuthStep = 1

	return b.SendMessage(user.ChatID, "🔐 *Авторизация*\n\nВведите ваш логин:\n\n_Пример: Ivanov_", nil)
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

// handleAuthInput обрабатывает ввод данных авторизации
func (b *Bot) handleAuthInput(user *UserState, text string) error {
	switch user.AuthStep {
	case 1: // Логин
		user.TempLogin = strings.TrimSpace(text)
		user.AuthStep = 2
		return b.SendMessage(user.ChatID, "🔑 Теперь введите ваш пароль:\n\n_Пример: password123_", nil)

	case 2: // Пароль
		user.TempPassword = strings.TrimSpace(text)

		// Выполняем авторизацию
		err := user.Client.Authenticate(user.TempLogin, user.TempPassword)

		// Очищаем временные данные
		user.TempLogin = ""
		user.TempPassword = ""
		user.State = "idle"
		user.AuthStep = 0

		if err != nil {
			return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка авторизации: %v\n\nПопробуйте еще раз с помощью /login", err), nil)
		}

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

	text := fmt.Sprintf("📅 *Выберите неделю из %s:*\n\n", period.FullName)

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
	diaryText.WriteString("📚 *Дневник за выбранную неделю:*\n\n")

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

									diaryText.WriteString(fmt.Sprintf("📅 *%s*\n", title))

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
						} else {
							diaryText.WriteString("📝 Ошибка обработки данных студентов")
						}
					}
				}
			}
		}
	}

	if !hasLessons {
		diaryText.WriteString("📝 Уроков на этой неделе нет")
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать другую неделю", "diary"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, diaryText.String(), keyboard)
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
	text := "📅 *Учебные периоды:*\n\n"

	for _, period := range student.Periods {
		status := "✅"
		if period.Disabled {
			status = "⏸"
		}

		text += fmt.Sprintf("%s *%s*\n", status, period.FullName)
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

	return b.SendMessage(user.ChatID, "💬 *Сообщения*\n\nВыберите действие:", keyboard)
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

// 