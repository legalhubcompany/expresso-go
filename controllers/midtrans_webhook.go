package controllers

import (
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
	}
	var cb Callback
	if err := c.BodyParser(&cb); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid input")
	}

	// Update status_midtrans di payments
	_, err := database.DB.Exec(`UPDATE payments SET status_midtrans = ?, updated_at = ? WHERE order_id = ?`,
		cb.TransactionStatus, time.Now(), cb.OrderID)
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal update status payment")
	}

	// Jika sukses bayar, update transaksi juga
	if cb.TransactionStatus == "capture" || cb.TransactionStatus == "settlement" {
		_, err := database.DB.Exec(`UPDATE transaksi SET status = 1 WHERE id = (SELECT id_transaksi FROM payments WHERE order_id = ?)`, cb.OrderID)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal update status transaksi")
		}
	}

	return c.SendStatus(200)
}
