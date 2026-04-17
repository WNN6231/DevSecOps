package common

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv          string
	GinMode         string
	HTTPAddr        string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

func LoadConfig() (Config, error) {
	readTimeout, err := getEnvInt("HTTP_READ_TIMEOUT_SEC", 10)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_READ_TIMEOUT_SEC: %w", err)
	}

	writeTimeout, err := getEnvInt("HTTP_WRITE_TIMEOUT_SEC", 10)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_WRITE_TIMEOUT_SEC: %w", err)
	}

	shutdownTimeout, err := getEnvInt("HTTP_SHUTDOWN_TIMEOUT_SEC", 10)
	if err != nil {
		return Config{}, fmt.Errorf("parse HTTP_SHUTDOWN_TIMEOUT_SEC: %w", err)
	}

	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		GinMode:         getEnv("GIN_MODE", "release"),
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ReadTimeout:     time.Duration(readTimeout) * time.Second,
		WriteTimeout:    time.Duration(writeTimeout) * time.Second,
		ShutdownTimeout: time.Duration(shutdownTimeout) * time.Second,
	}, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}

	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}

	return number, nil
}
