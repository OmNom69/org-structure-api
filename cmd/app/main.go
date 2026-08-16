package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OmNom69/org-structure-api/internal/cache"
	"github.com/OmNom69/org-structure-api/internal/config"
	"github.com/OmNom69/org-structure-api/internal/database"
	"github.com/OmNom69/org-structure-api/internal/handler"
	"github.com/OmNom69/org-structure-api/internal/middleware"
	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := run(logger); err != nil {
		logger.Error("application stopped", slog.Any("error", err))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	logger.Info("database connected")

	redisStore := setupRedis(logger, cfg)
	if redisStore != nil {
		defer func() {
			if err := redisStore.Close(); err != nil {
				logger.Error("failed to close redis client", slog.Any("error", err))
			}
		}()
	}
	cacheStore := cacheStoreForRedis(redisStore, logger)

	departmentRepo := repository.NewDepartmentRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)

	employeeService := service.NewEmployeeService(employeeRepo, departmentRepo, cacheStore)
	departmentService := service.NewDepartmentService(departmentRepo, employeeRepo, cacheStore, cfg.CacheTTL)

	departmentHandler := handler.NewDepartmentHandler(departmentService, logger)
	employeeHandler := handler.NewEmployeeHandler(employeeService, logger)
	healthHandler := handler.NewHealthHandler(logger, db)

	router := http.NewServeMux()

	// department
	router.HandleFunc("POST /departments", departmentHandler.CreateDepartment)
	router.HandleFunc("GET /departments/{id}", departmentHandler.GetDepartment)
	router.HandleFunc("PATCH /departments/{id}", departmentHandler.PatchDepartment)
	router.HandleFunc("DELETE /departments/{id}", departmentHandler.DeleteDepartment)

	// health
	router.HandleFunc("GET /health", healthHandler.Health)

	// employee
	router.HandleFunc("POST /departments/{id}/employees", employeeHandler.CreateEmployee)
	router.HandleFunc("PATCH /employees/{id}", employeeHandler.PatchEmployee)
	router.HandleFunc("GET /employees", employeeHandler.GetEmployees)
	router.HandleFunc("GET /employees/{id}", employeeHandler.GetEmployee)
	router.HandleFunc("DELETE /employees/{id}", employeeHandler.DeleteEmployee)

	addr := ":" + cfg.Port

	handlerWithLogging := middleware.Logging(logger, router)

	server := &http.Server{
		Addr:              addr,
		Handler:           handlerWithLogging,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("starting server", slog.String("addr", addr))
		serverErrors <- server.ListenAndServe()
	}()

	shutdownSignal, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server stopped unexpectedly: %w", err)
		}

		return nil

	case <-shutdownSignal.Done():
		logger.Info("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			logger.Error("failed to force server close", slog.Any("error", closeErr))
		}

		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	logger.Info("server stopped gracefully")

	return nil
}

func setupRedis(logger *slog.Logger, cfg config.Config) *cache.RedisStore {
	if !cfg.RedisEnabled {
		logger.Info("redis disabled")

		return nil
	}

	store := cache.NewRedisStore(cache.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  cfg.RedisDialTimeout,
		ReadTimeout:  cfg.RedisReadTimeout,
		WriteTimeout: cfg.RedisWriteTimeout,
	})

	pingContext, cancel := context.WithTimeout(
		context.Background(),
		cfg.RedisDialTimeout,
	)
	defer cancel()

	if err := store.Ping(pingContext); err != nil {
		logger.Warn(
			"redis unavailable at startup; continuing with cache degraded",
			slog.Any("error", err),
		)

		return store
	}

	logger.Info("redis connected")

	return store
}

func cacheStoreForRedis(redisStore *cache.RedisStore, logger *slog.Logger) service.CacheStore {
	if redisStore == nil {
		return nil
	}

	return cache.NewLoggingStore(redisStore, logger)
}
