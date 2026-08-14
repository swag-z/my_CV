package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/logistics-doc-ai/internal/config"
	"github.com/logistics-doc-ai/internal/db"
	"github.com/logistics-doc-ai/internal/llm"
	"github.com/logistics-doc-ai/internal/ocr"
	"github.com/logistics-doc-ai/internal/queue"
	"github.com/logistics-doc-ai/internal/repository/postgres"
	"github.com/logistics-doc-ai/internal/service"
	"github.com/logistics-doc-ai/internal/storage"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if cfg.App.LogLevel == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	logger.Info().Str("env", cfg.App.Env).Msg("Starting Worker")

	ctx := context.Background()

	// Connect to database
	database, err := db.New(ctx, cfg.DB.DSN, cfg.DB.MaxConns)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Setup MinIO
	minioClient, err := storage.NewMinIOClient(cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, cfg.MinIO.UseSSL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create MinIO client")
	}
	minioClient.SetBucket(cfg.MinIO.Bucket)

	// Setup OCR client
	var ocrClient ocr.Client
	if cfg.OCR.Mock {
		ocrClient = ocr.NewMockClient()
		logger.Info().Msg("Using mock OCR client")
	} else {
		ocrClient = ocr.NewClient(cfg.OCR.ServiceURL, cfg.OCR.InternalToken, time.Duration(cfg.OCR.TimeoutSeconds)*time.Second)
		logger.Info().Str("url", cfg.OCR.ServiceURL).Msg("Using remote OCR service")
	}

	// Setup LLM client
	var llmClient llm.Client
	if cfg.LLM.Provider == "mock" {
		llmClient = llm.NewMockClient()
		logger.Info().Msg("Using mock LLM client")
	} else {
		llmClient = llm.NewOpenAICompatibleClient(
			cfg.LLM.BaseURL,
			cfg.LLM.APIKey,
			cfg.LLM.Model,
			cfg.LLM.Temperature,
			cfg.LLM.MaxTokens,
			time.Duration(cfg.LLM.TimeoutSeconds)*time.Second,
		)
		logger.Info().Str("provider", cfg.LLM.Provider).Msg("Using OpenAI-compatible LLM")
	}

	// Setup repositories
	docRepo := postgres.NewDocumentRepository(database.Pool)
	extRepo := postgres.NewExtractionRepository(database.Pool)
	auditRepo := postgres.NewAuditRepository(database.Pool)

	// Setup processing service
	processingService := service.NewProcessingService(
		docRepo,
		extRepo,
		auditRepo,
		minioClient,
		ocrClient,
		llmClient,
		cfg.MinIO.Bucket,
		cfg.Validation.ConfidenceThreshold,
		cfg.Validation.AutoApprove,
		cfg.App.MaxAttempts,
	)

	// Setup consumer
	consumer := queue.NewConsumer(rdb, queue.StreamName, queue.ConsumerGroup, "worker-1")

	// Number of workers
	numWorkers := cfg.Workers.Count
	if numWorkers <= 0 {
		numWorkers = 2
	}

	logger.Info().Int("workers", numWorkers).Msg("Starting workers")

	// Start workers
	var wg sync.WaitGroup
	workerCtx, cancel := context.WithCancel(ctx)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			logger.Info().Int("worker_id", workerID).Msg("Worker started")
			consumer.Start(workerCtx, processingService.ProcessDocument)
			logger.Info().Int("worker_id", workerID).Msg("Worker stopped")
		}(i)
	}

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down workers...")
	cancel()

	// Wait for all workers to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.Info().Msg("All workers stopped gracefully")
	case <-time.After(30 * time.Second):
		logger.Warn().Msg("Workers shutdown timeout")
	}

	logger.Info().Msg("Worker stopped")
}
