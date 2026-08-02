package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"gorm.io/gorm"
)

type HealthHandler struct {
	logger *slog.Logger
	db     *gorm.DB
}

type HealthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func NewHealthHandler(logger *slog.Logger, db *gorm.DB) *HealthHandler {
	return &HealthHandler{
		logger: logger,
		db:     db,
	}
}

func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	sqlDB, err := h.db.DB()
	if err != nil {
		h.writeUnavailableResponse(w, r, err)
		return
	}

	if err := sqlDB.PingContext(r.Context()); err != nil {
		h.writeUnavailableResponse(w, r, err)
		return
	}

	response := HealthResponse{
		Status:   "ok",
		Database: "available",
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.ErrorContext(
			r.Context(),
			"failed to encode health check response",
			slog.Any("error", err),
		)
	}
}

func (h *HealthHandler) writeUnavailableResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.ErrorContext(
		r.Context(),
		"database health check failed",
		slog.Any("error", err),
	)

	response := HealthResponse{
		Status:   "unhealthy",
		Database: "unavailable",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		h.logger.ErrorContext(
			r.Context(),
			"failed to encode health check response",
			slog.Any("error", encodeErr),
		)
	}
}
