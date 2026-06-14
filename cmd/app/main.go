package main

import (
	"log"
	"net/http"

	"github.com/OmNom69/org-structure-api/internal/config"
	"github.com/OmNom69/org-structure-api/internal/database"
	"github.com/OmNom69/org-structure-api/internal/handler"
	"github.com/OmNom69/org-structure-api/internal/repository"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("database connected")

	departmentRepo := repository.NewDepartmentRepository(db)
	employeeRepo := repository.NewEmployeeRepository(db)

	departmentHandler := handler.NewDepartmentHandler(departmentRepo, employeeRepo)
	employeeHandler := handler.NewEmployeeHandler(employeeRepo, departmentRepo)

	router := http.NewServeMux()

	// department
	router.HandleFunc("POST /departments/", departmentHandler.CreateDepartment)
	router.HandleFunc("GET /departments/{id}", departmentHandler.GetDepartment)
	router.HandleFunc("PATCH /departments/{id}", departmentHandler.PatchDepartment)
	router.HandleFunc("DELETE /departments/{id}", departmentHandler.DeleteDepartment)

	// employee
	router.HandleFunc("POST /departments/{id}/employees/", employeeHandler.CreateEmployee)

	addr := ":" + cfg.Port
	log.Println("server started", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}

}
