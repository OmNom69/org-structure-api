package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (recorder *responseRecorder) WriteHeader(statusCode int) {
	if recorder.statusCode != 0 {
		return
	}

	recorder.statusCode = statusCode
	recorder.ResponseWriter.WriteHeader(statusCode)
}

func (recorder *responseRecorder) Write(body []byte) (int, error) {
	if recorder.statusCode == 0 {
		recorder.WriteHeader(http.StatusOK)
	}

	return recorder.ResponseWriter.Write(body)
}

func Logging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		startedAt := time.Now()

		recorder := &responseRecorder{
			ResponseWriter: response,
		}

		next.ServeHTTP(recorder, request)

		if recorder.statusCode == 0 {
			recorder.statusCode = http.StatusOK
		}

		duration := time.Since(startedAt)

		logArguments := []any{
			slog.String("method", request.Method),
			slog.String("path", request.URL.Path),
			slog.Int("status", recorder.statusCode),
			slog.String("duration", duration.Round(time.Microsecond).String()),
		}

		switch {
		case recorder.statusCode >= 500:
			logger.ErrorContext(
				request.Context(),
				"http request completed",
				logArguments...,
			)

		case recorder.statusCode >= 400:
			logger.WarnContext(
				request.Context(),
				"http request completed",
				logArguments...,
			)

		default:
			logger.InfoContext(
				request.Context(),
				"http request completed",
				logArguments...,
			)
		}
	})
}
