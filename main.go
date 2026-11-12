package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Item struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	if err != nil {
		slog.Error("error in handleHealth", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	slog.Info("headers set")

}

func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var newItem Item

	err := json.NewDecoder(r.Body).Decode(&newItem)
	if err != nil {
		slog.Error("error in handleCreateItem", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	slog.Info("headers set")
	err = json.NewEncoder(w).Encode(newItem)
	if err != nil {
		slog.Error("error in handleCreateItem", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /items", handleCreateItem)

	server := http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		//WriteTimeout: 10*time.Second, deprecated, replaced by http.TimeoutHandler
		IdleTimeout: 15 * time.Second,
		Handler:     http.TimeoutHandler(mux, 10*time.Second, "Unavalible"),
	}

	go func() {
		slog.Info("server starting", "Addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server error", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("error shutting down server", "err", err)
	}
	slog.Info("server stopped")

}
