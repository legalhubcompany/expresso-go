package controllers

import (
	"database/sql"
	"net/http"
	"shollu/config"
	"shollu/database"
	"shollu/models"
	"shollu/utils"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
)

type LoginInput struct {
	PhoneNumber string `json:"phone_number" validate:"required"`
	Password    string `json:"password" validate:"required"`
}

func Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body")
	}

	// Validasi input
	if err := utils.Validate.Struct(req); err != nil {
		return utils.ValidationErrorResponse(c, err)
	}

	// Normalisasi email lowercase
	email := strings.ToLower(req.Email)

	// Cek phone number unik
	var count int
	err := database.DB.QueryRow("SELECT COUNT(*) FROM users WHERE phone_number = ?", req.PhoneNumber).Scan(&count)
	if err != nil {
		return utils.ErrorResponse(c, http.StatusInternalServerError, "Database error")
	}
	if count > 0 {
		return utils.ErrorResponse(c, http.StatusBadRequest, "Phone number already registered")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to hash password")
	}

	// Insert ke database
	_, err = database.DB.Exec(`
		INSERT INTO users (full_name, email, phone_number, gender, password_hash, role)
		VALUES (?, ?, ?, ?, ?, ?)`,
		req.FullName, email, req.PhoneNumber, req.Gender, hashedPassword, req.Role)
	if err != nil {
		return utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to register user")
	}

	return utils.SuccessResponse(c, "User registered successfully", nil)
}

func Login(c *fiber.Ctx) error {
	var input LoginInput
	if err := c.BodyParser(&input); err != nil {
		return utils.ErrorResponse(c, http.StatusBadRequest, "Invalid input")
	}
	if err := utils.Validate.Struct(input); err != nil {
		return utils.ValidationErrorResponse(c, err)
	}

	var user models.User
	err := database.DB.QueryRow(`
		SELECT id, full_name, email, phone_number, gender, password_hash, role, profile_picture
		FROM users
		WHERE phone_number = ?`,
		input.PhoneNumber,
	).Scan(&user.ID, &user.FullName, &user.Email, &user.PhoneNumber, &user.Gender, &user.PasswordHash, &user.Role, &user.ProfilePicture)

	if err == sql.ErrNoRows {
		return utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid phone number or password")
	}
	if err != nil {
		return utils.ErrorResponse(c, http.StatusInternalServerError, "Database error")
	}

	// Cek password
	if !utils.CheckPassword(user.PasswordHash, input.Password) {
		return utils.ErrorResponse(c, http.StatusUnauthorized, "Invalid phone number or password")
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   user.ID,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 72).Unix(),
	})
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return utils.ErrorResponse(c, http.StatusInternalServerError, "Could not generate token")
	}

	return utils.SuccessResponse(c, "Login successful", fiber.Map{
		"token": tokenString,
		"user": fiber.Map{
			"id":              user.ID,
			"full_name":       user.FullName,
			"email":           user.Email,
			"phone_number":    user.PhoneNumber,
			"gender":          user.Gender,
			"role":            user.Role,
			"profile_picture": user.ProfilePicture,
		},
	})
}
