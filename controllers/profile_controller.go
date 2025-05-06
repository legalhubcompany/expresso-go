package controllers

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"shollu/database"
	"shollu/models"
	"shollu/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

func UploadProfilePicture(c *fiber.Ctx) error {
	// Ambil data user dari token JWT
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := claims["id"].(string)

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "File tidak ditemukan",
		})
	}

	// Validasi ekstensi
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		return c.Status(400).JSON(fiber.Map{
			"success": false,
			"message": "Hanya file JPG, JPEG, atau PNG yang diizinkan",
		})
	}

	// Buka file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuka file"})
	}
	defer src.Close()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)

	part, err := writer.CreateFormFile("file", file.Filename)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuat form file"})
	}
	_, err = io.Copy(part, src)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menyalin file"})
	}
	writer.Close()

	// URL tujuan upload ke Supabase Storage
	bucket := "legalhub"
	objectPath := fmt.Sprintf("users/%s%s", userID, ext)
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", os.Getenv("SUPABASE_URL"), bucket, objectPath)

	req, err := http.NewRequest("PUT", url, &b)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal membuat request ke Supabase"})
	}

	req.Header.Set("Authorization", "Bearer "+os.Getenv("SUPABASE_API_KEY"))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	fmt.Println(resp)
	if err != nil || resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal upload ke Supabase", "details": string(body)})
	}
	defer resp.Body.Close()

	// body, _ := io.ReadAll(resp.Body)
	// fmt.Printf("Response Status: %d\n", resp.StatusCode)
	// fmt.Printf("Response Body: %s\n", string(body))

	// URL publik gambar
	publicURL := fmt.Sprintf("%s/storage/v1/object/public/%s/%s", os.Getenv("SUPABASE_URL"), bucket, objectPath)

	// Simpan URL ke database
	_, err = database.DB.Exec("UPDATE users SET profile_picture = ?, updated_at = NOW() WHERE id = ?", publicURL, userID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"success": false, "message": "Gagal menyimpan ke database"})
	}

	return c.Status(200).JSON(fiber.Map{
		"success":         true,
		"message":         "Berhasil upload foto profil",
		"profile_picture": publicURL,
	})
}

func GetProfile(c *fiber.Ctx) error {
	// Ambil data user dari token JWT
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := claims["id"].(string)

	var user models.User
	err := database.DB.QueryRow(`
		SELECT id, full_name, email, phone_number, gender, role, profile_picture
		FROM users WHERE id = ?
	`, userID).Scan(
		&user.ID,
		&user.FullName,
		&user.Email,
		&user.PhoneNumber,
		&user.Gender,
		&user.Role,
		&user.ProfilePicture,
	)

	if err == sql.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "User tidak ditemukan"})
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal mengambil data user"})
	}

	return c.JSON(user)
}

type UpdateProfileRequest struct {
	FullName    string `json:"full_name" validate:"required"`
	Email       string `json:"email" validate:"required,email"`
	PhoneNumber string `json:"phone_number" validate:"required"`
}

func UpdateProfile(c *fiber.Ctx) error {
	// Ambil data user dari token JWT
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	userID := claims["id"].(string)

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Input tidak valid"})
	}

	if err := utils.Validate.Struct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": utils.FormatValidationErrors(err),
		})
	}

	// Update database
	_, err := database.DB.Exec(`
		UPDATE users SET full_name = ?, email = ?, phone_number = ?, updated_at = NOW()
		WHERE id = ?
	`, req.FullName, strings.ToLower(req.Email), req.PhoneNumber, userID)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Gagal update profil"})
	}

	return c.JSON(fiber.Map{
		"message": "Profil berhasil diperbarui",
	})
}
