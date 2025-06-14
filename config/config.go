package config

import (
	"os"

	"github.com/joho/godotenv"
)

var (
	DBUser      string
	DBPassword  string
	DBName      string
	DBHost      string
	JWTSecret   string
	AccessToken string
	PhoneID     string
	VerifyToken string
)

func LoadConfig() {
	godotenv.Load()
	DBUser = os.Getenv("DB_USER")
	DBPassword = os.Getenv("DB_PASSWORD")
	DBName = os.Getenv("DB_NAME")
	DBHost = os.Getenv("DB_HOST")
	JWTSecret = os.Getenv("JWT_SECRET")
	AccessToken = os.Getenv("WHATSAPP_ACCESS_TOKEN")
	PhoneID = os.Getenv("WHATSAPP_PHONE_ID")
	VerifyToken = os.Getenv("WHATSAPP_VERIFY_TOKEN")
}
