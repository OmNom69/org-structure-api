package main

import (
	"log/slog"
	"net/http"
	"os"

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

	router := http.NewServeMux()

	// department
	router.HandleFunc("POST /departments/", departmentHandler.CreateDepartment)
	router.HandleFunc("GET /departments/{id}", departmentHandler.GetDepartment)
	router.HandleFunc("PATCH /departments/{id}", departmentHandler.PatchDepartment)
	router.HandleFunc("DELETE /departments/{id}", departmentHandler.DeleteDepartment)

	// employee
	router.HandleFunc("POST /departments/{id}/employees/", employeeHandler.CreateEmployee)
	router.HandleFunc("PATCH /employees/{id}", employeeHandler.PatchEmployee)
	router.HandleFunc("GET /employees/", employeeHandler.GetEmployees)
	router.HandleFunc("GET /employees/{id}", employeeHandler.GetEmployee)
	router.HandleFunc("DELETE /employees/{id}", employeeHandler.DeleteEmployee)

	addr := ":" + cfg.Port

	handlerWithLogging := middleware.Logging(logger, router)

	logger.Info("server started", slog.String("address", addr))

	if err := http.ListenAndServe(addr, handlerWithLogging); err != nil {
		logger.Error("server stopped", slog.Any("error", err))

		os.Exit(1)
	}
}
