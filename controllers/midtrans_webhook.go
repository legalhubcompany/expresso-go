package controllers

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"net/http"
	"shollu/database"
	"shollu/utils"
	"time"

	"github.com/gofiber/fiber/v2"
)

func MidtransWebhookCallback(c *fiber.Ctx) error {
	type Callback struct {
		TransactionStatus string `json:"transaction_status"`
		OrderID           string `json:"order_id"`
		PaymentType       string `json:"payment_type"`
		FraudStatus       string `json:"fraud_status"`
		SignatureKey      string `json:"signature_key"`
		StatusCode        string `json:"status_code"`
		GrossAmount       string `json:"gross_amount"`
	}

	var cb Callback
	if err := c.BodyParser(&cb); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid input")
	}

	// 🔐 Verifikasi signature
	// rawSignature := cb.OrderID + cb.StatusCode + cb.GrossAmount + config.MidtransServerKey // ganti dengan variable server key di config mu
	rawSignature := cb.OrderID + cb.StatusCode + cb.GrossAmount + "SB-Mid-server-hYrHVhezuyR5OsijtGfBGSbQ"
	hash := sha512.New()
	hash.Write([]byte(rawSignature))
	expectedSignature := hex.EncodeToString(hash.Sum(nil))

	if cb.SignatureKey != expectedSignature {
		return utils.ErrorResponse(c, 403, "Invalid signature")
	}

	// ✅ Update status_midtrans di payments
	_, err := database.DB.Exec(`UPDATE payments SET status_midtrans = ?, updated_at = ? WHERE order_id = ?`,
		cb.TransactionStatus, time.Now(), cb.OrderID)
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal update status payment")
	}

	// 🔄 Jika sukses bayar, update transaksi juga
	if cb.TransactionStatus == "capture" || cb.TransactionStatus == "settlement" {
		_, err := database.DB.Exec(`UPDATE transaksi SET status = 1 WHERE id = (SELECT id_transaksi FROM payments WHERE order_id = ?)`, cb.OrderID)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal update status transaksi")
		}

		// ✅ Ambil id_transaksi dari payments
		var idTransaksi string
		err = database.DB.QueryRow(`SELECT id_transaksi FROM payments WHERE order_id = ?`, cb.OrderID).Scan(&idTransaksi)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal mengambil id_transaksi")
		}

		// 🚀 Kirim ke API eksternal
		go func(id string) {
			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			_ = w.WriteField("id_transaksi", id)
			w.Close()

			req, err := http.NewRequest("POST", "https://expressoexpress.bestariagrosolusi.id/api/transaksi/point/update", &b)
			if err != nil {
				fmt.Println("Gagal membuat request:", err)
				return
			}
			req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2V4cHJlc3NvZXhwcmVzcy5iZXN0YXJpYWdyb3NvbHVzaS5pZC9hcGkvbG9naW4iLCJpYXQiOjE3NTE2MDE0MTksIm5iZiI6MTc1MTYwMTQxOSwianRpIjoiMXpoeWhWZTU2WWNwakxQMyIsInN1YiI6IjEiLCJwcnYiOiIyM2JkNWM4OTQ5ZjYwMGFkYjM5ZTcwMWM0MDA4NzJkYjdhNTk3NmY3In0.AKwTDI1nlR2Daay3_LZlFqSPYKDo8qIRKqnhO3xrhyw")
			req.Header.Set("Content-Type", w.FormDataContentType())

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println("Gagal kirim request:", err)
				return
			}
			defer resp.Body.Close()
			fmt.Println("Status kirim ke API:", resp.Status)
		}(idTransaksi)
	}

	return c.SendStatus(200)
}
