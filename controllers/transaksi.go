package controllers

import (
	"database/sql"
	"fmt"
	"shollu/database"
	"shollu/utils"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

// CreateTransaksi dengan harga dari tabel detail_menu
func CreateTransaksi(c *fiber.Ctx) error {
	type MenuReq struct {
		IdMenu    int64 `json:"id_menu" validate:"required"`
		IdVariant int64 `json:"id_variant" validate:"required"`
		Qty       int64 `json:"qty" validate:"required,min=1"`
	}

	type Req struct {
		IdBooth   int64     `json:"id_booth" validate:"required"`   // 🔁 Ganti dari id_outlet
		IdBarista int64     `json:"id_barista" validate:"required"` // ✅ Tambah id_barista
		Nama      string    `json:"nama" validate:"required"`
		Menus     []MenuReq `json:"menus" validate:"required,dive"`
		IdPromo   int64     `json:"id_promo"`
	}

	var req Req
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid input")
	}
	if err := utils.Validate.Struct(req); err != nil {
		return utils.ValidationErrorResponse(c, err)
	}

	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	now := time.Now()
	total := int64(0)

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal mulai transaksi DB")
	}

	// Logging Debug
	fmt.Println("DEBUG total: ", total)
	fmt.Println("DEBUG idUser: ", userID)
	fmt.Println("DEBUG req.IdBooth: ", req.IdBooth)
	fmt.Println("DEBUG req.Nama: ", req.Nama)
	fmt.Println("DEBUG req.IdBarista: ", req.IdBarista)

	// Simpan transaksi awal
	result, err := tx.Exec(`
		INSERT INTO transaksi 
		(id_user, id_barista, id_booth, nama, tanggal, pukul, total, status, id_metode, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, req.IdBarista, req.IdBooth, req.Nama,
		now.Format("2006-01-02"), now.Format("15:04:05"),
		0, 0, 3, now,
	)
	if err != nil {
		tx.Rollback()
		fmt.Println("Insert transaksi error:", err)
		return utils.ErrorResponse(c, 500, "Gagal membuat transaksi")
	}
	idTransaksi, _ := result.LastInsertId()

	stmt, err := tx.Prepare(`
		INSERT INTO transaksi_detail
		(id_transaksi, id_menu, nama_menu, id_variant, nama_variant, harga, qty, subtotal)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return utils.ErrorResponse(c, 500, "Gagal prepare insert detail")
	}
	defer stmt.Close()

	for _, menu := range req.Menus {
		var hargaStr string
		var namaMenu, namaVariant string

		err := tx.QueryRow(`
			SELECT dm.harga, m.nama, v.nama
			FROM detail_menu dm
			JOIN menu m ON dm.id_menu = m.id
			JOIN varian v ON dm.id_varian = v.id
			WHERE dm.id = ?`, menu.IdVariant).
			Scan(&hargaStr, &namaMenu, &namaVariant)

		if err == sql.ErrNoRows {
			tx.Rollback()
			return utils.ErrorResponse(c, 404, "Menu + Variant tidak ditemukan")
		} else if err != nil {
			tx.Rollback()
			return utils.ErrorResponse(c, 500, "Database error (detail_menu)")
		}

		hargaFinal := utils.StringToInt64(hargaStr)
		subtotal := hargaFinal * menu.Qty
		total += subtotal

		_, err = stmt.Exec(
			idTransaksi, menu.IdMenu, namaMenu,
			menu.IdVariant, namaVariant,
			hargaFinal, menu.Qty, subtotal,
		)
		if err != nil {
			tx.Rollback()
			return utils.ErrorResponse(c, 500, "Gagal insert menu detail")
		}
	}

	var potongan int64
	if req.IdPromo != 0 {
		var promoNama string
		var promoPotongan, promoMinimal int64
		var mulai, berakhir time.Time
		var isActive bool

		err := tx.QueryRow(`
			SELECT nama, potongan_nominal, minimal_total, mulai, berakhir, is_active
			FROM promo WHERE id = ?`, req.IdPromo).
			Scan(&promoNama, &promoPotongan, &promoMinimal, &mulai, &berakhir, &isActive)

		if err == sql.ErrNoRows {
			// Promo tidak ditemukan, bisa diabaikan
		} else if err != nil {
			tx.Rollback()
			return utils.ErrorResponse(c, 500, "Database error (promo)")
		} else {
			if isActive && !now.Before(mulai) && !now.After(berakhir) && total >= promoMinimal {
				potongan = promoPotongan
				total -= potongan
			}
		}
	}

	_, err = tx.Exec(`UPDATE transaksi SET total = ? WHERE id = ?`, total, idTransaksi)
	if err != nil {
		tx.Rollback()
		return utils.ErrorResponse(c, 500, "Gagal update total transaksi")
	}

	if err := tx.Commit(); err != nil {
		return utils.ErrorResponse(c, 500, "Gagal commit transaksi DB")
	}

	return utils.SuccessResponse(c, "Transaksi dan detail berhasil dibuat", fiber.Map{
		"id_transaksi": idTransaksi,
		"total":        total,
		"potongan":     potongan,
	})
}
