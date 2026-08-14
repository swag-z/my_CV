package main

import (
"context"
"fmt"
"log"
"os"
"os/signal"
"syscall"
"time"

"github.com/gofiber/fiber/v2"
"github.com/gofiber/fiber/v2/middleware/cors"
"github.com/gofiber/fiber/v2/middleware/logger"
"github.com/gofiber/fiber/v2/middleware/recover"
"github.com/redis/go-redis/v9"
"github.com/rs/zerolog"

"github.com/logistics-doc-ai/internal/api"
"github.com/logistics-doc-ai/internal/config"
"github.com/logistics-doc-ai/internal/db"
"github.com/logistics-doc-ai/internal/queue"
"github.com/logistics-doc-ai/internal/repository/postgres"
"github.com/logistics-doc-ai/internal/service"
"github.com/logistics-doc-ai/internal/storage"
)

func main() {
// Load config
cfg, err := config.Load()
if err != nil {
log.Fatalf("Failed to load config: %v", err)
}

// Setup logging
zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
zerolog.SetGlobalLevel(zerolog.InfoLevel)
if cfg.App.LogLevel == "debug" {
zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
logger.Info().Str("env", cfg.App.Env).Msg("Starting API server")

// Connect to database
ctx := context.Background()
database, err := db.New(ctx, cfg.DB.DSN, cfg.DB.MaxConns)
if err != nil {
logger.Fatal().Err(err).Msg("Failed to connect to database")
}
defer database.Close()

// Run migrations
if err := db.RunMigrationsManual(ctx, cfg.DB.DSN); err != nil {
logger.Warn().Err(err).Msg("Migration warning")
}

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

// Ensure bucket exists
if err := minioClient.EnsureBucket(ctx, cfg.MinIO.Bucket); err != nil {
logger.Fatal().Err(err).Msg("Failed to ensure bucket")
}

// Setup queue producer
producer := queue.NewProducer(rdb, queue.StreamName)
if err := producer.EnsureConsumerGroup(ctx, queue.ConsumerGroup); err != nil {
logger.Warn().Err(err).Msg("Consumer group warning")
}

// Setup repositories
docRepo := postgres.NewDocumentRepository(database.Pool)
extRepo := postgres.NewExtractionRepository(database.Pool)

// Setup services
docService := service.NewDocumentService(docRepo, extRepo, minioClient, producer, cfg.MinIO.Bucket)

// Setup handlers
handlers := api.NewHandlers(docService)

// Setup Fiber app
app := fiber.New(fiber.Config{
AppName:      "Logistics Doc AI API",
ReadTimeout:  time.Second * 30,
WriteTimeout: time.Second * 30,
})

// Middleware
app.Use(logger.New())
app.Use(recover.New())
app.Use(cors.New())

// Routes
app.Get("/healthz", handlers.HealthCheck)

v1 := app.Group("/api/v1")
v1.Post("/documents", handlers.UploadDocument)
v1.Get("/documents", handlers.ListDocuments)
v1.Get("/documents/:id", handlers.GetDocument)
v1.Patch("/documents/:id/extraction", handlers.UpdateExtraction)
v1.Post("/documents/:id/reprocess", handlers.ReprocessDocument)
v1.Get("/export", handlers.ExportDocument)

// Start server
go func() {
addr := fmt.Sprintf(":%s", cfg.App.Port)
logger.Info().Str("addr", addr).Msg("Server starting")
if err := app.Listen(addr); err != nil {
logger.Fatal().Err(err).Msg("Server failed")
}
}()

// Graceful shutdown
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info().Msg("Shutting down server...")
if err := app.Shutdown(); err != nil {
logger.Error().Err(err).Msg("Shutdown error")
}

logger.Info().Msg("Server stopped")
}
