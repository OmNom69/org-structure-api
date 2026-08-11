package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OmNom69/org-structure-api/internal/config"
	"github.com/OmNom69/org-structure-api/internal/database"
	"github.com/OmNom69/org-structure-api/internal/handler"
	"github.com/OmNom69/org-structure-api/internal/middleware"
	"github.com/OmNom69/org-structure-api/internal/repository"
	"github.com/OmNom69/org-structure-api/internal/service"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		logger.Error("failed to connect to database", slog.Any("error", err))

		os.Exit(1)
	}

	logger.Info("database connected")

	departmentRepo := repository.NewDepartmentRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)

	employeeService := service.NewEmployeeService(employeeRepo, departmentRepo)
	departmentService := service.NewDepartmentService(departmentRepo, employeeRepo)

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
			logger.Error("server stopped unexpectedly", slog.Any("error", err))

			os.Exit(1)
		}

		return

	case <-shutdownSignal.Done():
		logger.Info("shutdown signal received")
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))

		if closeErr := server.Close(); closeErr != nil {
			logger.Error("failed to force server close", slog.Any("error", closeErr))
		}

		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
