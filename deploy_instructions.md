# Инструкция по деплою Telegram бота на Vercel

## Настройка локальной разработки

1. **Получите токен бота:**
   - Создайте бота через @BotFather в Telegram
   - Получите токен API

2. **Настройте переменные окружения:**
   ```bash
   export TELEGRAM_BOT_TOKEN="ваш_токен_бота"
   export ELJUR_API_URL="https://eljur.gospmr.org/apiv3/"
   export ELJUR_DEV_KEY="dd06cf484d85581e1976d93c639deee7"
   ```
   
   Или скопируйте `.env.example` в `.env` и отредактируйте:
   ```bash
   cp .env.example .env
   # Отредактируйте .env файл
   ```

3. **Запустите локально:**
   ```bash
   go run main.go
   ```

## Деплой на Vercel

### 1. Подготовка проекта

1. Убедитесь что у вас есть аккаунт на Vercel
2. Подключите репозиторий к Vercel

### 2. Настройка переменных окружения в Vercel

В панели Vercel добавьте:
- `TELEGRAM_BOT_TOKEN` = ваш токен бота
- `ELJUR_API_URL` = https://eljur.gospmr.org/apiv3/
- `ELJUR_DEV_KEY` = dd06cf484d85581e1976d93c639deee7

### 3. Настройка webhook

После деплоя настройте webhook для вашего бота:

```bash
curl -X POST "https://api.telegram.org/bot<BOT_TOKEN>/setWebhook" \
     -H "Content-Type: application/json" \
     -d '{"url": "https://ваш-домен.vercel.app/api/webhook"}'
```

### 4. Проверка деплоя

Проверьте что webhook работает:
```bash
curl -X GET "https://api.telegram.org/bot<BOT_TOKEN>/getWebhookInfo"
```

## Структура файлов для Vercel

- `api/webhook.go` - функция обработки webhook'ов (serverless функция)
- `vercel.json` - конфигурация развертывания
- `main.go` - только для локальной разработки

## Важные заметки

1. **main.go** не используется на Vercel - только для локальной разработки
2. **api/webhook.go** - это serverless функция для обработки запросов от Telegram
3. На Vercel бот работает через webhook, а не через polling
4. Переменные окружения должны быть настроены в панели Vercel

## Тестирование

После деплоя:
1. Отправьте `/start` вашему боту
2. Проверьте логи в Vercel Dashboard
3. Протестируйте основные команды