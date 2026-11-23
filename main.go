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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type ItemHandler struct {
	repository *ItemRepository
}

func NewItemHandler(repo *ItemRepository) *ItemHandler {
	return &ItemHandler{
		repository: repo,
	}
}

type Item struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Price     int       `json:"price"`
	CreatedAt time.Time `json:"created_at"`
	Stock     int       `json:"stock"`
}

type itemRequest struct {
	Name  string `json:"name"`
	Price int    `json:"price"`
	Stock int    `json:"stock"`
}

func (h *ItemHandler) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var newItemRequest itemRequest
	if err := json.NewDecoder(r.Body).Decode(&newItemRequest); err != nil {
		slog.Error("error in handleCreateItem while decoding request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(newItemRequest.Name) == "" || newItemRequest.Price <= 0 {
		slog.Error("item name or/and price are not valid in handleCreateItem", slog.Any("itemRequest", newItemRequest))
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	item, err := h.repository.CreateItem(ctx, newItemRequest.Name, newItemRequest.Price, newItemRequest.Stock)
	if err != nil {
		slog.Error("failed to create item", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, item)

	// payload, err := json.Marshal(item)
	// if err != nil {
	// 	slog.Error("error while serilization object in handleGetItems", slog.Any("error", err))
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusCreated)
	// slog.Info("headers set")
	// _, _ = w.Write(payload)
	// //TODO подумать над кодированием json предварительно, а затем если ошибок нет вызывать w.WriteHeader и писать в буфер w
	// if err := json.NewEncoder(w).Encode(item); err != nil {
	// 	slog.Error("error in handleCreateItem while encoding response", "err", err)
	// 	//http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }

}

func (h *ItemHandler) handleGetItems(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	itemSlice, err := h.repository.List(ctx)
	if err != nil {
		slog.Error("failed to get items", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, itemSlice)

	// payload, err := json.Marshal(itemSlice)
	// if err != nil {
	// 	slog.Error("error while serilization object in handleGetItems", slog.Any("error", err))
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// slog.Info("headers set")
	// _, _ = w.Write(payload)

}

func (h *ItemHandler) handleGetByID(w http.ResponseWriter, r *http.Request) {

	reqId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id in handleGetById", slog.Any("error", err))
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}
	if reqId <= 0 {
		slog.Error("passed incorrect id in handleGetByID", slog.Any("id", reqId))
		http.Error(w, "BadRequest", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	item, err := h.repository.GetByID(ctx, reqId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("item not found in handleGetByID", slog.Any("error", err))
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("error while getting item from repo in handleGetByID", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, item)
	// payload, err := json.Marshal(item)
	// if err != nil {
	// 	slog.Error("error while serilization object in handleGetByID", slog.Any("error", err))
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// slog.Info("headers set")
	// _, _ = w.Write(payload)
}

func (h *ItemHandler) handleUpdateByID(w http.ResponseWriter, r *http.Request) {

	var updateItemRequest itemRequest

	reqId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id in handleUpdateByID")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if reqId <= 0 {
		slog.Error("passed incorrect id in handleUpdateByID", slog.Any("id", reqId))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&updateItemRequest); err != nil {
		slog.Error("error in handleUpdateByID while decoding request", "err", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(updateItemRequest.Name) == "" || updateItemRequest.Price <= 0 {
		slog.Error("item name or/and price are not valid in handleUpdateByID", slog.Any("itemRequest", updateItemRequest))
		http.Error(w, "Bad request:", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	updatedItem, err := h.repository.Update(ctx, reqId, updateItemRequest.Name, updateItemRequest.Price)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("item not found handleUpdateByID", slog.Any("error", err))
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("error while updating item in handleUpdateByID", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, updatedItem)

	// payload, err := json.Marshal(updatedItem)
	// if err != nil {
	// 	slog.Error("error while serilization object in handleUpdateByID", slog.Any("error", err))
	// 	http.Error(w, "internal server error", http.StatusInternalServerError)
	// 	return
	// }
	// w.Header().Set("Content-Type", "application/json")
	// w.WriteHeader(http.StatusOK)
	// slog.Info("headers set")
	// _, _ = w.Write(payload)

}

func (h *ItemHandler) handleDeleteByID(w http.ResponseWriter, r *http.Request) {
	reqId, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		slog.Error("failed to parse id in handleDeleteByID", slog.Any("error", err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if reqId <= 0 {
		slog.Error("passed incorrect id in handleDeleteByID", slog.Any("id", reqId))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	err = h.repository.Delete(ctx, reqId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("item not found in handleDeleteByID", slog.Any("error", err))
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		slog.Error("error while getting item from repo in handleDeleteByID", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func writeJSON(w http.ResponseWriter, status int, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal response", slog.Any("error", err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := w.Write(payload); err != nil {
		slog.Error("failed to write response", slog.Any("error", err))
	}
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
	itemHandler := NewItemHandler(NewItemRepository(pool))

	//server setup
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("POST /items", itemHandler.handleCreateItem)
	mux.HandleFunc("GET /items", itemHandler.handleGetItems)
	mux.HandleFunc("GET /items/{id}", itemHandler.handleGetByID)
	mux.HandleFunc("DELETE /items/{id}", itemHandler.handleDeleteByID)
	mux.HandleFunc("PUT /items/{id}", itemHandler.handleUpdateByID)

	server := http.Server{
		Addr:              ":8080",
		ReadHeaderTimeout: 2 * time.Second,
		ReadTimeout:       5 * time.Second,
		//WriteTimeout: 10*time.Second, deprecated, replaced by http.TimeoutHandler
		IdleTimeout: 15 * time.Second,
		Handler:     http.TimeoutHandler(mux, 10*time.Second, "Unavailable"),
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
