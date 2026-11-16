package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

var repository *ItemRepository

type Item struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Price     int       `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("error in handleHealth", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("headers set")

}

func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var newItem Item
	if err := json.NewDecoder(r.Body).Decode(&newItem); err != nil {
		slog.Error("error in handleCreateItem while decoding request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	item, err := repository.CreateItem(ctx, newItem.Name, newItem.Price)
	if err != nil {
		slog.Error("failed to create item", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	slog.Info("headers set")
	//TODO подумать над кодированием json предварительно, а затем если ошибок нет вызывать w.WriteHeader и писать в буфер w
	if err := json.NewEncoder(w).Encode(item); err != nil {
		slog.Error("error in handleCreateItem while encoding response", "err", err)
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func handleGetItems(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	itemSlice, err := repository.List(ctx)
	if err != nil {
		slog.Error("failed to get items", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	slog.Info("headers set")

	if err := json.NewEncoder(w).Encode(itemSlice); err != nil {
		slog.Error("error in handleGetItems while encoding response", "err", err)
		//TODO подумать над кодированием json предварительно, а затем если ошибок нет вызывать w.WriteHeader и писать в буфер w
		//http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func newDBpool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 15 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create db pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil

}

func main() {
	//logger setup
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	//database conn setup
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("missing required environment variable", slog.String("var", "DATABASE_URL"))
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := newDBpool(ctx, dsn)
	if err != nil {
		slog.Error("database initialization failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()
	repository = NewItemRepository(pool)

	//server setup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", handleHealth)
	mux.HandleFunc("POST /items", handleCreateItem)
	mux.HandleFunc("GET /items", handleGetItems)

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
