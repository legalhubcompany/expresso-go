package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"shollu/config"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type WhatsAppMessagePayload struct {
	Entry []struct {
		Changes []struct {
			Value struct {
				Messages []struct {
					From string `json:"from"`
					Text struct {
						Body string `json:"body"`
					} `json:"text"`
				} `json:"messages"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

func WebhookVerify(c *fiber.Ctx) error {
	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == config.VerifyToken {
		return c.SendString(challenge)
	}
	return c.SendStatus(403)
}

func WebhookReceiver(c *fiber.Ctx) error {
	var payload WhatsAppMessagePayload

	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		log.Println("❌ Error parsing body:", err)
		return c.SendStatus(http.StatusBadRequest)
	}

	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			for _, msg := range change.Value.Messages {
				from := msg.From
				body := msg.Text.Body
				log.Println("📨 Incoming from:", from)
				log.Println("💬 Message:", body)

				lines := strings.Split(body, "\n")
				var token string
				for _, line := range lines {
					if strings.HasPrefix(strings.TrimSpace(line), "CODE:") {
						token = strings.TrimSpace(strings.TrimPrefix(line, "CODE:"))
						break
					}
				}

				if token != "" {
					// Call your existing API
					go validateCodeAndReply(token, from)
				}
			}
		}
	}
	return c.SendStatus(200)
}

func validateCodeAndReply(token string, phone string) {
	messageSend := fmt.Sprintf("LOGIN CODE: %s", token)
	reqBody := fmt.Sprintf(`{"message": "%s", "phone_number": "+%s"}`, messageSend, phone)

	resp, err := http.Post(
		"http://localhost:5000/api/auth/v2/login/whatsapp-bot",
		"application/json",
		strings.NewReader(reqBody),
	)

	if err != nil || resp.StatusCode >= 400 {
		log.Println("❌ Login error:", err)
		sendReply(phone, "❌ Kode login tidak valid atau sudah kedaluwarsa.")
		return
	}

	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(bodyBytes, &result)

	link := result["data"].(map[string]interface{})["login_link"].(string)
	sendReply(phone, fmt.Sprintf("✅ Login link: %s", link))
}

func sendReply(to string, text string) {
	url := fmt.Sprintf("https://graph.facebook.com/v19.0/%s/messages", config.PhoneID)
	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}
	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", url, strings.NewReader(string(jsonPayload)))
	// req.Header.Set("Authorization", "Bearer YOUR_ACCESS_TOKEN")
	req.Header.Set("Authorization", "Bearer "+config.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("❌ Gagal kirim balasan:", err)
	}
	defer resp.Body.Close()
}
