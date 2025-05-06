package utils

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
)

func SuccessResponse(c *fiber.Ctx, message string, data interface{}) error {
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": message,
		"data":    data,
	})
}

func ErrorResponse(c *fiber.Ctx, code int, message string) error {
	return c.Status(code).JSON(fiber.Map{
		"success": false,
		"message": message,
	})
}

func ValidationErrorResponse(c *fiber.Ctx, err error) error {
	errors := FormatValidationErrors(err)
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"success": false,
		"errors":  errors,
	})
}
