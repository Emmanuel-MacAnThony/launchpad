package config

import (
	"log"
	"os"
)

type Config struct {
	DB     DBConfig
	Server ServerConfig
	Notify NotifyConfig
	Crypto CryptoConfig
	Nginx  NginxConfig
}

type NginxConfig struct {
	BaseDir string
}

type CryptoConfig struct {
	EncryptionKey string // hex-encoded 32-byte key
}

type DBConfig struct {
	URL string
}

type ServerConfig struct {
	Port    string
	BaseURL string
}

type NotifyConfig struct {
	SlackWebhookURL string
}

func Load() Config {
	return Config{
		DB: DBConfig{
			URL: mustEnv("DATABASE_URL"),
		},
		Server: ServerConfig{
			Port:    env("PORT", "8080"),
			BaseURL: env("BASE_URL", "http://localhost:8080"),
		},
		Notify: NotifyConfig{
			SlackWebhookURL: env("SLACK_WEBHOOK_URL", ""),
		},
		Crypto: CryptoConfig{
			EncryptionKey: mustEnv("ENCRYPTION_KEY"),
		},
		Nginx: NginxConfig{
			BaseDir: env("NGINX_BASE_DIR", "/etc/launchpad/services"),
		},
	}
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
