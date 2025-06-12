package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
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

func generateAuthString(userID int64, salt string) string {
	userIDStr := fmt.Sprintf("%d", userID)
	hash := md5.Sum([]byte(userIDStr + salt))
	authCode := hex.EncodeToString(hash[:])
	return fmt.Sprintf("%s:%s", userIDStr, authCode)
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

	if update.Message.Text == "/start" {
		userID := update.Message.From.ID
		chatID := update.Message.Chat.ID
		responseText := generateAuthString(userID, salt)
		go sendTelegramMessage(chatID, responseText)
	}

	w.WriteHeader(http.StatusOK)
}

func sendTelegramMessage(chatID int64, text string) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
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

	// Парсим токен: ожидаем формат USER_ID:AUTH_CODE
	parts := strings.Split(req.Token, ":")
	if len(parts) != 2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid token format"))
		return
	}
	userIDStr := parts[0]
	// authCode := parts[1] // больше не используется

	// Проверяем auth_code
	var userID int64
	_, err = fmt.Sscanf(userIDStr, "%d", &userID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid user ID in token"))
		return
	}
	validToken := generateAuthString(userID, salt)
	if req.Token != validToken {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid token"))
		return
	}

	// Отправляем сообщение пользователю
	go sendTelegramMessage(userID, req.Message)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
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

	http.HandleFunc("/webhook", webhookHandler)
	http.HandleFunc("/message", messageHandler)

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
