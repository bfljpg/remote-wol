package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port       string
	DBPath     string
	JWTSecret  string
	AdminUser  string
	AdminPass  string
	AgentURL   string
	AgentToken string
}

func Load() *Config {
	// Try loading .env file
	loadEnvFile(".env")

	return &Config{
		Port:       getEnv("PORT", "3000"),
		DBPath:     getEnv("DB_PATH", "./data/wol.db"),
		JWTSecret:  getEnv("JWT_SECRET", "change-me-to-a-random-secret-at-least-32-chars"),
		AdminUser:  getEnv("ADMIN_USER", "admin"),
		AdminPass:  getEnv("ADMIN_PASS", "admin"),
		AgentURL:   getEnv("AGENT_URL", "http://127.0.0.1:9090"),
		AgentToken: getEnv("AGENT_TOKEN", "change-me-agent-secret"),
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Don't override existing env vars
		if _, ok := os.LookupEnv(key); !ok {
			os.Setenv(key, val)
		}
	}
}
