# ETM Server

Сервер для взаимодействия с Telegram ботом и приложением ETM (Every Trade Matter). Сервер принимает вебхуки от Telegram бота и предоставляет API для отправки сообщений пользователям.

## Функциональность

### Telegram Bot
- Команда `/start` или `/auth` - генерирует уникальный токен для пользователя
- Команда `/revoke` - отзывает старый токен и генерирует новый
- Команда `/help` - показывает инструкцию по использованию

### API Endpoints
- `/webhook` - принимает вебхуки от Telegram бота
- `/message` - принимает POST запросы для отправки сообщений пользователям
- `/` - возвращает "ETM - SERVER" (для проверки работоспособности)

## Требования

- Go 1.21 или выше
- PostgreSQL база данных
- Telegram Bot Token
- SALT для генерации токенов (для обратной совместимости)

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
DATABASE_URL=postgres://user:password@host:port/dbname
PORT=8080
```

Для Heroku переменная `DATABASE_URL` будет автоматически установлена при добавлении Heroku Postgres.

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
    "token": "YOUR_TOKEN_HERE",
    "message": "Текст сообщения"
}
```

Где:
- `token` - токен, полученный от бота (поддерживаются как старые токены формата USER_ID:AUTH_CODE, так и новые случайные токены)
- `message` - текст сообщения для отправки

## Безопасность

- Новые токены генерируются случайным образом и хранятся в PostgreSQL
- Старые токены (формата USER_ID:AUTH_CODE) поддерживаются для обратной совместимости
- Команда `/revoke` позволяет отозвать скомпрометированный токен
- Токены проверяются при каждом запросе к API

## Deploy на Heroku

1. Добавьте PostgreSQL addon:
```bash
heroku addons:create heroku-postgresql:hobby-dev
```

2. Установите переменные окружения:
```bash
heroku config:set TELEGRAM_BOT_TOKEN=your_bot_token
heroku config:set SALT=your_salt_value
```

3. Деплой:
```bash
git push heroku main
```

4. После деплоя необходимо настроить вебхук для Telegram бота:
  ```bash
  curl -F "url=https://your-server-url/webhook" https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook
  ```
  Замените:
  - `your-server-url` на URL вашего сервера
  - `<YOUR_BOT_TOKEN>` на токен вашего бота
