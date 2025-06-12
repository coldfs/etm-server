# ETM Server

Сервер для взаимодействия с Telegram ботом и приложением ETM (Every Trade Matter). Сервер принимает вебхуки от Telegram бота и предоставляет API для отправки сообщений пользователям.

## Функциональность

### Telegram Bot
- Команда `/start` или `/auth` - генерирует уникальный токен для пользователя
- Команда `/help` - показывает инструкцию по использованию

### API Endpoints
- `/webhook` - принимает вебхуки от Telegram бота
- `/message` - принимает POST запросы для отправки сообщений пользователям
- `/` - возвращает "ETM - SERVER" (для проверки работоспособности)

## Требования

- Go 1.21 или выше
- Telegram Bot Token
- SALT для генерации токенов

## Установка и запуск

1. Клонируйте репозиторий:
```bash
git clone https://github.com/coldfs/etm-server.git
cd etm-server
```

2. Создайте файл `.env` на основе `env.dist`:
```bash
cp .env.dist .env
```

3. Отредактируйте `.env`, указав необходимые переменные окружения:
```
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here
SALT=your_secret_salt_here
PORT=8080
```

4. Запустите сервер:
```bash
go run main.go
```

Сервер будет доступен на порту 8080.

## API

### Отправка сообщения пользователю

```http
POST /message
Content-Type: application/json

{
    "token": "USER_ID:AUTH_CODE",
    "message": "Текст сообщения"
}
```

Где:
- `token` - строка вида USER_ID:AUTH_CODE, полученная от бота
- `message` - текст сообщения для отправки

## Безопасность

- Все токены генерируются с использованием SALT
- Токены проверяются при каждом запросе к API

## Deploy
- Проект можно as-is деплоить на heroku
- После деплоя необходимо настроить вебхук для Telegram бота:
  ```bash
  curl -F "url=https://your-server-url/webhook" https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook
  ```
  Замените:
  - `your-server-url` на URL вашего сервера
  - `<YOUR_BOT_TOKEN>` на токен вашего бота
