package controllers

import (
	"database/sql"
	"fmt"
	"shollu/database"
	"shollu/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	midtrans "github.com/veritrans/go-midtrans"
)

// CreatePayment untuk generate Snap token dan simpan ke tabel payments
func CreatePayment(c *fiber.Ctx) error {
	type Req struct {
		IdTransaksi int64 `json:"id_transaksi" validate:"required"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid input")
	}
	if err := utils.Validate.Struct(req); err != nil {
		return utils.ValidationErrorResponse(c, err)
	}

	// 🔑 Ambil user ID dari JWT
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	// Ambil transaksi dari DB, pastikan milik user tsb
	var total int64
	var nama, phone, email sql.NullString

	err := database.DB.QueryRow(`
		SELECT total, nama FROM transaksi WHERE id = ? AND id_user = ?`, req.IdTransaksi, userID).
		Scan(&total, &nama)
	if err == sql.ErrNoRows {
		return utils.ErrorResponse(c, 404, "Transaksi tidak ditemukan atau bukan milik Anda")
	} else if err != nil {
		return utils.ErrorResponse(c, 500, "Database error")
	}

	// Generate unique order_id
	orderID := fmt.Sprintf("ORD-%d-%d", req.IdTransaksi, time.Now().Unix())

	// Inisialisasi Midtrans client
	midclient := midtrans.NewClient()
	midclient.ServerKey = "SB-Mid-server-hYrHVhezuyR5OsijtGfBGSbQ"
	midclient.ClientKey = "SB-Mid-client-p0ktJt88DsJHmwKQ"
	midclient.APIEnvType = midtrans.Sandbox

	// Inisialisasi Snap Gateway dengan client tersebut
	snapGateway := midtrans.SnapGateway{
		Client: midclient,
	}

	// Buat Snap request
	snapReq := &midtrans.SnapReq{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  orderID,
			GrossAmt: total,
		},
		CustomerDetail: &midtrans.CustDetail{
			FName: nama.String,
			Email: email.String,
			Phone: phone.String,
		},
	}

	// Panggil Midtrans Snap API
	snapResp, err := snapGateway.GetToken(snapReq)
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal membuat transaksi Midtrans: "+err.Error())
	}

	expiredAt := time.Now().Add(2 * time.Hour)

	// Simpan ke tabel payments
	_, err = database.DB.Exec(`
		INSERT INTO payments (order_id, id_transaksi, snap_token, snap_expired_at, status_midtrans, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		orderID, req.IdTransaksi, snapResp.Token, expiredAt, "pending", time.Now(),
	)
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal menyimpan payment ke database")
	}

	return utils.SuccessResponse(c, "Payment berhasil dibuat", fiber.Map{
		"order_id":     orderID,
		"snap_token":   snapResp.Token,
		"redirect_url": snapResp.RedirectURL,
		"expired_at":   expiredAt.Format(time.RFC3339),
		"amount":       total,
	})
}
