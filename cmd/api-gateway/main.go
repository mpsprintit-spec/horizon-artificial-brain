package main

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/project-horizon/horizon-core/internal/config"
	"github.com/project-horizon/horizon-core/internal/logger"
	"github.com/project-horizon/horizon-core/internal/middleware"
)

func main() {
	cfg := config.Load()
	log := logger.New(logger.Options{Environment: cfg.App.Environment, Level: os.Getenv("LOG_LEVEL")})
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": cfg.App.Name, "version": cfg.App.Version})
	})
	handler := middleware.Chain(mux, middleware.Recover(log), middleware.Logging(log), middleware.RequestID, middleware.SecurityHeaders)
	log.Info("api gateway listening", "address", cfg.HTTP.Address)
	if err := http.ListenAndServe(cfg.HTTP.Address, handler); err != nil {
		slog.Error("api gateway stopped", "error", err)
		os.Exit(1)
	}
}
