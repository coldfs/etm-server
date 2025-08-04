package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Структуры для парсинга Telegram webhook
// Минимально необходимые поля

type Update struct {
	Message Message `json:"message"`
}

type Message struct {
	Text string `json:"text"`
	From User   `json:"from"`
	Chat Chat   `json:"chat"`
}

type User struct {
	ID int64 `json:"id"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type MessageRequest struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

var salt string
var db *sql.DB

func initDB() error {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is not set")
	}

	var err error
	db, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return err
	}

	if err = db.Ping(); err != nil {
		return err
	}

	// Создаем таблицу если не существует
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS user_tokens (
		user_id BIGINT PRIMARY KEY,
		token_hash VARCHAR(64) UNIQUE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	
	_, err = db.Exec(createTableQuery)
	return err
}

func generateRandomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateAuthString(userID int64, salt string) string {
	userIDStr := fmt.Sprintf("%d", userID)
	hash := md5.Sum([]byte(userIDStr + salt))
	authCode := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s:%s", userIDStr, authCode)
}

func saveOrUpdateToken(userID int64, token string) error {
	query := `
	INSERT INTO user_tokens (user_id, token_hash) 
	VALUES ($1, $2) 
	ON CONFLICT (user_id) 
	DO UPDATE SET token_hash = EXCLUDED.token_hash, created_at = CURRENT_TIMESTAMP`
	
	_, err := db.Exec(query, userID, token)
	return err
}

func getTokenByUserID(userID int64) (string, error) {
	var token string
	query := `SELECT token_hash FROM user_tokens WHERE user_id = $1`
	err := db.QueryRow(query, userID).Scan(&token)
	return token, err
}

func getUserIDByToken(token string) (int64, error) {
	var userID int64
	query := `SELECT user_id FROM user_tokens WHERE token_hash = $1`
	err := db.QueryRow(query, token).Scan(&userID)
	return userID, err
}

func isOldFormatToken(token string) bool {
	parts := strings.Split(token, ":")
	return len(parts) == 2
}

func migrateOldToken(token string) error {
	parts := strings.Split(token, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid old token format")
	}
	
	var userID int64
	if _, err := fmt.Sscanf(parts[0], "%d", &userID); err != nil {
		return err
	}
	
	// Проверяем валидность старого токена
	expectedToken := generateAuthString(userID, salt)
	if token != expectedToken {
		return fmt.Errorf("invalid old token")
	}
	
	// Сохраняем старый токен в БД
	return saveOrUpdateToken(userID, token)
}

func formatAuthCode(authString string) string {
	return fmt.Sprintf("Ваш бот-токен:\n```\n%s\n```\n\nВставьте его в настройки приложения ETM", authString)
}

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Printf("Received webhook: %s", string(body))

	var update Update
	err = json.Unmarshal(body, &update)
	if err != nil {
		log.Printf("Error parsing update: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch update.Message.Text {
	case "/start", "/auth":
		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		
		// Проверяем, есть ли уже токен у пользователя
		existingToken, err := getTokenByUserID(userID)
		if err == sql.ErrNoRows {
			// Новый пользователь - генерируем случайный токен
			newToken, err := generateRandomToken()
			if err != nil {
				log.Printf("Error generating token: %v", err)
				go sendTelegramMessage(chatID, "Произошла ошибка при генерации токена. Попробуйте позже.")
				break
			}
			
			if err := saveOrUpdateToken(userID, newToken); err != nil {
				log.Printf("Error saving token: %v", err)
				go sendTelegramMessage(chatID, "Произошла ошибка при сохранении токена. Попробуйте позже.")
				break
			}
			
			responseText := formatAuthCode(newToken)
			go sendTelegramMessage(chatID, responseText)
		} else if err != nil {
			log.Printf("Error getting token: %v", err)
			go sendTelegramMessage(chatID, "Произошла ошибка. Попробуйте позже.")
		} else {
			// Пользователь уже есть - возвращаем существующий токен
			responseText := formatAuthCode(existingToken)
			go sendTelegramMessage(chatID, responseText)
		}
		
	case "/revoke":
		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		
		// Генерируем новый токен
		newToken, err := generateRandomToken()
		if err != nil {
			log.Printf("Error generating token: %v", err)
			go sendTelegramMessage(chatID, "Произошла ошибка при генерации нового токена. Попробуйте позже.")
			break
		}
		
		// Сохраняем новый токен
		if err := saveOrUpdateToken(userID, newToken); err != nil {
			log.Printf("Error saving new token: %v", err)
			go sendTelegramMessage(chatID, "Произошла ошибка при сохранении нового токена. Попробуйте позже.")
			break
		}
		
		responseText := fmt.Sprintf("Старый токен отозван.\n\n%s", formatAuthCode(newToken))
		go sendTelegramMessage(chatID, responseText)
		
	case "/help":
		chatID := update.Message.Chat.ID
		helpText := `Доступные команды:
/auth - получить ваш бот-токен
/revoke - отозвать старый токен и получить новый
/help - показать эту справку

Токен нужно ввести в настройках приложения ETM`
		go sendTelegramMessage(chatID, helpText)
	}

	w.WriteHeader(http.StatusOK)
}

func sendTelegramMessage(chatID int64, text string) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error sending message: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("Telegram API error: %s", string(respBody))
	}
}

func messageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req MessageRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var userID int64

	// Проверяем формат токена
	if isOldFormatToken(req.Token) {
		// Старый формат USER_ID:AUTH_CODE
		parts := strings.Split(req.Token, ":")
		_, err = fmt.Sscanf(parts[0], "%d", &userID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid user ID in token"))
			return
		}
		
		// Проверяем валидность старого токена
		expectedToken := generateAuthString(userID, salt)
		if req.Token != expectedToken {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Invalid token"))
			return
		}
		
		// Мигрируем старый токен в БД для последующего использования
		if err := migrateOldToken(req.Token); err != nil {
			log.Printf("Error migrating old token: %v", err)
		}
	} else {
		// Новый формат - случайный токен
		userID, err = getUserIDByToken(req.Token)
		if err != nil {
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte("Invalid token"))
			} else {
				log.Printf("Error checking token: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal server error"))
			}
			return
		}
	}

	// Отправляем сообщение пользователю
	go sendTelegramMessage(userID, req.Message)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Write([]byte("ETM - SERVER"))
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env")
	}

	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN is not set")
	}

	salt = os.Getenv("SALT")
	if salt == "" {
		log.Fatal("SALT is not set")
	}

	// Инициализируем подключение к БД
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database connected successfully")

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/message", messageHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server started on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
