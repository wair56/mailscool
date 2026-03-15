package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DBPath            string
	ListenAddr        string
	JWTSecret         string
	MailRetentionDays int
	DefaultAdminUser  string
	DefaultAdminPass  string
	CORSOrigins       []string
	TelegramBotToken  string
}

var C Config

func Init() {
	C = Config{
		DBPath:            getEnv("DB_PATH", "/data/mailer.db"),
		ListenAddr:        getEnv("LISTEN_ADDR", ":8080"),
		JWTSecret:         getEnv("JWT_SECRET", ""),
		MailRetentionDays: getEnvInt("MAIL_RETENTION_DAYS", 7),
		DefaultAdminUser:  getEnv("DEFAULT_ADMIN_USER", "admin"),
		DefaultAdminPass:  getEnv("DEFAULT_ADMIN_PASS", "REDACTED"),
		CORSOrigins:       getEnvList("CORS_ORIGINS", []string{"http://localhost:5173", "http://127.0.0.1:5173"}),
		TelegramBotToken:  getEnv("TELEGRAM_BOT_TOKEN", ""),
	}

	// JWT Secret 延迟到 database.Init() 之后处理，见 InitJWTSecret()
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvList(key string, fallback []string) []string {
	if v := os.Getenv(key); v != "" {
		parts := strings.Split(v, ",")
		var result []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}
