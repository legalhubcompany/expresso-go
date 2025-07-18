package controllers

import (
	"database/sql"
	"fmt"
	"shollu/config"
	"shollu/database"
	"shollu/models"
	"shollu/utils"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

func nullToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// Generate token login sementara tanpa phone number
func WhatsAppLoginRequest(c *fiber.Ctx) error {
	tokenID := utils.GenerateRandomString(32)
	expiresAt := time.Now().Add(5 * time.Minute)

	expoHost := c.Query("expo_host") // Ambil dari query param

	if expoHost != "" {
		// Jika ada expo_host, insert juga ke kolom expo_host
		_, err := database.DB.Exec(`
			INSERT INTO login_tokens (token_id, status, expires_at, expo_host)
			VALUES (?, ?, ?, ?)`,
			tokenID, "pending", expiresAt, expoHost,
		)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal membuat token login")
		}
	} else {
		// Jika tidak ada expo_host, insert tanpa kolom tersebut
		_, err := database.DB.Exec(`
			INSERT INTO login_tokens (token_id, status, expires_at)
			VALUES (?, ?, ?)`,
			tokenID, "pending", expiresAt,
		)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal membuat token login")
		}
	}

	return utils.SuccessResponse(c, "Token login berhasil dibuat", fiber.Map{
		"token_id":   tokenID,
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

// Bot WhatsApp menerima pesan dari user
func WhatsAppBotCallback(c *fiber.Ctx) error {
	type Req struct {
		Message     string `json:"message"`
		PhoneNumber string `json:"phone_number"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Invalid input")
	}

	tokenID := strings.TrimSpace(strings.Replace(req.Message, "LOGIN CODE:", "", 1))
	if tokenID == "" {
		return utils.ErrorResponse(c, 400, "Token tidak ditemukan di pesan")
	}

	var expiresAt time.Time
	var status string
	err := database.DB.QueryRow(`
		SELECT expires_at, status FROM login_tokens WHERE token_id = ?`, tokenID).Scan(&expiresAt, &status)

	if err == sql.ErrNoRows || time.Now().After(expiresAt) || status != "pending" {
		return utils.ErrorResponse(c, 400, "Token tidak valid atau sudah kedaluwarsa")
	}

	var userID string
	isNewUser := false
	err = database.DB.QueryRow(`SELECT id FROM users WHERE username = ?`, req.PhoneNumber).Scan(&userID)

	if err == sql.ErrNoRows {
		_, err := database.DB.Exec(`INSERT INTO users (username, level) VALUES (?, ?)`, req.PhoneNumber, 5)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal membuat user")
		}
		err = database.DB.QueryRow(`SELECT id FROM users WHERE username = ?`, req.PhoneNumber).Scan(&userID)
		if err != nil {
			return utils.ErrorResponse(c, 500, "Gagal mengambil ID user baru")
		}
		isNewUser = true
	} else if err != nil {
		return utils.ErrorResponse(c, 500, "Database error")
	}

	_, _ = database.DB.Exec(`
		UPDATE login_tokens SET status = 'used', phone_number = ? WHERE token_id = ?
	`, req.PhoneNumber, tokenID)

	loginLink := "https://auth.airshare.web.id/gateway/whatsapp-login?token_id=" + tokenID
	// loginLink := "http://ec2-13-229-251-91.ap-southeast-1.compute.amazonaws.com:5000/gateway/whatsapp-login?token_id=" + tokenID

	return utils.SuccessResponse(c, "Token valid. Kirim link ke user via WA", fiber.Map{
		"is_new_user": isNewUser,
		"login_link":  loginLink,
	})
}

// Redirect dari link WA ke app
func WhatsAppGateway(c *fiber.Ctx) error {
	tokenID := c.Query("token_id")
	if tokenID == "" {
		return c.SendString("Token ID tidak ditemukan.")
	}

	var expiresAt time.Time
	var status string
	var expoHost sql.NullString

	err := database.DB.QueryRow(`
		SELECT expires_at, status, expo_host FROM login_tokens WHERE token_id = ?`, tokenID).
		Scan(&expiresAt, &status, &expoHost)

	if err != nil || time.Now().After(expiresAt) || status != "used" {
		return c.SendString("Token sudah tidak berlaku atau belum digunakan.")
	}

	if expoHost.Valid && expoHost.String != "" {
		expoURL := expoHost.String
		if !strings.HasPrefix(expoURL, "exp://") {
			expoURL = "exp://" + expoURL
		}
		return c.Redirect(fmt.Sprintf("%s/--/(auth)/callback?token_id=%s", expoURL, tokenID))
	}
	// expoURL := "u.expo.dev/f4783bf2-1e22-4027-8f67-4c07f2109382/group/622dd2c0-14a1-4dd2-b884-a3d9bdbf5c00"
	// if !strings.HasPrefix(expoURL, "exp://") {
	// 	expoURL = "exp://" + expoURL
	// }
	// return c.Redirect(fmt.Sprintf("%s/--/(auth)/callback?token_id=%s", expoURL, tokenID))

	var expoURL sql.NullString
	err = database.DB.QueryRow(`SELECT url FROM expo_config LIMIT 1`).Scan(&expoURL)
	if err != nil || !expoURL.Valid || expoURL.String == "" {
		return c.SendString("Expo URL tidak tersedia.")
	}

	expoURLStr := expoURL.String
	if !strings.HasPrefix(expoURLStr, "exp://") {
		expoURLStr = "exp://" + expoURLStr
	}
	return c.Redirect(fmt.Sprintf("%s/--/(auth)/callback?token_id=%s", expoURLStr, tokenID))
	// Fallback ke schema app
	// return c.Redirect("expressocoffee://login/callback?token_id=" + tokenID)
}

// Validasi token dari app dan generate JWT final
// func ValidateWhatsAppLoginToken(c *fiber.Ctx) error {
// 	type Req struct {
// 		TokenID string `json:"token_id" validate:"required"`
// 	}
// 	var req Req
// 	if err := c.BodyParser(&req); err != nil {
// 		return utils.ErrorResponse(c, 400, "Token ID tidak ditemukan")
// 	}
// 	if err := utils.Validate.Struct(req); err != nil {
// 		return utils.ValidationErrorResponse(c, err)
// 	}

// 	var phoneNumber string
// 	var status string
// 	var expiresAt time.Time
// 	err := database.DB.QueryRow(`
// 		SELECT phone_number, status, expires_at
// 		FROM login_tokens
// 		WHERE token_id = ?
// 	`, req.TokenID).Scan(&phoneNumber, &status, &expiresAt)

// 	if err == sql.ErrNoRows || time.Now().After(expiresAt) || status != "used" {
// 		return utils.ErrorResponse(c, 401, "Token sudah tidak valid")
// 	}

// 	var (
// 		id, phone string
// 		fullName  sql.NullString
// 	)

// 	err = database.DB.QueryRow(`
// 		SELECT id, name, username
// 		FROM users WHERE username = ?`, phoneNumber).
// 		Scan(&id, &fullName, &phone)
// 	if err != nil {
// 		fmt.Println(err)
// 		return utils.ErrorResponse(c, 401, "User tidak ditemukan")
// 	}

// 	respUser := models.UserResponse{
// 		ID:          id,
// 		FullName:    nullToString(fullName),
// 		PhoneNumber: phone,
// 	}

// 	isNewUser := respUser.FullName == ""

// 	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
// 		"id":   respUser.ID,
// 		"role": respUser.Role,
// 		"exp":  time.Now().Add(time.Hour * 72).Unix(),
// 	})
// 	finalToken, _ := accessToken.SignedString([]byte(config.JWTSecret))

// 	return utils.SuccessResponse(c, "Login sukses", fiber.Map{
// 		"access_token": finalToken,
// 		"is_new_user":  isNewUser,
// 		"user":         respUser,
// 	})
// }

func ValidateWhatsAppLoginToken(c *fiber.Ctx) error {
	type Req struct {
		TokenID string `json:"token_id" validate:"required"`
	}
	var req Req
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, 400, "Token ID tidak ditemukan")
	}
	if err := utils.Validate.Struct(req); err != nil {
		return utils.ValidationErrorResponse(c, err)
	}

	var phoneNumber string
	var status string
	var expiresAt time.Time
	err := database.DB.QueryRow(`
		SELECT phone_number, status, expires_at
		FROM login_tokens
		WHERE token_id = ?
	`, req.TokenID).Scan(&phoneNumber, &status, &expiresAt)

	if err == sql.ErrNoRows || time.Now().After(expiresAt) || status != "used" {
		return utils.ErrorResponse(c, 401, "Token sudah tidak valid")
	}

	var (
		id, phone string
		fullName  sql.NullString
	)

	err = database.DB.QueryRow(`
		SELECT id, name, username
		FROM users WHERE username = ?`, phoneNumber).
		Scan(&id, &fullName, &phone)
	if err != nil {
		fmt.Println(err)
		return utils.ErrorResponse(c, 401, "User tidak ditemukan")
	}

	respUser := models.UserResponse{
		ID:          id,
		FullName:    nullToString(fullName),
		PhoneNumber: phone,
	}

	isNewUser := respUser.FullName == ""

	// JWT standard claims for Laravel (tymon/jwt-auth compatibility)
	now := time.Now()
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss": "http://127.0.0.1:8000/api/login",    // issuer URL sesuai Laravel
		"iat": now.Unix(),                           // issued at
		"nbf": now.Unix(),                           // not before
		"exp": now.Add(time.Hour * 24 * 365).Unix(), // expiration
		"sub": respUser.ID,                          // user ID
		// "jti": "optional-unique-id",                        // optional
	})

	finalToken, _ := accessToken.SignedString([]byte(config.JWTSecret))

	return utils.SuccessResponse(c, "Login sukses", fiber.Map{
		"access_token": finalToken,
		"is_new_user":  isNewUser,
		"user":         respUser,
	})
}

func UpdateProfile(c *fiber.Ctx) error {
	type Req struct {
		Name        string `json:"name" validate:"required"`
		Gender      string `json:"gender"`       // opsional, validasi bisa ditambah kalau perlu
		IDPekerjaan int    `json:"id_pekerjaan"` // opsional, validasi bisa ditambah kalau perlu
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

	result, err := database.DB.Exec(`
		UPDATE users 
		SET name = ?, gender = ?, id_pekerjaan = ? 
		WHERE id = ?
	`, req.Name, req.Gender, req.IDPekerjaan, userID)

	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal update profil")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal membaca hasil update")
	}

	if rowsAffected == 0 {
		return utils.ErrorResponse(c, 404, "User tidak ditemukan")
	}

	return utils.SuccessResponse(c, "Profil berhasil diupdate", fiber.Map{
		"id":           userID,
		"name":         req.Name,
		"gender":       req.Gender,
		"id_pekerjaan": req.IDPekerjaan,
	})
}

func GetMasterPekerjaanList(c *fiber.Ctx) error {
	rows, err := database.DB.Query(`SELECT id, nama FROM master_pekerjaan ORDER BY nama ASC`)
	if err != nil {
		return utils.ErrorResponse(c, 500, "Gagal mengambil data master pekerjaan")
	}
	defer rows.Close()

	var list []fiber.Map
	for rows.Next() {
		var id int
		var nama string
		if err := rows.Scan(&id, &nama); err != nil {
			continue
		}
		list = append(list, fiber.Map{
			"id":   id,
			"nama": nama,
		})
	}

	return utils.SuccessResponse(c, "Berhasil mengambil daftar pekerjaan", list)
}
