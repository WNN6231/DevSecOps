package common

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type DatabaseConfig struct {
	Host               string
	Port               int
	User               string
	Password           string
	Name               string
	Charset            string
	ParseTime          bool
	Loc                string
	MaxIdleConns       int
	MaxOpenConns       int
	ConnMaxLifetimeSec int
}

type Config struct {
	AppEnv          string
	GinMode         string
	HTTPAddr        string
	LogLevel        string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	Database        DatabaseConfig
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

	dbPort, err := getEnvInt("DB_PORT", 3306)
	if err != nil {
		return Config{}, fmt.Errorf("parse DB_PORT: %w", err)
	}

	dbMaxIdleConns, err := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return Config{}, fmt.Errorf("parse DB_MAX_IDLE_CONNS: %w", err)
	}

	dbMaxOpenConns, err := getEnvInt("DB_MAX_OPEN_CONNS", 50)
	if err != nil {
		return Config{}, fmt.Errorf("parse DB_MAX_OPEN_CONNS: %w", err)
	}

	dbConnMaxLifetimeSec, err := getEnvInt("DB_CONN_MAX_LIFETIME_SEC", 300)
	if err != nil {
		return Config{}, fmt.Errorf("parse DB_CONN_MAX_LIFETIME_SEC: %w", err)
	}

	return Config{
		AppEnv:          getEnv("APP_ENV", "development"),
		GinMode:         getEnv("GIN_MODE", "release"),
		HTTPAddr:        getEnv("HTTP_ADDR", ":8080"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		ReadTimeout:     time.Duration(readTimeout) * time.Second,
		WriteTimeout:    time.Duration(writeTimeout) * time.Second,
		ShutdownTimeout: time.Duration(shutdownTimeout) * time.Second,
		Database: DatabaseConfig{
			Host:               getEnv("DB_HOST", "127.0.0.1"),
			Port:               dbPort,
			User:               getEnv("DB_USER", "dev"),
			Password:           getEnv("DB_PASSWORD", "dev123"),
			Name:               getEnv("DB_NAME", "devsecops"),
			Charset:            getEnv("DB_CHARSET", "utf8mb4"),
			ParseTime:          getEnvBool("DB_PARSE_TIME", true),
			Loc:                getEnv("DB_LOC", "Local"),
			MaxIdleConns:       dbMaxIdleConns,
			MaxOpenConns:       dbMaxOpenConns,
			ConnMaxLifetimeSec: dbConnMaxLifetimeSec,
		},
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

func getEnvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
