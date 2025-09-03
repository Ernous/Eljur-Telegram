package bot

import (
	"fmt"
	"strconv"
	"strings"

	"school-diary-bot/internal/eljur"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
	)

	welcomeText := "👋 *Добро пожаловать в школьный электронный дневник!*\n\n"
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

		// Преобразуем даты в читаблый формат
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
	// Обрабатываем данные дневника
	var diaryText strings.Builder
	diaryText.WriteString("📚 *Дневник за выбранную неделю:*\n\n")

	// Парсим результат как гибкую структуру
	result := diary.Response.Result
	
	// Проверяем, есть ли студенты в результате
	studentsData, hasStudents := result["students"]
	if !hasStudents {
		// Если нет ключа students, проверяем прямую структуру дат
		hasLessons := false
		for key, value := range result {
			// Проверяем, является ли ключ датой (формат YYYYMMDD)
			if len(key) == 8 {
				if dayData, ok := value.(map[string]interface{}); ok {
					title, _ := dayData["title"].(string)
					if title == "" {
						title = key
					}
					
					diaryText.WriteString(fmt.Sprintf("📅 *%s*\n", title))
					
					itemsData, ok := dayData["items"]
					if !ok {
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
					
					// Простая сортировка по номеру урока
					for i := 1; i <= 10; i++ {
						lessonNum := fmt.Sprintf("%d", i)
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
								
								if room != "" {
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
		
		if !hasLessons {
			diaryText.WriteString("📝 Уроков на этой неделе нет")
		}
	} else {
		// Обработка старого формата с массивом студентов
		students, ok := studentsData.([]interface{})
		if !ok {
			diaryText.WriteString("📝 Ошибка обработки данных дневника")
		} else if len(students) == 0 {
			diaryText.WriteString("📝 Записей в дневнике пока нет")
		} else {
			hasLessons := false
			for _, studentData := range students {
				student, ok := studentData.(map[string]interface{})
				if !ok {
					continue
				}

				daysData, ok := student["days"]
				if !ok {
					continue
				}

				days, ok := daysData.([]interface{})
				if !ok {
					continue
				}

				for _, dayData := range days {
					day, ok := dayData.(map[string]interface{})
					if !ok {
						continue
					}

					date, _ := day["date"].(string)
					diaryText.WriteString(fmt.Sprintf("📅 *%s*\n", date))

					lessonsData, ok := day["lessons"]
					if !ok {
						diaryText.WriteString("   Уроков нет\n\n")
						continue
					}

					lessons, ok := lessonsData.([]interface{})
					if !ok || len(lessons) == 0 {
						diaryText.WriteString("   Уроков нет\n\n")
						continue
					}

					hasLessons = true
					for _, lessonData := range lessons {
						lesson, ok := lessonData.(map[string]interface{})
						if !ok {
							continue
						}

						name, _ := lesson["name"].(string)
						number, _ := lesson["number"].(float64)
						homework, _ := lesson["homework"].(string)

						diaryText.WriteString(fmt.Sprintf("   %.0f. %s", number, name))

						if marksData, ok := lesson["marks"]; ok {
							if marks, ok := marksData.([]interface{}); ok && len(marks) > 0 {
								diaryText.WriteString(" - Оценки: ")
								for i, markData := range marks {
									if mark, ok := markData.(map[string]interface{}); ok {
										if i > 0 {
											diaryText.WriteString(", ")
										}
										if value, ok := mark["value"].(string); ok {
											diaryText.WriteString(value)
										}
									}
								}
							}
						}

						if homework != "" {
							diaryText.WriteString(fmt.Sprintf("\n      📝 ДЗ: %s", homework))
						}

						diaryText.WriteString("\n")
					}
					diaryText.WriteString("\n")
				}
			}

			if !hasLessons {
				diaryText.WriteString("📝 Уроков на этой неделе нет")
			}
		}
	}

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔙 Выбрать другую неделю", "diary"),
			tgbotapi.NewInlineKeyboardButtonData("🏠 Главное меню", "start"),
		),
	)

	return b.SendMessage(user.ChatID, diaryText.String(), keyboard)
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

	text := fmt.Sprintf("💬 *%s сообщения:*\n\nНажмите на сообщение для просмотра:", folderName)
	var keyboard [][]tgbotapi.InlineKeyboardButton

	if len(messages.Response.Result.Messages) == 0 {
		text += "\n\n_Сообщений нет_"
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

			sender := msg.From
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

	return b.SendMessage(user.ChatID, "🗑 *Чат очищен*\n\nВыберите действие:",
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

	// log.Printf("[MESSAGE_DETAILS] Полученные детали: %+v", msgDetails.Response.Result)

	from := msgDetails.Response.Result.From
	subject := msgDetails.Response.Result.Subject
	text := msgDetails.Response.Result.Text
	date := msgDetails.Response.Result.Date
	to := msgDetails.Response.Result.To

	if from == "" && to != "" {
		from = "Вы → " + to
	} else if from == "" {
		from = "Неизвестный отправитель"
	}
	if subject == "" {
		subject = "Без темы"
	}
	if text == "" {
		text = "_Текст сообщения отсутствует_"
	}
	if date == "" {
		date = "_Дата не указана_"
	}

	messageText := fmt.Sprintf("📨 *Детали сообщения:*\n\n"+
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

	return b.SendMessage(user.ChatID, "✍️ *Новое сообщение*\n\n📝 Введите тему сообщения:", nil)
}

// startComposeMessage начинает создание сообщения с выбором получателя
func (b *Bot) startComposeMessage(user *UserState) error {
	// Получаем список получателей
	receivers, err := user.Client.GetMessageReceivers()
	if err != nil {
		return b.SendMessage(user.ChatID, fmt.Sprintf("❌ Ошибка получения получателей: %v", err), nil)
	}

	if len(receivers.Response.Result.Receivers) == 0 {
		return b.SendMessage(user.ChatID, "❌ Нет доступных получателей", nil)
	}

	text := "✍️ *Написать сообщение*\n\nВыберите получателя:"
	var keyboard [][]tgbotapi.InlineKeyboardButton

	for i, receiver := range receivers.Response.Result.Receivers {
		if i >= 20 { // Показываем максимум 20 получателей
			break
		}

		buttonText := fmt.Sprintf("👤 %s", receiver.Name)
		callbackData := fmt.Sprintf("compose_to_%s", receiver.ID)

		button := tgbotapi.NewInlineKeyboardButtonData(buttonText, callbackData)
		keyboard = append(keyboard, []tgbotapi.InlineKeyboardButton{button})
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
		for _, receiver := range receivers.Response.Result.Receivers {
			if receiver.ID == recipientID {
				recipientName = receiver.Name
				break
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

	return b.SendMessage(user.ChatID, fmt.Sprintf("✅ **Сообщение отправлено!**\n\n👤 Получатель: %s\n📝 Тема: %s", recipientName, subject), keyboard)
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

	text := "📋 *Расписание занятий:*\n\n"

	if len(schedule.Response.Result.Students) == 0 {
		text += "_Расписание не найдено_"
	} else {
		student := schedule.Response.Result.Students[0]
		for _, day := range student.Days {
			// Преобразуем дату в читаемый формат
			dayFormatted := formatDateRu(day.Date)
			text += fmt.Sprintf("📅 *%s*\n", dayFormatted)

			if len(day.Lessons) == 0 {
				text += "   _Занятий нет_\n\n"
				continue
			}

			for _, lesson := range day.Lessons {
				text += fmt.Sprintf("   %d. *%s*\n", lesson.Number, lesson.Name)
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
	text := fmt.Sprintf("📊 *Оценки - %s:*\n\n", periodName)

	if len(marks.Response.Result.Students) == 0 {
		text += "_Оценки не найдены_"
	} else {
		student := marks.Response.Result.Students[0]

		if len(student.Subjects) == 0 {
			text += "_Оценки отсутствуют за выбранный период_"
		} else {
			for _, subject := range student.Subjects {
				text += fmt.Sprintf("📚 *%s*\n", subject.Name)

				if len(subject.Marks) == 0 {
					text += "   _Оценок нет_\n\n"
				} else {
					text += "   "
					for _, mark := range subject.Marks {
						text += fmt.Sprintf("`%s` ", mark.Value)
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
							text += fmt.Sprintf("\n   📈 Средний балл: `%.2f`", avg)
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