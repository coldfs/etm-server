# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build
```bash
go build -o etm-server
```

### Run server
```bash
go run main.go
# Or if already built:
./etm-server
```

### Dependencies
```bash
go mod download
go mod tidy
```

### Environment setup
```bash
cp .env.dist .env
# Edit .env with required values:
# - TELEGRAM_BOT_TOKEN
# - SALT (for token generation)
# - PORT (default: 8080)
```

### Deploy
```bash
# Build for production
go build -o etm-server

# Set Telegram webhook after deploy
curl -F "url=https://your-server-url/webhook" https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook
```

## Architecture

### Overview
ETM Server is a simple Go HTTP server that acts as a bridge between a Telegram bot and the ETM (Every Trade Matter) application. It handles Telegram webhooks and provides an API for sending messages to users.

### Key Components

1. **main.go** - Single file application containing all logic:
   - HTTP server setup with three endpoints
   - Telegram webhook handler for bot commands
   - Message API for authenticated message sending
   - Token generation and validation using MD5 hash with SALT

2. **Authentication Flow**:
   - User sends `/start` or `/auth` to Telegram bot
   - Server generates token: `USER_ID:MD5(USER_ID + SALT)`
   - User uses this token in ETM app to send messages
   - All API requests validate token against regenerated hash

3. **API Endpoints**:
   - `GET /` - Health check endpoint
   - `POST /webhook` - Receives Telegram bot updates
   - `POST /message` - Authenticated endpoint for sending messages to users

### Environment Variables
- `TELEGRAM_BOT_TOKEN` - Required for Telegram API
- `SALT` - Required for secure token generation
- `PORT` - Server port (default: 8080)

### Deployment
Configured for Heroku deployment with Procfile. The binary `etm-server` must be built before deployment.