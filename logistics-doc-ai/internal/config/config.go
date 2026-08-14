package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
type Config struct {
	App      AppConfig
	DB       DBConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
	OCR      OCRConfig
	LLM      LLMConfig
	Processing ProcessingConfig
	Security SecurityConfig
}

type AppConfig struct {
	Port     string
	Env      string
	LogLevel string
}

type DBConfig struct {
	DSN         string
	MaxConns    int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type OCRConfig struct {
	ServiceURL     string
	InternalToken  string
	Timeout        time.Duration
	Mock           bool
	MaxFileMB      int64
	MaxPages       int
}

type LLMConfig struct {
	Provider     string
	BaseURL      string
	APIKey       string
	Model        string
	Timeout      time.Duration
	Temperature  float64
	MaxTokens    int
}

type ProcessingConfig struct {
	MaxFileMB         int64
	ConfidenceThreshold float64
	AutoApprove       bool
	MaxProcessAttempts int
	Workers           int
}

type SecurityConfig struct {
	SecretKey string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// App
	cfg.App.Port = getEnv("APP_PORT", "8080")
	cfg.App.Env = getEnv("APP_ENV", "local")
	cfg.App.LogLevel = getEnv("LOG_LEVEL", "info")

	// Database
	cfg.DB.DSN = getEnv("POSTGRES_DSN", "")
	if cfg.DB.DSN == "" {
		return nil, fmt.Errorf("POSTGRES_DSN is required")
	}
	maxConns, err := strconv.Atoi(getEnv("DATABASE_MAX_CONNS", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_MAX_CONNS: %w", err)
	}
	cfg.DB.MaxConns = maxConns

	// Redis
	cfg.Redis.Addr = getEnv("REDIS_ADDR", "localhost:6379")
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	dbNum, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %w", err)
	}
	cfg.Redis.DB = dbNum

	// MinIO
	cfg.MinIO.Endpoint = getEnv("MINIO_ENDPOINT", "localhost:9000")
	cfg.MinIO.AccessKey = getEnv("MINIO_ACCESS_KEY", "minioadmin")
	cfg.MinIO.SecretKey = getEnv("MINIO_SECRET_KEY", "minioadmin")
	cfg.MinIO.Bucket = getEnv("MINIO_BUCKET", "documents")
	cfg.MinIO.UseSSL = getEnv("MINIO_USE_SSL", "false") == "true"

	// OCR
	cfg.OCR.ServiceURL = getEnv("OCR_SERVICE_URL", "http://localhost:8000")
	cfg.OCR.InternalToken = getEnv("OCR_INTERNAL_TOKEN", "dev-internal-token")
	timeoutSec, err := strconv.Atoi(getEnv("OCR_TIMEOUT_SECONDS", "120"))
	if err != nil {
		return nil, fmt.Errorf("invalid OCR_TIMEOUT_SECONDS: %w", err)
	}
	cfg.OCR.Timeout = time.Duration(timeoutSec) * time.Second
	cfg.OCR.Mock = getEnv("OCR_MOCK", "false") == "true"
	maxFileMB, err := strconv.ParseInt(getEnv("OCR_MAX_FILE_MB", "20"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid OCR_MAX_FILE_MB: %w", err)
	}
	cfg.OCR.MaxFileMB = maxFileMB
	maxPages, err := strconv.Atoi(getEnv("OCR_MAX_PAGES", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid OCR_MAX_PAGES: %w", err)
	}
	cfg.OCR.MaxPages = maxPages

	// LLM
	cfg.LLM.Provider = getEnv("LLM_PROVIDER", "mock")
	cfg.LLM.BaseURL = getEnv("LLM_BASE_URL", "")
	cfg.LLM.APIKey = getEnv("LLM_API_KEY", "")
	cfg.LLM.Model = getEnv("LLM_MODEL", "qwen2.5:7b")
	llmTimeoutSec, err := strconv.Atoi(getEnv("LLM_TIMEOUT_SECONDS", "120"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_TIMEOUT_SECONDS: %w", err)
	}
	cfg.LLM.Timeout = time.Duration(llmTimeoutSec) * time.Second
	temp, err := strconv.ParseFloat(getEnv("LLM_TEMPERATURE", "0"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_TEMPERATURE: %w", err)
	}
	cfg.LLM.Temperature = temp
	maxTokens, err := strconv.Atoi(getEnv("LLM_MAX_TOKENS", "3000"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM_MAX_TOKENS: %w", err)
	}
	cfg.LLM.MaxTokens = maxTokens

	// Processing
	maxProcFileMB, err := strconv.ParseInt(getEnv("MAX_FILE_MB", "20"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_FILE_MB: %w", err)
	}
	cfg.Processing.MaxFileMB = maxProcFileMB
	confThreshold, err := strconv.ParseFloat(getEnv("CONFIDENCE_THRESHOLD", "0.80"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid CONFIDENCE_THRESHOLD: %w", err)
	}
	cfg.Processing.ConfidenceThreshold = confThreshold
	cfg.Processing.AutoApprove = getEnv("AUTO_APPROVE", "false") == "true"
	maxAttempts, err := strconv.Atoi(getEnv("MAX_PROCESS_ATTEMPTS", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_PROCESS_ATTEMPTS: %w", err)
	}
	cfg.Processing.MaxProcessAttempts = maxAttempts
	workers, err := strconv.Atoi(getEnv("WORKERS", "2"))
	if err != nil {
		return nil, fmt.Errorf("invalid WORKERS: %w", err)
	}
	cfg.Processing.Workers = workers

	// Security
	cfg.Security.SecretKey = getEnv("SECRET_KEY", "change-me")

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
